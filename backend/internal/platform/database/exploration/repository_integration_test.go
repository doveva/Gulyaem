package exploration

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	domain "github.com/doveva/Gulyaem/backend/internal/exploration"
	"github.com/doveva/Gulyaem/backend/internal/platform/database"
	"github.com/jackc/pgx/v5"
)

type completionFixture struct{ city, version, districtVersion, actor, segment, walk string }

func TestCompletionIsAtomicIdempotentAndActorScopedAgainstPostGIS(t *testing.T) {
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
	fixture := seedCompletionFixture(t, ctx, db)
	defer cleanupCompletionFixture(t, ctx, db, fixture)
	repository := New(db)
	results := make([]int, 2)
	errorsFound := make([]error, 2)
	var group sync.WaitGroup
	group.Add(2)
	for i := range 2 {
		go func(index int) {
			defer group.Done()
			result, callErr := repository.Complete(ctx, fixture.actor, fixture.walk)
			errorsFound[index] = callErr
			results[index] = result.Exploration.NewSegmentsCount
		}(i)
	}
	group.Wait()
	for i, callErr := range errorsFound {
		if callErr != nil {
			t.Fatalf("completion %d: %v", i, callErr)
		}
		if results[i] != 1 {
			t.Fatalf("completion %d new=%d", i, results[i])
		}
	}
	assertVisitCount(t, ctx, db, fixture.actor, fixture.segment, 1)
	cityResult, err := repository.City(ctx, fixture.actor, fixture.city)
	if err != nil {
		t.Fatal(err)
	}
	if cityResult.City.ExploredSegmentsCount != 1 || cityResult.City.ExploredLengthMeters != 100 {
		t.Fatalf("city exploration=%+v", cityResult.City)
	}
	viewport, err := repository.Segments(ctx, fixture.actor, fixture.city, [4]float64{30.29, 59.92, 30.31, 59.94}, 10000)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(viewport), fixture.segment) {
		t.Fatalf("explored viewport=%s", viewport)
	}
	secondWalk := insertReviewWalk(t, ctx, db, fixture, "second")
	second, err := repository.Complete(ctx, fixture.actor, secondWalk)
	if err != nil {
		t.Fatal(err)
	}
	if second.Exploration.NewSegmentsCount != 0 || second.Exploration.RevisitedSegmentsCount != 1 {
		t.Fatalf("second delta=%+v", second.Exploration)
	}
	assertVisitCount(t, ctx, db, fixture.actor, fixture.segment, 2)
	var foreignActor string
	readTx, err := db.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := readTx.QueryRow(ctx, `SELECT gen_random_uuid()`).Scan(&foreignActor); err != nil {
		t.Fatal(err)
	}
	_ = readTx.Rollback(ctx)
	if _, err := repository.Complete(ctx, foreignActor, fixture.walk); err == nil {
		t.Fatal("foreign actor completed another actor walk")
	}
}

func TestCompletionVersionShareLockBlocksPublisher(t *testing.T) {
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
	fixture := seedCompletionFixture(t, ctx, db)
	defer cleanupCompletionFixture(t, ctx, db, fixture)
	holder, err := db.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer holder.Rollback(ctx)
	lockedVersion, err := lockCurrentReadyVersion(ctx, holder, fixture.city)
	if err != nil {
		t.Fatal(err)
	}
	if lockedVersion != fixture.version {
		t.Fatalf("locked version=%s want %s", lockedVersion, fixture.version)
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
		_, updateErr := publisher.Exec(ctx, `UPDATE geo_data_versions SET status='SUPERSEDED' WHERE id=$1`, fixture.version)
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
		t.Fatal("publisher remained blocked after completion version lock released")
	}
}

