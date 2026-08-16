package walks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/doveva/Gulyaem/backend/internal/platform/database"
	domain "github.com/doveva/Gulyaem/backend/internal/walks"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Repository struct{ database *database.Pool }

func New(database *database.Pool) *Repository { return &Repository{database: database} }

func (r *Repository) FindByClientRequest(ctx context.Context, actorID, requestID string) (domain.Aggregate, error) {
	tx, err := r.database.Begin(ctx)
	if err != nil {
		return domain.Aggregate{}, fmt.Errorf("begin idempotent walk read: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	return scanAggregate(tx.QueryRow(ctx, aggregateQuery+` WHERE w.actor_id=$1 AND w.client_request_id=$2`, actorID, requestID))
}

func (r *Repository) Get(ctx context.Context, actorID, walkID string) (domain.Aggregate, error) {
	tx, err := r.database.Begin(ctx)
	if err != nil {
		return domain.Aggregate{}, fmt.Errorf("begin walk read: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	return scanAggregate(tx.QueryRow(ctx, aggregateQuery+` WHERE w.actor_id=$1 AND w.id=$2`, actorID, walkID))
}

func (r *Repository) Create(ctx context.Context, materialization domain.Materialization, walk domain.Walk) (domain.Aggregate, error) {
	tx, err := r.database.Begin(ctx)
	if err != nil {
		return domain.Aggregate{}, fmt.Errorf("begin walk materialization: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockExpectedReadyVersion(ctx, tx, materialization.Route.GeoDataVersionID, materialization.Route.CityID); err != nil {
		return domain.Aggregate{}, err
	}
	waypoints, err := json.Marshal(materialization.Route.Waypoints)
	if err != nil {
		return domain.Aggregate{}, err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO routes (id,actor_id,city_id,geo_data_version_id,profile,waypoints,geometry,normalized_geometry,
			distance_m,estimated_duration_sec,routing_provenance,analysis_provenance,materialization_fingerprint)
		VALUES ($1,$2,$3,$4,$5,$6,ST_SetSRID(ST_GeomFromGeoJSON($7),4326),ST_SetSRID(ST_GeomFromGeoJSON($8),4326),$9,$10,$11,$12,$13)
	`, materialization.Route.ID, materialization.Route.ActorID, materialization.Route.CityID,
		materialization.Route.GeoDataVersionID, materialization.Route.Profile, waypoints,
		string(materialization.Route.Geometry), string(materialization.Route.NormalizedGeometry),
		materialization.Route.DistanceMeters, materialization.Route.EstimatedDurationSeconds,
		materialization.Route.RoutingProvenance, materialization.Route.AnalysisProvenance,
		materialization.Route.MaterializationFingerprint)
	if err != nil {
		return domain.Aggregate{}, mapWriteError("insert route", err)
	}
	if err := insertMatches(ctx, tx, materialization.Route.ID, materialization.Matches); err != nil {
		return domain.Aggregate{}, err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO walks (id,actor_id,city_id,route_id,client_request_id,request_fingerprint,status)
		VALUES ($1,$2,$3,$4,$5,$6,'DRAFT')
	`, walk.ID, walk.ActorID, walk.CityID, materialization.Route.ID, walk.ClientRequestID, walk.RequestFingerprint)
	if err != nil {
		return domain.Aggregate{}, mapWriteError("insert walk", err)
	}
	result, err := scanAggregate(tx.QueryRow(ctx, aggregateQuery+` WHERE w.actor_id=$1 AND w.id=$2`, walk.ActorID, walk.ID))
	if err != nil {
		return domain.Aggregate{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.Aggregate{}, fmt.Errorf("commit walk materialization: %w", err)
	}
	return result, nil
}

func (r *Repository) Transition(ctx context.Context, actorID, walkID string, from, to domain.Status, at time.Time) (domain.Aggregate, error) {
	tx, err := r.database.Begin(ctx)
	if err != nil {
		return domain.Aggregate{}, fmt.Errorf("begin walk transition: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	command, err := tx.Exec(ctx, `
		UPDATE walks SET status=$4::walk_status,
			started_at=CASE WHEN $4::walk_status='ACTIVE' THEN $5 ELSE started_at END,
			finished_at=CASE WHEN $4::walk_status='REVIEW' THEN $5 ELSE finished_at END,
			updated_at=$5
		WHERE id=$2 AND actor_id=$1 AND status=$3::walk_status
	`, actorID, walkID, from, to, at)
	if err != nil {
		return domain.Aggregate{}, fmt.Errorf("transition walk: %w", err)
	}
	if command.RowsAffected() != 1 {
		return domain.Aggregate{}, domain.ErrConcurrentChange
	}
	result, err := scanAggregate(tx.QueryRow(ctx, aggregateQuery+` WHERE w.actor_id=$1 AND w.id=$2`, actorID, walkID))
	if err != nil {
		return domain.Aggregate{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.Aggregate{}, fmt.Errorf("commit walk transition: %w", err)
	}
	return result, nil
}

func (r *Repository) ReplaceRoute(ctx context.Context, actorID, walkID string, allowed []domain.Status, materialization domain.Materialization) (domain.Aggregate, error) {
	tx, err := r.database.Begin(ctx)
	if err != nil {
		return domain.Aggregate{}, fmt.Errorf("begin route correction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var routeID string
	var status domain.Status
	err = tx.QueryRow(ctx, `SELECT route_id,status FROM walks WHERE id=$1 AND actor_id=$2 FOR UPDATE`, walkID, actorID).Scan(&routeID, &status)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Aggregate{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Aggregate{}, fmt.Errorf("lock walk for correction: %w", err)
	}
	if !slices.Contains(allowed, status) {
		return domain.Aggregate{}, domain.ErrRouteNotEditable
	}
	if err := lockExpectedReadyVersion(ctx, tx, materialization.Route.GeoDataVersionID, materialization.Route.CityID); err != nil {
		return domain.Aggregate{}, err
	}
	waypoints, err := json.Marshal(materialization.Route.Waypoints)
	if err != nil {
		return domain.Aggregate{}, err
	}
	command, err := tx.Exec(ctx, `
		UPDATE routes SET geo_data_version_id=$3,profile=$4,waypoints=$5,
			geometry=ST_SetSRID(ST_GeomFromGeoJSON($6),4326),normalized_geometry=ST_SetSRID(ST_GeomFromGeoJSON($7),4326),
			distance_m=$8,estimated_duration_sec=$9,routing_provenance=$10,analysis_provenance=$11,
			materialization_fingerprint=$12,revision=revision+1,updated_at=now()
		WHERE id=$1 AND actor_id=$2 AND finalized_at IS NULL
	`, routeID, actorID, materialization.Route.GeoDataVersionID, materialization.Route.Profile, waypoints,
		string(materialization.Route.Geometry), string(materialization.Route.NormalizedGeometry), materialization.Route.DistanceMeters,
		materialization.Route.EstimatedDurationSeconds, materialization.Route.RoutingProvenance,
		materialization.Route.AnalysisProvenance, materialization.Route.MaterializationFingerprint)
	if err != nil {
		return domain.Aggregate{}, fmt.Errorf("replace route: %w", err)
	}
	if command.RowsAffected() != 1 {
		return domain.Aggregate{}, domain.ErrRouteNotEditable
	}
	if _, err := tx.Exec(ctx, `DELETE FROM route_segment_matches WHERE route_id=$1`, routeID); err != nil {
		return domain.Aggregate{}, fmt.Errorf("clear route matches: %w", err)
	}
	if err := insertMatches(ctx, tx, routeID, materialization.Matches); err != nil {
		return domain.Aggregate{}, err
	}
	result, err := scanAggregate(tx.QueryRow(ctx, aggregateQuery+` WHERE w.actor_id=$1 AND w.id=$2`, actorID, walkID))
	if err != nil {
		return domain.Aggregate{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.Aggregate{}, fmt.Errorf("commit route correction: %w", err)
	}
	return result, nil
}

func insertMatches(ctx context.Context, tx pgx.Tx, routeID string, matches []domain.SegmentMatch) error {
	for _, match := range matches {
		command, err := tx.Exec(ctx, `INSERT INTO route_segment_matches
			(route_id,street_segment_id,classification,matched_length_m,covered_length_m,direct_length_m,required_length_m,coverage_status,provenance,confidence)
			SELECT r.id,ss.id,$3,$4,$5,$6,$7,$8,$9,$10
			FROM routes r
			JOIN street_segments ss
			  ON ss.id=$2
			 AND ss.geo_data_version_id=r.geo_data_version_id
			WHERE r.id=$1`, routeID, match.SegmentID, match.Classification, match.MatchedMeters,
			match.CoveredMeters, match.DirectMeters, match.RequiredMeters, match.Status, nullableString(match.Provenance), match.Confidence)
		if err != nil {
			return fmt.Errorf("insert route match %s: %w", match.SegmentID, err)
		}
		if command.RowsAffected() != 1 {
			return fmt.Errorf("insert route match %s: %w", match.SegmentID, domain.ErrSegmentGeoVersionMismatch)
		}
	}
	return nil
}

const aggregateQuery = `
	SELECT w.id,w.actor_id,w.city_id,w.route_id,w.client_request_id,w.request_fingerprint,w.status,
		w.started_at,w.finished_at,w.completed_at,w.duration_sec,w.distance_m,w.created_at,w.updated_at,
		r.id,r.actor_id,r.city_id,r.geo_data_version_id,r.profile,r.waypoints,
		ST_AsGeoJSON(r.geometry),ST_AsGeoJSON(r.normalized_geometry),r.distance_m,r.estimated_duration_sec,
		r.routing_provenance,r.analysis_provenance,r.materialization_fingerprint,r.revision,r.finalized_at,r.created_at,r.updated_at
	FROM walks w
	JOIN routes r
	  ON r.id=w.route_id
	 AND r.actor_id=w.actor_id
	 AND r.city_id=w.city_id`

type scanner interface{ Scan(...any) error }

func scanAggregate(row scanner) (domain.Aggregate, error) {
	var result domain.Aggregate
	var waypoints, geometry, normalized, routing, analysis []byte
	err := row.Scan(&result.Walk.ID, &result.Walk.ActorID, &result.Walk.CityID, &result.Walk.RouteID, &result.Walk.ClientRequestID,
		&result.Walk.RequestFingerprint, &result.Walk.Status, &result.Walk.StartedAt, &result.Walk.FinishedAt, &result.Walk.CompletedAt,
		&result.Walk.DurationSeconds, &result.Walk.DistanceMeters, &result.Walk.CreatedAt, &result.Walk.UpdatedAt,
		&result.Route.ID, &result.Route.ActorID, &result.Route.CityID, &result.Route.GeoDataVersionID, &result.Route.Profile, &waypoints,
		&geometry, &normalized, &result.Route.DistanceMeters, &result.Route.EstimatedDurationSeconds, &routing, &analysis,
		&result.Route.MaterializationFingerprint, &result.Route.Revision, &result.Route.FinalizedAt, &result.Route.CreatedAt, &result.Route.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Aggregate{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Aggregate{}, fmt.Errorf("scan walk: %w", err)
	}
	if err := json.Unmarshal(waypoints, &result.Route.Waypoints); err != nil {
		return domain.Aggregate{}, fmt.Errorf("decode route waypoints: %w", err)
	}
	result.Route.Geometry = json.RawMessage(geometry)
	result.Route.NormalizedGeometry = json.RawMessage(normalized)
	result.Route.RoutingProvenance = json.RawMessage(routing)
	result.Route.AnalysisProvenance = json.RawMessage(analysis)
	return result, nil
}

func mapWriteError(operation string, err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return domain.ErrIdempotencyConflict
	}
	return fmt.Errorf("%s: %w", operation, err)
}
func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

// lockExpectedReadyVersion closes the preview→persistence race. A geo
// publisher cannot supersede this version until the materialization or route
// correction transaction commits.
func lockExpectedReadyVersion(ctx context.Context, tx pgx.Tx, versionID, cityID string) error {
	var lockedID string
	err := tx.QueryRow(ctx, `
		SELECT id
		FROM geo_data_versions
		WHERE id=$1 AND city_id=$2 AND status='READY'
		FOR SHARE
	`, versionID, cityID).Scan(&lockedID)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrMaterializationGeoVersionStale
	}
	if err != nil {
		return fmt.Errorf("lock materialization geo version: %w", err)
	}
	return nil
}

var _ domain.Store = (*Repository)(nil)
