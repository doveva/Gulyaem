package exploration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	domain "github.com/doveva/Gulyaem/backend/internal/exploration"
	"github.com/jackc/pgx/v5"
)

func (r *Repository) CompletedWalks(ctx context.Context, actorID, cityID string) ([]domain.RebuildWalk, error) {
	tx, err := r.database.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	rows, err := tx.Query(ctx, `SELECT w.id,w.finished_at,ST_AsGeoJSON(r.geometry)
		FROM walks w
		JOIN routes r ON r.id=w.route_id AND r.actor_id=w.actor_id AND r.city_id=w.city_id
		WHERE w.actor_id=$1 AND w.city_id=$2 AND w.status='COMPLETED'
		ORDER BY w.finished_at,w.id`, actorID, cityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []domain.RebuildWalk{}
	for rows.Next() {
		var item domain.RebuildWalk
		var geometry []byte
		if err := rows.Scan(&item.ID, &item.FinishedAt, &geometry); err != nil {
			return nil, err
		}
		item.Geometry = json.RawMessage(geometry)
		result = append(result, item)
	}
	return result, rows.Err()
}

func (r *Repository) PublishRebuild(ctx context.Context, actorID, cityID, versionID string, progress []domain.RebuiltProgress) error {
	tx, err := r.database.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, actorID+":"+cityID); err != nil {
		return err
	}
	var current string
	err = tx.QueryRow(ctx, `SELECT id FROM geo_data_versions WHERE city_id=$1 AND status='READY' FOR UPDATE`, cityID).Scan(&current)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrRebuildRequired
	}
	if err != nil {
		return err
	}
	if current != versionID {
		return domain.ErrRebuildRequired
	}
	_, err = tx.Exec(ctx, `DELETE FROM user_street_segment_progress p USING street_segments s WHERE p.actor_id=$1 AND p.street_segment_id=s.id AND s.city_id=$2`, actorID, cityID)
	if err != nil {
		return fmt.Errorf("clear materialized progress: %w", err)
	}
	for _, item := range progress {
		_, err = tx.Exec(ctx, `INSERT INTO user_street_segment_progress(actor_id,street_segment_id,first_explored_at,last_explored_at,visit_count,first_walk_id,last_walk_id) VALUES($1,$2,$3,$4,$5,$6,$7)`, actorID, item.SegmentID, item.FirstExploredAt, item.LastExploredAt, item.VisitCount, item.FirstWalkID, item.LastWalkID)
		if err != nil {
			return fmt.Errorf("publish rebuilt segment %s: %w", item.SegmentID, err)
		}
	}
	_, err = tx.Exec(ctx, `INSERT INTO exploration_states(actor_id,city_id,geo_data_version_id,status,updated_at,rebuilt_at) VALUES($1,$2,$3,'READY',now(),now()) ON CONFLICT(actor_id,city_id) DO UPDATE SET geo_data_version_id=EXCLUDED.geo_data_version_id,status='READY',updated_at=now(),rebuilt_at=now()`, actorID, cityID, versionID)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

var _ domain.RebuildRepository = (*Repository)(nil)