func TestCompletionRejectsRouteWhenPinnedVersionIsNoLongerReady(t *testing.T) {
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
	fixture := seedCompletionFixture(t, ctx, db)
	defer cleanupCompletionFixture(t, ctx, db, fixture)
	tx, err := db.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `UPDATE geo_data_versions SET status='SUPERSEDED' WHERE id=$1`, fixture.version); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	_, err = New(db).Complete(ctx, fixture.actor, fixture.walk)
	if !errors.Is(err, domain.ErrRouteGeoVersionStale) {
		t.Fatalf("completion error=%v", err)
	}
	check, err := db.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer check.Rollback(ctx)
	var status string
	var deltaCount int
	if err := check.QueryRow(ctx, `SELECT status FROM walks WHERE id=$1`, fixture.walk).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if err := check.QueryRow(ctx, `SELECT COUNT(*) FROM exploration_deltas WHERE walk_id=$1`, fixture.walk).Scan(&deltaCount); err != nil {
		t.Fatal(err)
	}
	if status != "REVIEW" || deltaCount != 0 {
		t.Fatalf("status=%s deltas=%d", status, deltaCount)
	}
}

func TestCompletionRollsBackEveryMutationAfterLateDatabaseFailure(t *testing.T) {
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
	fixture := seedCompletionFixture(t, ctx, db)
	defer cleanupCompletionFixture(t, ctx, db, fixture)

	functionName := "completion_rollback_" + strings.ReplaceAll(fixture.walk, "-", "")
	triggerName := "completion_rollback_" + strings.ReplaceAll(fixture.walk, "-", "")
	installCompletionFailureTrigger(t, ctx, db, functionName, triggerName, fixture.walk)
	defer dropCompletionFailureTrigger(t, ctx, db, functionName, triggerName)

	_, err = New(db).Complete(ctx, fixture.actor, fixture.walk)
	if err == nil || !strings.Contains(err.Error(), "injected completion failure") {
		t.Fatalf("completion error=%v, want injected database failure", err)
	}

	check, err := db.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer check.Rollback(ctx)
	var status string
	var walkUnchanged, routeNotFinalized bool
	var progressCount, deltaCount, deltaSegmentCount, districtDeltaCount, stateCount int
	err = check.QueryRow(ctx, `SELECT w.status,
		w.completed_at IS NULL AND w.duration_sec IS NULL AND w.distance_m IS NULL,
		r.finalized_at IS NULL,
		(SELECT COUNT(*) FROM user_street_segment_progress WHERE actor_id=$2),
		(SELECT COUNT(*) FROM exploration_deltas WHERE walk_id=$1),
		(SELECT COUNT(*) FROM exploration_delta_segments WHERE walk_id=$1),
		(SELECT COUNT(*) FROM walk_district_deltas WHERE walk_id=$1),
		(SELECT COUNT(*) FROM exploration_states WHERE actor_id=$2 AND city_id=$3)
		FROM walks w JOIN routes r ON r.id=w.route_id WHERE w.id=$1`, fixture.walk, fixture.actor, fixture.city).
		Scan(&status, &walkUnchanged, &routeNotFinalized, &progressCount, &deltaCount, &deltaSegmentCount, &districtDeltaCount, &stateCount)
	if err != nil {
		t.Fatal(err)
	}
	if status != "REVIEW" || !walkUnchanged || !routeNotFinalized || progressCount != 0 ||
		deltaCount != 0 || deltaSegmentCount != 0 || districtDeltaCount != 0 || stateCount != 0 {
		t.Fatalf("rollback state: status=%s walkUnchanged=%t routeNotFinalized=%t progress=%d delta=%d deltaSegments=%d districtDeltas=%d states=%d",
			status, walkUnchanged, routeNotFinalized, progressCount, deltaCount, deltaSegmentCount, districtDeltaCount, stateCount)
	}
}

func installCompletionFailureTrigger(t *testing.T, ctx context.Context, db *database.Pool, functionName, triggerName, walkID string) {
	t.Helper()
	tx, err := db.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	function := pgx.Identifier{"public", functionName}.Sanitize()
	trigger := pgx.Identifier{triggerName}.Sanitize()
	if _, err := tx.Exec(ctx, fmt.Sprintf(`CREATE FUNCTION %s() RETURNS trigger LANGUAGE plpgsql AS $$
		BEGIN RAISE EXCEPTION 'injected completion failure'; END $$`, function)); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, fmt.Sprintf(`CREATE TRIGGER %s BEFORE UPDATE OF status ON walks
		FOR EACH ROW WHEN (NEW.id='%s'::uuid AND NEW.status='COMPLETED') EXECUTE FUNCTION %s()`, trigger, walkID, function)); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
}

