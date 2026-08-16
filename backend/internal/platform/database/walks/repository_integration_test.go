package walks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/doveva/Gulyaem/backend/internal/platform/database"
	"github.com/doveva/Gulyaem/backend/internal/routing/port"
	domain "github.com/doveva/Gulyaem/backend/internal/walks"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestMaterializationLifecycleCorrectionAndOwnershipAgainstPostGIS(t *testing.T) {
	databaseURL := os.Getenv("GULYAEM_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("GULYAEM_TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	db, err := database.Open(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	city, version, segment, actor, routeID, walkID, requestID := seedWalkRepositoryFixture(t, ctx, db)
	defer cleanupWalkRepositoryFixture(t, ctx, db, actor, city)
	repository := New(db)
	materialization := walkMaterialization(actor, city, version, segment, routeID, 30.301)
	walk := domain.Walk{ID: walkID, ActorID: actor, CityID: city, RouteID: routeID, ClientRequestID: requestID, RequestFingerprint: "request-v1", Status: domain.StatusDraft}
	created, err := repository.Create(ctx, materialization, walk)
	if err != nil {
		t.Fatal(err)
	}
	if created.Route.Revision != 1 || created.Walk.Status != domain.StatusDraft {
		t.Fatalf("created=%+v", created)
	}
	var foreign string
	tx, _ := db.Begin(ctx)
	_ = tx.QueryRow(ctx, `SELECT gen_random_uuid()`).Scan(&foreign)
	_ = tx.Rollback(ctx)
	if _, err := repository.Get(ctx, foreign, walkID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("foreign read error=%v", err)
	}
	now := time.Now().UTC()
	active, err := repository.Transition(ctx, actor, walkID, domain.StatusDraft, domain.StatusActive, now)
	if err != nil || active.Walk.StartedAt == nil {
		t.Fatalf("active=%+v err=%v", active, err)
	}
	review, err := repository.Transition(ctx, actor, walkID, domain.StatusActive, domain.StatusReview, now.Add(time.Minute))
	if err != nil || review.Walk.FinishedAt == nil {
		t.Fatalf("review=%+v err=%v", review, err)
	}
	corrected := walkMaterialization(actor, city, version, segment, routeID, 30.302)
	updated, err := repository.ReplaceRoute(ctx, actor, walkID, []domain.Status{domain.StatusReview}, corrected)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Route.Revision != 2 || updated.Route.Geometry == nil {
		t.Fatalf("updated route=%+v", updated.Route)
	}
	staleTx, err := db.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := staleTx.Exec(ctx, `UPDATE geo_data_versions SET status='SUPERSEDED' WHERE id=$1`, version); err != nil {
		t.Fatal(err)
	}
	if err := staleTx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	_, err = repository.ReplaceRoute(ctx, actor, walkID, []domain.Status{domain.StatusReview}, corrected)
	if !errors.Is(err, domain.ErrMaterializationGeoVersionStale) {
		t.Fatalf("stale correction error=%v", err)
	}
	unchanged, err := repository.Get(ctx, actor, walkID)
	if err != nil {
		t.Fatal(err)
	}
	if unchanged.Route.Revision != 2 {
		t.Fatalf("stale correction changed revision to %d", unchanged.Route.Revision)
	}
	idTx, _ := db.Begin(ctx)
	var staleRoute, staleWalk, staleRequest string
	_ = idTx.QueryRow(ctx, `SELECT gen_random_uuid(),gen_random_uuid(),gen_random_uuid()`).Scan(&staleRoute, &staleWalk, &staleRequest)
	_ = idTx.Rollback(ctx)
	staleMaterialization := walkMaterialization(actor, city, version, segment, staleRoute, 30.302)
	_, err = repository.Create(ctx, staleMaterialization, domain.Walk{ID: staleWalk, ActorID: actor, CityID: city, RouteID: staleRoute, ClientRequestID: staleRequest, RequestFingerprint: "stale", Status: domain.StatusDraft})
	if !errors.Is(err, domain.ErrMaterializationGeoVersionStale) {
		t.Fatalf("stale materialization error=%v", err)
	}
	checkTx, err := db.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer checkTx.Rollback(ctx)
	var staleRoutes, staleWalks int
	if err := checkTx.QueryRow(ctx, `SELECT COUNT(*) FROM routes WHERE id=$1`, staleRoute).Scan(&staleRoutes); err != nil {
		t.Fatal(err)
	}
	if err := checkTx.QueryRow(ctx, `SELECT COUNT(*) FROM walks WHERE id=$1`, staleWalk).Scan(&staleWalks); err != nil {
		t.Fatal(err)
	}
	if staleRoutes != 0 || staleWalks != 0 {
		t.Fatalf("stale write persisted routes=%d walks=%d", staleRoutes, staleWalks)
	}
}

func TestMaterializationVersionShareLockBlocksPublisher(t *testing.T) {
	databaseURL := os.Getenv("GULYAEM_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("GULYAEM_TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	db, err := database.Open(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	city, version, _, actor, _, _, _ := seedWalkRepositoryFixture(t, ctx, db)
	defer cleanupWalkRepositoryFixture(t, ctx, db, actor, city)
	holder, err := db.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer holder.Rollback(ctx)
	if err := lockExpectedReadyVersion(ctx, holder, version, city); err != nil {
		t.Fatal(err)
	}
	publisher, err := db.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer publisher.Rollback(ctx)
	started := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		close(started)
		_, updateErr := publisher.Exec(ctx, `UPDATE geo_data_versions SET status='SUPERSEDED' WHERE id=$1`, version)
		done <- updateErr
	}()
	<-started
	select {
	case updateErr := <-done:
		t.Fatalf("publisher was not blocked: %v", updateErr)
	case <-time.After(150 * time.Millisecond):
	}
	if err := holder.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case updateErr := <-done:
		if updateErr != nil {
			t.Fatal(updateErr)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("publisher remained blocked after materialization commit")
	}
}

func TestWalkRouteOwnershipConstraintAgainstPostGIS(t *testing.T) {
	databaseURL := os.Getenv("GULYAEM_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("GULYAEM_TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	db, err := database.Open(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	city, version, segment, actor, routeID, walkID, requestID := seedWalkRepositoryFixture(t, ctx, db)
	defer cleanupWalkRepositoryFixture(t, ctx, db, actor, city)
	repository := New(db)
	materialization := walkMaterialization(actor, city, version, segment, routeID, 30.301)
	_, err = repository.Create(ctx, materialization, domain.Walk{ID: walkID, ActorID: actor, CityID: city, RouteID: routeID, ClientRequestID: requestID, RequestFingerprint: "owner", Status: domain.StatusDraft})
	if err != nil {
		t.Fatal(err)
	}

	t.Run("actor", func(t *testing.T) {
		tx, err := db.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer tx.Rollback(ctx)
		var foreignActor, foreignWalk, foreignRequest string
		if err := tx.QueryRow(ctx, `SELECT gen_random_uuid(),gen_random_uuid(),gen_random_uuid()`).Scan(&foreignActor, &foreignWalk, &foreignRequest); err != nil {
			t.Fatal(err)
		}
		_, err = tx.Exec(ctx, `INSERT INTO walks(id,actor_id,city_id,route_id,client_request_id,request_fingerprint,status) VALUES($1,$2,$3,$4,$5,'foreign-actor','DRAFT')`, foreignWalk, foreignActor, city, routeID, foreignRequest)
		assertConstraintViolation(t, err, "walks_route_owner_fk")
	})

	t.Run("city", func(t *testing.T) {
		tx, err := db.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer tx.Rollback(ctx)
		var foreignCity, foreignWalk, foreignRequest string
		if err := tx.QueryRow(ctx, `INSERT INTO cities(code,name,country_code,timezone) VALUES($1,'Foreign City','RU','Europe/Moscow') RETURNING id`, fmt.Sprintf("walk-owner-%d", time.Now().UnixNano())).Scan(&foreignCity); err != nil {
			t.Fatal(err)
		}
		if err := tx.QueryRow(ctx, `SELECT gen_random_uuid(),gen_random_uuid()`).Scan(&foreignWalk, &foreignRequest); err != nil {
			t.Fatal(err)
		}
		_, err = tx.Exec(ctx, `INSERT INTO walks(id,actor_id,city_id,route_id,client_request_id,request_fingerprint,status) VALUES($1,$2,$3,$4,$5,'foreign-city','DRAFT')`, foreignWalk, actor, foreignCity, routeID, foreignRequest)
		assertConstraintViolation(t, err, "walks_route_owner_fk")
	})
}

func TestMaterializationRejectsSegmentFromDifferentGeoVersionAgainstPostGIS(t *testing.T) {
	databaseURL := os.Getenv("GULYAEM_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("GULYAEM_TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	db, err := database.Open(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	city, version, _, actor, routeID, walkID, requestID := seedWalkRepositoryFixture(t, ctx, db)
	defer cleanupWalkRepositoryFixture(t, ctx, db, actor, city)
	tx, err := db.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	var otherVersion, otherSegment string
	if err := tx.QueryRow(ctx, `INSERT INTO geo_data_versions(city_id,source,source_checksum,source_file_name,source_size_bytes,normalization_version,status,import_finished_at,imported_at) VALUES($1,'test',$2,'other.pbf',1,'v1','SUPERSEDED',now(),now()) RETURNING id`, city, strings.Repeat("e", 64)).Scan(&otherVersion); err != nil {
		t.Fatal(err)
	}
	if err := tx.QueryRow(ctx, `INSERT INTO street_segments(city_id,geo_data_version_id,geometry,length_m,classification) VALUES($1,$2,ST_GeomFromText('LINESTRING(30.3 59.94,30.302 59.94)',4326),100,'EXPLORE') RETURNING id`, city, otherVersion).Scan(&otherSegment); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}

	repository := New(db)
	materialization := walkMaterialization(actor, city, version, otherSegment, routeID, 30.301)
	_, err = repository.Create(ctx, materialization, domain.Walk{ID: walkID, ActorID: actor, CityID: city, RouteID: routeID, ClientRequestID: requestID, RequestFingerprint: "wrong-version", Status: domain.StatusDraft})
	if !errors.Is(err, domain.ErrSegmentGeoVersionMismatch) {
		t.Fatalf("materialization error=%v", err)
	}
	check, err := db.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer check.Rollback(ctx)
	var routes, walks, matches int
	if err := check.QueryRow(ctx, `SELECT (SELECT COUNT(*) FROM routes WHERE id=$1),(SELECT COUNT(*) FROM walks WHERE id=$2),(SELECT COUNT(*) FROM route_segment_matches WHERE route_id=$1)`, routeID, walkID).Scan(&routes, &walks, &matches); err != nil {
		t.Fatal(err)
	}
	if routes != 0 || walks != 0 || matches != 0 {
		t.Fatalf("mismatched materialization persisted routes=%d walks=%d matches=%d", routes, walks, matches)
	}
}

func assertConstraintViolation(t *testing.T, err error, constraint string) {
	t.Helper()
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23503" || pgErr.ConstraintName != constraint {
		t.Fatalf("constraint error=%v", err)
	}
}

func walkMaterialization(actor, city, version, segment, routeID string, end float64) domain.Materialization {
	geometry := json.RawMessage(fmt.Sprintf(`{"type":"LineString","coordinates":[[30.3,59.93],[%.3f,59.93]]}`, end))
	normalized := json.RawMessage(fmt.Sprintf(`{"type":"MultiLineString","coordinates":[[[30.3,59.93],[%.3f,59.93]]]}`, end))
	return domain.Materialization{Route: domain.Route{ID: routeID, ActorID: actor, CityID: city, GeoDataVersionID: version, Profile: "pedestrian", Waypoints: []port.Point{{Lat: 59.93, Lon: 30.3}, {Lat: 59.93, Lon: end}}, Geometry: geometry, NormalizedGeometry: normalized, DistanceMeters: 100, EstimatedDurationSeconds: 80, RoutingProvenance: json.RawMessage(`{"engine":"valhalla"}`), AnalysisProvenance: json.RawMessage(`{"analysisVersion":"v1"}`), MaterializationFingerprint: "preview-v1", Revision: 1}, Matches: []domain.SegmentMatch{{SegmentID: segment, Classification: "EXPLORE", MatchedMeters: 100, CoveredMeters: 100, DirectMeters: 100, RequiredMeters: 60, Status: "COMPLETED"}}}
}
func seedWalkRepositoryFixture(t *testing.T, ctx context.Context, db *database.Pool) (city, version, segment, actor, route, walk, request string) {
	t.Helper()
	tx, err := db.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	if err := tx.QueryRow(ctx, `INSERT INTO cities(code,name,country_code,timezone) VALUES($1,'Walk Repo Test','RU','Europe/Moscow') RETURNING id`, fmt.Sprintf("walk-repo-%d", time.Now().UnixNano())).Scan(&city); err != nil {
		t.Fatal(err)
	}
	if err := tx.QueryRow(ctx, `INSERT INTO geo_data_versions(city_id,source,source_checksum,source_file_name,source_size_bytes,normalization_version,status,import_finished_at,imported_at) VALUES($1,'test',$2,'test.pbf',1,'v1','READY',now(),now()) RETURNING id`, city, strings.Repeat("f", 64)).Scan(&version); err != nil {
		t.Fatal(err)
	}
	if err := tx.QueryRow(ctx, `INSERT INTO street_segments(city_id,geo_data_version_id,geometry,length_m,classification) VALUES($1,$2,ST_GeomFromText('LINESTRING(30.3 59.93,30.302 59.93)',4326),100,'EXPLORE') RETURNING id`, city, version).Scan(&segment); err != nil {
		t.Fatal(err)
	}
	if err := tx.QueryRow(ctx, `SELECT gen_random_uuid(),gen_random_uuid(),gen_random_uuid(),gen_random_uuid()`).Scan(&actor, &route, &walk, &request); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	return
}
func cleanupWalkRepositoryFixture(t *testing.T, ctx context.Context, db *database.Pool, actor, city string) {
	t.Helper()
	tx, err := db.Begin(ctx)
	if err != nil {
		t.Error(err)
		return
	}
	defer tx.Rollback(ctx)
	for _, item := range []struct{ sql, value string }{{`DELETE FROM walks WHERE actor_id=$1`, actor}, {`DELETE FROM routes WHERE actor_id=$1`, actor}, {`DELETE FROM geo_data_versions WHERE city_id=$1`, city}, {`DELETE FROM cities WHERE id=$1`, city}} {
		if _, err := tx.Exec(ctx, item.sql, item.value); err != nil {
			t.Error(err)
			return
		}
	}
	if err := tx.Commit(ctx); err != nil {
		t.Error(err)
	}
}
