package exploration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	domain "github.com/doveva/Gulyaem/backend/internal/exploration"
	"github.com/doveva/Gulyaem/backend/internal/geo/querying"
	"github.com/doveva/Gulyaem/backend/internal/platform/database"
	"github.com/doveva/Gulyaem/backend/internal/walks"
	"github.com/jackc/pgx/v5"
)

type Repository struct{ database *database.Pool }

func New(database *database.Pool) *Repository { return &Repository{database: database} }

type coveredSegment struct {
	id              string
	length, covered float64
	existed         bool
}

func (r *Repository) Complete(ctx context.Context, actorID, walkID string) (walks.CompletionResult, error) {
	tx, err := r.database.Begin(ctx)
	if err != nil {
		return walks.CompletionResult{}, fmt.Errorf("begin exploration completion: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var status walks.Status
	var routeID, cityID, routeVersion string
	var distance float64
	var started, finished *time.Time
	err = tx.QueryRow(ctx, `SELECT w.status,w.route_id,w.city_id,r.geo_data_version_id,r.distance_m,w.started_at,w.finished_at
		FROM walks w
		JOIN routes r ON r.id=w.route_id AND r.actor_id=w.actor_id AND r.city_id=w.city_id
		WHERE w.id=$1 AND w.actor_id=$2 FOR UPDATE OF w,r`, walkID, actorID).
		Scan(&status, &routeID, &cityID, &routeVersion, &distance, &started, &finished)
	if errors.Is(err, pgx.ErrNoRows) {
		return walks.CompletionResult{}, walks.ErrNotFound
	}
	if err != nil {
		return walks.CompletionResult{}, fmt.Errorf("lock walk: %w", err)
	}
	if status == walks.StatusCompleted {
		return loadCompletion(ctx, tx, actorID, walkID)
	}
	if status != walks.StatusReview {
		return walks.CompletionResult{}, walks.ErrInvalidState
	}
	// Serialize exploration mutation per actor/city, including the first Walk
	// where no exploration_states row exists yet.
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, actorID+":"+cityID); err != nil {
		return walks.CompletionResult{}, fmt.Errorf("lock actor exploration: %w", err)
	}
	currentVersion, err := lockCurrentReadyVersion(ctx, tx, cityID)
	if errors.Is(err, pgx.ErrNoRows) {
		return walks.CompletionResult{}, domain.ErrRouteGeoVersionStale
	}
	if err != nil {
		return walks.CompletionResult{}, err
	}
	if currentVersion != routeVersion {
		return walks.CompletionResult{}, domain.ErrRouteGeoVersionStale
	}
	var stateVersion, stateStatus string
	err = tx.QueryRow(ctx, `SELECT geo_data_version_id,status FROM exploration_states WHERE actor_id=$1 AND city_id=$2 FOR UPDATE`, actorID, cityID).Scan(&stateVersion, &stateStatus)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return walks.CompletionResult{}, fmt.Errorf("lock exploration state: %w", err)
	}
	if err == nil && (stateVersion != currentVersion || stateStatus != "READY") {
		return walks.CompletionResult{}, domain.ErrRebuildRequired
	}

	rows, err := tx.Query(ctx, `SELECT ss.id,ss.length_m,rsm.covered_length_m,(usp.street_segment_id IS NOT NULL)
		FROM route_segment_matches rsm JOIN street_segments ss ON ss.id=rsm.street_segment_id
		LEFT JOIN user_street_segment_progress usp ON usp.actor_id=$2 AND usp.street_segment_id=ss.id
		WHERE rsm.route_id=$1 AND rsm.coverage_status='COMPLETED' AND ss.classification='EXPLORE'
		  AND ss.geo_data_version_id=$3 ORDER BY ss.id FOR UPDATE OF ss`, routeID, actorID, currentVersion)
	if err != nil {
		return walks.CompletionResult{}, fmt.Errorf("load completed route coverage: %w", err)
	}
	segments := []coveredSegment{}
	for rows.Next() {
		var s coveredSegment
		if err := rows.Scan(&s.id, &s.length, &s.covered, &s.existed); err != nil {
			rows.Close()
			return walks.CompletionResult{}, err
		}
		segments = append(segments, s)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return walks.CompletionResult{}, err
	}
	rows.Close()
	newCount, revisitedCount := 0, 0
	newLength := 0.0
	for _, s := range segments {
		if s.existed {
			revisitedCount++
		} else {
			newCount++
			newLength += s.length
		}
	}
	_, err = tx.Exec(ctx, `INSERT INTO exploration_deltas(walk_id,actor_id,geo_data_version_id,new_segments_count,revisited_segments_count,new_network_length_m)
		VALUES($1,$2,$3,$4,$5,$6)`, walkID, actorID, currentVersion, newCount, revisitedCount, newLength)
	if err != nil {
		return walks.CompletionResult{}, fmt.Errorf("insert exploration delta: %w", err)
	}
	for _, s := range segments {
		kind := "NEW"
		if s.existed {
			kind = "REVISITED"
		}
		_, err = tx.Exec(ctx, `INSERT INTO exploration_delta_segments(walk_id,street_segment_id,kind,segment_length_m,covered_length_m) VALUES($1,$2,$3,$4,$5)`, walkID, s.id, kind, s.length, s.covered)
		if err != nil {
			return walks.CompletionResult{}, fmt.Errorf("insert segment delta: %w", err)
		}
		_, err = tx.Exec(ctx, `INSERT INTO user_street_segment_progress(actor_id,street_segment_id,first_explored_at,last_explored_at,visit_count,first_walk_id,last_walk_id)
			VALUES($1,$2,$3,$3,1,$4,$4) ON CONFLICT(actor_id,street_segment_id) DO UPDATE SET
			first_walk_id=CASE WHEN EXCLUDED.first_explored_at<user_street_segment_progress.first_explored_at THEN EXCLUDED.first_walk_id ELSE user_street_segment_progress.first_walk_id END,
			first_explored_at=LEAST(user_street_segment_progress.first_explored_at,EXCLUDED.first_explored_at),
			last_walk_id=CASE WHEN EXCLUDED.last_explored_at>=user_street_segment_progress.last_explored_at THEN EXCLUDED.last_walk_id ELSE user_street_segment_progress.last_walk_id END,
			last_explored_at=GREATEST(user_street_segment_progress.last_explored_at,EXCLUDED.last_explored_at),
			visit_count=user_street_segment_progress.visit_count+1`, actorID, s.id, *finished, walkID)
		if err != nil {
			return walks.CompletionResult{}, fmt.Errorf("upsert progress: %w", err)
		}
	}
	if err := insertDistrictDeltas(ctx, tx, actorID, walkID, cityID, currentVersion); err != nil {
		return walks.CompletionResult{}, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO exploration_states(actor_id,city_id,geo_data_version_id,status) VALUES($1,$2,$3,'READY')
		ON CONFLICT(actor_id,city_id) DO UPDATE SET geo_data_version_id=EXCLUDED.geo_data_version_id,status='READY',updated_at=now()`, actorID, cityID, currentVersion)
	if err != nil {
		return walks.CompletionResult{}, fmt.Errorf("publish exploration state: %w", err)
	}
	completedAt := time.Now().UTC()
	duration := int(finished.Sub(*started).Seconds())
	if duration < 0 {
		duration = 0
	}
	_, err = tx.Exec(ctx, `UPDATE routes SET finalized_at=$2,updated_at=$2 WHERE id=$1 AND finalized_at IS NULL`, routeID, completedAt)
	if err != nil {
		return walks.CompletionResult{}, err
	}
	_, err = tx.Exec(ctx, `UPDATE walks SET status='COMPLETED',completed_at=$3,duration_sec=$4,distance_m=$5,updated_at=$3 WHERE id=$1 AND actor_id=$2 AND status='REVIEW'`, walkID, actorID, completedAt, duration, distance)
	if err != nil {
		return walks.CompletionResult{}, err
	}
	result, err := loadCompletion(ctx, tx, actorID, walkID)
	if err != nil {
		return walks.CompletionResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return walks.CompletionResult{}, fmt.Errorf("commit exploration completion: %w", err)
	}
	return result, nil
}

func insertDistrictDeltas(ctx context.Context, tx pgx.Tx, actorID, walkID, cityID, versionID string) error {
	_, err := tx.Exec(ctx, `WITH current_districts AS (
		SELECT d.id,d.district_data_version_id,d.boundary FROM districts d JOIN district_data_versions dv ON dv.id=d.district_data_version_id WHERE d.city_id=$3 AND dv.status='READY'
	), eligible AS (
		SELECT d.id,SUM(ST_Length(ST_CollectionExtract(ST_Intersection(s.geometry,d.boundary),2)::geography)) AS meters FROM current_districts d JOIN street_segments s ON s.geo_data_version_id=$4 AND s.classification='EXPLORE' AND ST_Intersects(s.geometry,d.boundary) GROUP BY d.id
	), explored AS (
		SELECT d.id,SUM(ST_Length(ST_CollectionExtract(ST_Intersection(s.geometry,d.boundary),2)::geography)) AS meters FROM current_districts d JOIN street_segments s ON s.geo_data_version_id=$4 AND s.classification='EXPLORE' AND ST_Intersects(s.geometry,d.boundary) JOIN user_street_segment_progress p ON p.actor_id=$2 AND p.street_segment_id=s.id GROUP BY d.id
	), newly AS (
		SELECT d.id,SUM(ST_Length(ST_CollectionExtract(ST_Intersection(s.geometry,d.boundary),2)::geography)) AS meters FROM current_districts d JOIN exploration_delta_segments eds ON eds.walk_id=$1 AND eds.kind='NEW' JOIN street_segments s ON s.id=eds.street_segment_id AND ST_Intersects(s.geometry,d.boundary) GROUP BY d.id
	)
	INSERT INTO walk_district_deltas(walk_id,district_id,district_data_version_id,geo_data_version_id,eligible_length_m,explored_before_m,explored_after_m,new_length_m,percentage_before,percentage_after)
	SELECT $1,d.id,d.district_data_version_id,$4,e.meters,GREATEST(0,x.meters-n.meters),x.meters,n.meters,GREATEST(0,x.meters-n.meters)/e.meters,x.meters/e.meters
	FROM current_districts d JOIN eligible e ON e.id=d.id JOIN explored x ON x.id=d.id JOIN newly n ON n.id=d.id WHERE e.meters>0`, walkID, actorID, cityID, versionID)
	if err != nil {
		return fmt.Errorf("insert district deltas: %w", err)
	}
	return nil
}

func loadCompletion(ctx context.Context, tx pgx.Tx, actorID, walkID string) (walks.CompletionResult, error) {
	var result walks.CompletionResult
	err := tx.QueryRow(ctx, `SELECT w.id,w.actor_id,w.city_id,w.route_id,w.status,w.started_at,w.finished_at,w.completed_at,w.duration_sec,w.distance_m,w.created_at,w.updated_at,
		ed.geo_data_version_id,ed.new_segments_count,ed.revisited_segments_count,ed.new_network_length_m
		FROM walks w JOIN exploration_deltas ed ON ed.walk_id=w.id WHERE w.id=$1 AND w.actor_id=$2`, walkID, actorID).Scan(&result.Walk.ID, &result.Walk.ActorID, &result.Walk.CityID, &result.Walk.RouteID, &result.Walk.Status, &result.Walk.StartedAt, &result.Walk.FinishedAt, &result.Walk.CompletedAt, &result.Walk.DurationSeconds, &result.Walk.DistanceMeters, &result.Walk.CreatedAt, &result.Walk.UpdatedAt, &result.Exploration.GeoDataVersionID, &result.Exploration.NewSegmentsCount, &result.Exploration.RevisitedSegmentsCount, &result.Exploration.NewNetworkLengthMeters)
	if errors.Is(err, pgx.ErrNoRows) {
		return walks.CompletionResult{}, walks.ErrNotFound
	}
	if err != nil {
		return walks.CompletionResult{}, fmt.Errorf("load completion: %w", err)
	}
	var geo []byte
	err = tx.QueryRow(ctx, `SELECT jsonb_build_object('type','FeatureCollection','features',COALESCE(jsonb_agg(jsonb_build_object('type','Feature','id',s.id,'geometry',ST_AsGeoJSON(s.geometry)::jsonb,'properties',jsonb_build_object('segmentId',s.id)) ORDER BY s.id),'[]'::jsonb)) FROM exploration_delta_segments eds JOIN street_segments s ON s.id=eds.street_segment_id WHERE eds.walk_id=$1 AND eds.kind='NEW'`, walkID).Scan(&geo)
	if err != nil {
		return walks.CompletionResult{}, err
	}
	result.Exploration.NewSegments = json.RawMessage(geo)
	rows, err := tx.Query(ctx, `SELECT wd.district_id,d.name,wd.percentage_before,wd.percentage_after,wd.new_length_m FROM walk_district_deltas wd JOIN districts d ON d.id=wd.district_id WHERE wd.walk_id=$1 ORDER BY d.name`, walkID)
	if err != nil {
		return walks.CompletionResult{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var d walks.DistrictDelta
		if err := rows.Scan(&d.DistrictID, &d.Name, &d.PercentageBefore, &d.PercentageAfter, &d.NewLengthMeters); err != nil {
			return walks.CompletionResult{}, err
		}
		result.Exploration.Districts = append(result.Exploration.Districts, d)
	}
	return result, rows.Err()
}

func (r *Repository) City(ctx context.Context, actorID, cityID string) (domain.CityResult, error) {
	tx, err := r.database.Begin(ctx)
	if err != nil {
		return domain.CityResult{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var result domain.CityResult
	var stateStatus *string
	var stateVersion *string
	var updated *time.Time
	if err := tx.QueryRow(ctx, `SELECT g.id,e.geo_data_version_id,e.status,e.updated_at FROM geo_data_versions g LEFT JOIN exploration_states e ON e.actor_id=$2 AND e.city_id=g.city_id WHERE g.city_id=$1 AND g.status='READY'`, cityID, actorID).Scan(&result.GeoDataVersion.ID, &stateVersion, &stateStatus, &updated); err != nil {
		return domain.CityResult{}, err
	}
	if stateStatus != nil && (*stateStatus != "READY" || *stateVersion != result.GeoDataVersion.ID) {
		return domain.CityResult{}, domain.ErrRebuildRequired
	}
	result.State.Status = "READY"
	result.State.UpdatedAt = updated
	if err := tx.QueryRow(ctx, `SELECT COALESCE(SUM(s.length_m) FILTER(WHERE p.street_segment_id IS NOT NULL),0),COALESCE(SUM(s.length_m),0),COUNT(p.street_segment_id) FROM street_segments s LEFT JOIN user_street_segment_progress p ON p.actor_id=$2 AND p.street_segment_id=s.id WHERE s.geo_data_version_id=$1 AND s.classification='EXPLORE'`, result.GeoDataVersion.ID, actorID).Scan(&result.City.ExploredLengthMeters, &result.City.EligibleLengthMeters, &result.City.ExploredSegmentsCount); err != nil {
		return domain.CityResult{}, err
	}
	if result.City.EligibleLengthMeters > 0 {
		result.City.Percentage = result.City.ExploredLengthMeters / result.City.EligibleLengthMeters
	}
	rows, err := tx.Query(ctx, `SELECT d.id,d.name,COALESCE(SUM(ST_Length(ST_CollectionExtract(ST_Intersection(s.geometry,d.boundary),2)::geography)) FILTER(WHERE p.street_segment_id IS NOT NULL),0),COALESCE(SUM(ST_Length(ST_CollectionExtract(ST_Intersection(s.geometry,d.boundary),2)::geography)),0) FROM districts d JOIN district_data_versions dv ON dv.id=d.district_data_version_id AND dv.status='READY' LEFT JOIN street_segments s ON s.geo_data_version_id=$3 AND s.classification='EXPLORE' AND ST_Intersects(s.geometry,d.boundary) LEFT JOIN user_street_segment_progress p ON p.actor_id=$2 AND p.street_segment_id=s.id WHERE d.city_id=$1 GROUP BY d.id,d.name ORDER BY d.name`, cityID, actorID, result.GeoDataVersion.ID)
	if err != nil {
		return domain.CityResult{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var d domain.DistrictMetric
		if err := rows.Scan(&d.DistrictID, &d.Name, &d.ExploredLengthMeters, &d.EligibleLengthMeters); err != nil {
			return domain.CityResult{}, err
		}
		if d.EligibleLengthMeters > 0 {
			d.Percentage = d.ExploredLengthMeters / d.EligibleLengthMeters
		}
		result.Districts = append(result.Districts, d)
	}
	return result, rows.Err()
}

func (r *Repository) Segments(ctx context.Context, actorID, cityID string, b [4]float64, limit int) (json.RawMessage, error) {
	tx, err := r.database.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var version string
	var stateVersion, stateStatus *string
	if err := tx.QueryRow(ctx, `SELECT g.id,e.geo_data_version_id,e.status FROM geo_data_versions g LEFT JOIN exploration_states e ON e.actor_id=$2 AND e.city_id=g.city_id WHERE g.city_id=$1 AND g.status='READY'`, cityID, actorID).Scan(&version, &stateVersion, &stateStatus); err != nil {
		return nil, err
	}
	if stateStatus != nil && (*stateStatus != "READY" || *stateVersion != version) {
		return nil, domain.ErrRebuildRequired
	}
	var raw []byte
	var count int
	err = tx.QueryRow(ctx, `SELECT jsonb_build_object('type','FeatureCollection','features',COALESCE(jsonb_agg(feature ORDER BY id),'[]'::jsonb)),COUNT(*) FROM (SELECT s.id,jsonb_build_object('type','Feature','id',s.id,'geometry',ST_AsGeoJSON(s.geometry)::jsonb,'properties',jsonb_build_object('segmentId',s.id,'classification','EXPLORE')) feature FROM street_segments s JOIN user_street_segment_progress p ON p.actor_id=$1 AND p.street_segment_id=s.id WHERE s.geo_data_version_id=$2 AND s.classification='EXPLORE' AND s.geometry&&ST_MakeEnvelope($3,$4,$5,$6,4326) AND ST_Intersects(s.geometry,ST_MakeEnvelope($3,$4,$5,$6,4326)) ORDER BY s.id LIMIT $7+1) q`, actorID, version, b[0], b[1], b[2], b[3], limit).Scan(&raw, &count)
	if err != nil {
		return nil, err
	}
	if count > limit {
		return nil, querying.ErrFeatureLimit
	}
	return json.RawMessage(raw), nil
}

var _ domain.Repository = (*Repository)(nil)

// lockCurrentReadyVersion makes current-version validation durable through
// commit. A geo publisher updating READY→SUPERSEDED must wait for completion.
func lockCurrentReadyVersion(ctx context.Context, tx pgx.Tx, cityID string) (string, error) {
	var versionID string
	err := tx.QueryRow(ctx, `
		SELECT id
		FROM geo_data_versions
		WHERE city_id=$1 AND status='READY'
		FOR SHARE
	`, cityID).Scan(&versionID)
	if err != nil {
		return "", fmt.Errorf("lock current READY geo version: %w", err)
	}
	return versionID, nil
}