func dropCompletionFailureTrigger(t *testing.T, ctx context.Context, db *database.Pool, functionName, triggerName string) {
	t.Helper()
	tx, err := db.Begin(ctx)
	if err != nil {
		t.Errorf("begin failure-trigger cleanup: %v", err)
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()
	function := pgx.Identifier{"public", functionName}.Sanitize()
	trigger := pgx.Identifier{triggerName}.Sanitize()
	if _, err := tx.Exec(ctx, fmt.Sprintf(`DROP TRIGGER IF EXISTS %s ON walks`, trigger)); err != nil {
		t.Errorf("drop failure trigger: %v", err)
		return
	}
	if _, err := tx.Exec(ctx, fmt.Sprintf(`DROP FUNCTION IF EXISTS %s()`, function)); err != nil {
		t.Errorf("drop failure function: %v", err)
		return
	}
	if err := tx.Commit(ctx); err != nil {
		t.Errorf("commit failure-trigger cleanup: %v", err)
	}
}

func seedCompletionFixture(t *testing.T, ctx context.Context, db *database.Pool) completionFixture {
	t.Helper()
	tx, err := db.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var f completionFixture
	code := fmt.Sprintf("completion-%d", time.Now().UnixNano())
	if err := tx.QueryRow(ctx, `INSERT INTO cities(code,name,country_code,timezone) VALUES($1,'Completion Test','RU','Europe/Moscow') RETURNING id`, code).Scan(&f.city); err != nil {
		t.Fatal(err)
	}
	if err := tx.QueryRow(ctx, `INSERT INTO geo_data_versions(city_id,source,source_checksum,source_file_name,source_size_bytes,normalization_version,status,import_finished_at,imported_at) VALUES($1,'test',$2,'test.pbf',1,'v1','READY',now(),now()) RETURNING id`, f.city, strings.Repeat("d", 64)).Scan(&f.version); err != nil {
		t.Fatal(err)
	}
	if err := tx.QueryRow(ctx, `INSERT INTO street_segments(city_id,geo_data_version_id,geometry,length_m,classification) VALUES($1,$2,ST_GeomFromText('LINESTRING(30.30 59.93,30.301 59.93)',4326),100,'EXPLORE') RETURNING id`, f.city, f.version).Scan(&f.segment); err != nil {
		t.Fatal(err)
	}
	var partial, routable string
	_ = tx.QueryRow(ctx, `INSERT INTO street_segments(city_id,geo_data_version_id,geometry,length_m,classification) VALUES($1,$2,ST_GeomFromText('LINESTRING(30.301 59.93,30.302 59.93)',4326),100,'EXPLORE') RETURNING id`, f.city, f.version).Scan(&partial)
	_ = tx.QueryRow(ctx, `INSERT INTO street_segments(city_id,geo_data_version_id,geometry,length_m,classification) VALUES($1,$2,ST_GeomFromText('LINESTRING(30.302 59.93,30.303 59.93)',4326),100,'ROUTABLE_ONLY') RETURNING id`, f.city, f.version).Scan(&routable)
	if err := tx.QueryRow(ctx, `SELECT gen_random_uuid()`).Scan(&f.actor); err != nil {
		t.Fatal(err)
	}
	if err := tx.QueryRow(ctx, `INSERT INTO district_data_versions(city_id,source,source_checksum,source_file_name,source_size_bytes,normalization_version,status,import_finished_at,imported_at) VALUES($1,'test',$2,'district.geojson',1,'v1','READY',now(),now()) RETURNING id`, f.city, strings.Repeat("e", 64)).Scan(&f.districtVersion); err != nil {
		t.Fatal(err)
	}
	_, err = tx.Exec(ctx, `INSERT INTO districts(city_id,district_data_version_id,external_id,name,kind,boundary,label_point) VALUES($1,$2,'d1','Test District','administrative',ST_GeomFromText('MULTIPOLYGON(((30.29 59.92,30.31 59.92,30.31 59.94,30.29 59.94,30.29 59.92)))',4326),ST_GeomFromText('POINT(30.30 59.93)',4326))`, f.city, f.districtVersion)
	if err != nil {
		t.Fatal(err)
	}
	routeID, walkID := insertReviewWalkTx(t, ctx, tx, f, "first")
	f.walk = walkID
	_, err = tx.Exec(ctx, `INSERT INTO route_segment_matches(route_id,street_segment_id,classification,covered_length_m,direct_length_m,required_length_m,coverage_status) VALUES($1,$2,'EXPLORE',100,100,60,'COMPLETED'),($1,$3,'EXPLORE',40,40,60,'PARTIAL'),($1,$4,'ROUTABLE_ONLY',100,100,0,'COMPLETED')`, routeID, f.segment, partial, routable)
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	return f
}

func insertReviewWalk(t *testing.T, ctx context.Context, db *database.Pool, f completionFixture, label string) string {
	t.Helper()
	tx, err := db.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	routeID, walkID := insertReviewWalkTx(t, ctx, tx, f, label)
	_, err = tx.Exec(ctx, `INSERT INTO route_segment_matches(route_id,street_segment_id,classification,covered_length_m,direct_length_m,required_length_m,coverage_status) VALUES($1,$2,'EXPLORE',100,100,60,'COMPLETED')`, routeID, f.segment)
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	return walkID
}
func insertReviewWalkTx(t *testing.T, ctx context.Context, tx pgx.Tx, f completionFixture, label string) (string, string) {
	t.Helper()
	var routeID, walkID string
	waypoints := `[{"lat":59.93,"lon":30.30},{"lat":59.93,"lon":30.303}]`
	geometry := `{"type":"LineString","coordinates":[[30.30,59.93],[30.303,59.93]]}`
	normalized := `{"type":"MultiLineString","coordinates":[[[30.30,59.93],[30.303,59.93]]]}`
	err := tx.QueryRow(ctx, `INSERT INTO routes(actor_id,city_id,geo_data_version_id,profile,waypoints,geometry,normalized_geometry,distance_m,estimated_duration_sec,routing_provenance,analysis_provenance,materialization_fingerprint) VALUES($1,$2,$3,'pedestrian',$4,ST_SetSRID(ST_GeomFromGeoJSON($5),4326),ST_SetSRID(ST_GeomFromGeoJSON($6),4326),300,240,'{}','{}',$7) RETURNING id`, f.actor, f.city, f.version, waypoints, geometry, normalized, label).Scan(&routeID)
	if err != nil {
		t.Fatal(err)
	}
	err = tx.QueryRow(ctx, `INSERT INTO walks(actor_id,city_id,route_id,client_request_id,request_fingerprint,status,started_at,finished_at) VALUES($1,$2,$3,gen_random_uuid(),$4,'REVIEW',now()-interval '10 minutes',now()) RETURNING id`, f.actor, f.city, routeID, label).Scan(&walkID)
	if err != nil {
		t.Fatal(err)
	}
	return routeID, walkID
}
func assertVisitCount(t *testing.T, ctx context.Context, db *database.Pool, actor, segment string, want int) {
	t.Helper()
	tx, err := db.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	var got int
	if err := tx.QueryRow(ctx, `SELECT visit_count FROM user_street_segment_progress WHERE actor_id=$1 AND street_segment_id=$2`, actor, segment).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("visit_count=%d want %d", got, want)
	}
}
func cleanupCompletionFixture(t *testing.T, ctx context.Context, db *database.Pool, f completionFixture) {
	t.Helper()
	tx, err := db.Begin(ctx)
	if err != nil {
		t.Error(err)
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := []struct {
		sql   string
		value string
	}{
		{`DELETE FROM exploration_deltas WHERE actor_id=$1`, f.actor}, {`DELETE FROM exploration_states WHERE actor_id=$1`, f.actor},
		{`DELETE FROM user_street_segment_progress WHERE actor_id=$1`, f.actor}, {`DELETE FROM walks WHERE actor_id=$1`, f.actor},
		{`DELETE FROM routes WHERE actor_id=$1`, f.actor}, {`DELETE FROM district_data_versions WHERE city_id=$1`, f.city},
		{`DELETE FROM geo_data_versions WHERE city_id=$1`, f.city}, {`DELETE FROM cities WHERE id=$1`, f.city},
	}
	for _, query := range queries {
		if _, err := tx.Exec(ctx, query.sql, query.value); err != nil {
			t.Errorf("cleanup: %v", err)
			return
		}
	}
	if err := tx.Commit(ctx); err != nil {
		t.Error(err)
	}
}
