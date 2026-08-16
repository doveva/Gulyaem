package exploration

import (
	"context"
	"fmt"
	"math"
	"os"
	"strings"
	"testing"
	"time"

	application "github.com/doveva/Gulyaem/backend/internal/exploration"
	geoanalysis "github.com/doveva/Gulyaem/backend/internal/geo/routeanalysis"
	"github.com/doveva/Gulyaem/backend/internal/platform/database"
	"github.com/doveva/Gulyaem/backend/internal/platform/database/geoquery"
	routeanalysisdb "github.com/doveva/Gulyaem/backend/internal/platform/database/routeanalysis"
	"github.com/jackc/pgx/v5"
)

type rebuildFixture struct {
	city, version, districtVersion, actor string
	segments                              []string
	walks                                 []string
}

type persistedProgress struct {
	segmentID                   string
	firstExplored, lastExplored time.Time
	visitCount                  int
	firstWalkID, lastWalkID     string
}

func TestRebuildRestoresNonEmptyProgressAndCurrentStatisticsAgainstPostGIS(t *testing.T) {
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
	fixture := seedNonEmptyRebuildFixture(t, ctx, db)
	defer cleanupNonEmptyRebuildFixture(t, ctx, db, fixture)
	repository := New(db)
	for _, walkID := range fixture.walks {
		if _, err := repository.Complete(ctx, fixture.actor, walkID); err != nil {
			t.Fatalf("complete Walk %s: %v", walkID, err)
		}
	}

	progressBefore := readPersistedProgress(t, ctx, db, fixture.actor)
	if len(progressBefore) != 2 || progressBefore[0].visitCount+progressBefore[1].visitCount != 3 {
		t.Fatalf("non-empty baseline progress=%+v", progressBefore)
	}
	currentBefore, err := repository.City(ctx, fixture.actor, fixture.city)
	if err != nil {
		t.Fatal(err)
	}
	deltaRowsBefore, districtDeltaRowsBefore := historicalDeltaCounts(t, ctx, db, fixture.actor)

	clear, err := db.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer clear.Rollback(ctx)
	if _, err := clear.Exec(ctx, `DELETE FROM user_street_segment_progress WHERE actor_id=$1`, fixture.actor); err != nil {
		t.Fatal(err)
	}
	if _, err := clear.Exec(ctx, `UPDATE exploration_states SET status='REBUILD_REQUIRED',updated_at=now() WHERE actor_id=$1 AND city_id=$2`, fixture.actor, fixture.city); err != nil {
		t.Fatal(err)
	}
	if err := clear.Commit(ctx); err != nil {
		t.Fatal(err)
	}

	analyzer := geoanalysis.NewAnalyzer(routeanalysisdb.New(db, geoquery.New(db)))
	result, err := application.NewRebuilder(analyzer, repository).Rebuild(ctx, fixture.actor, fixture.city)
	if err != nil {
		t.Fatal(err)
	}
	if result.GeoDataVersionID != fixture.version || result.WalksProcessed != 3 || result.SegmentsPublished != 2 {
		t.Fatalf("rebuild result=%+v", result)
	}

	progressAfter := readPersistedProgress(t, ctx, db, fixture.actor)
	assertProgressEquivalent(t, progressBefore, progressAfter)
	currentAfter, err := repository.City(ctx, fixture.actor, fixture.city)
	if err != nil {
		t.Fatal(err)
	}
	assertCurrentExplorationEquivalent(t, currentBefore, currentAfter)
	deltaRowsAfter, districtDeltaRowsAfter := historicalDeltaCounts(t, ctx, db, fixture.actor)
	if deltaRowsAfter != deltaRowsBefore || districtDeltaRowsAfter != districtDeltaRowsBefore {
		t.Fatalf("historical deltas mutated: before=%d/%d after=%d/%d", deltaRowsBefore, districtDeltaRowsBefore, deltaRowsAfter, districtDeltaRowsAfter)
	}
}

func seedNonEmptyRebuildFixture(t *testing.T, ctx context.Context, db *database.Pool) rebuildFixture {
	t.Helper()
	tx, err := db.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	var fixture rebuildFixture
	if err := tx.QueryRow(ctx, `INSERT INTO cities(code,name,country_code,timezone) VALUES($1,'Rebuild Equivalence','RU','Europe/Moscow') RETURNING id`, fmt.Sprintf("rebuild-equivalence-%d", time.Now().UnixNano())).Scan(&fixture.city); err != nil {
		t.Fatal(err)
	}
	if err := tx.QueryRow(ctx, `INSERT INTO geo_data_versions(city_id,source,source_checksum,source_file_name,source_size_bytes,normalization_version,status,import_finished_at,imported_at) VALUES($1,'test',$2,'rebuild.pbf',1,'v1','READY',now(),now()) RETURNING id`, fixture.city, strings.Repeat("c", 64)).Scan(&fixture.version); err != nil {
		t.Fatal(err)
	}
	for _, wkt := range []string{
		"LINESTRING(30.300 59.930,30.302 59.930)",
		"LINESTRING(30.310 59.930,30.312 59.930)",
	} {
		var segmentID string
		if err := tx.QueryRow(ctx, `WITH geometry AS (SELECT ST_GeomFromText($3,4326) AS value) INSERT INTO street_segments(city_id,geo_data_version_id,geometry,length_m,classification) SELECT $1,$2,value,ST_Length(value::geography),'EXPLORE' FROM geometry RETURNING id`, fixture.city, fixture.version, wkt).Scan(&segmentID); err != nil {
			t.Fatal(err)
		}
		fixture.segments = append(fixture.segments, segmentID)
	}
	if err := tx.QueryRow(ctx, `INSERT INTO district_data_versions(city_id,source,source_checksum,source_file_name,source_size_bytes,normalization_version,status,import_finished_at,imported_at) VALUES($1,'test',$2,'rebuild-districts.geojson',1,'v1','READY',now(),now()) RETURNING id`, fixture.city, strings.Repeat("d", 64)).Scan(&fixture.districtVersion); err != nil {
		t.Fatal(err)
	}
	districts := []struct{ externalID, name, boundary, label string }{
		{"a", "District A", "MULTIPOLYGON(((30.299 59.929,30.303 59.929,30.303 59.931,30.299 59.931,30.299 59.929)))", "POINT(30.301 59.930)"},
		{"b", "District B", "MULTIPOLYGON(((30.309 59.929,30.313 59.929,30.313 59.931,30.309 59.931,30.309 59.929)))", "POINT(30.311 59.930)"},
	}
	for _, district := range districts {
		if _, err := tx.Exec(ctx, `INSERT INTO districts(city_id,district_data_version_id,external_id,name,kind,boundary,label_point) VALUES($1,$2,$3,$4,'administrative',ST_GeomFromText($5,4326),ST_GeomFromText($6,4326))`, fixture.city, fixture.districtVersion, district.externalID, district.name, district.boundary, district.label); err != nil {
			t.Fatal(err)
		}
	}
	if err := tx.QueryRow(ctx, `SELECT gen_random_uuid()`).Scan(&fixture.actor); err != nil {
		t.Fatal(err)
	}
	base := time.Date(2026, 8, 14, 8, 0, 0, 0, time.UTC)
	fixture.walks = append(fixture.walks,
		insertRebuildReviewWalk(t, ctx, tx, fixture, fixture.segments[0], "LINESTRING(30.300 59.930,30.302 59.930)", base, "walk-a-first"),
		insertRebuildReviewWalk(t, ctx, tx, fixture, fixture.segments[1], "LINESTRING(30.310 59.930,30.312 59.930)", base.Add(time.Hour), "walk-b"),
		insertRebuildReviewWalk(t, ctx, tx, fixture, fixture.segments[0], "LINESTRING(30.300 59.930,30.302 59.930)", base.Add(2*time.Hour), "walk-a-last"),
	)
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	return fixture
}

func insertRebuildReviewWalk(t *testing.T, ctx context.Context, tx pgx.Tx, fixture rebuildFixture, segmentID, wkt string, finishedAt time.Time, label string) string {
	t.Helper()
	var routeID, walkID string
	if err := tx.QueryRow(ctx, `WITH geometry AS (SELECT ST_GeomFromText($4,4326) AS value) INSERT INTO routes(actor_id,city_id,geo_data_version_id,profile,waypoints,geometry,normalized_geometry,distance_m,estimated_duration_sec,routing_provenance,analysis_provenance,materialization_fingerprint) SELECT $1,$2,$3,'pedestrian','[{"lat":59.93,"lon":30.3},{"lat":59.93,"lon":30.302}]',value,ST_Multi(value),ST_Length(value::geography),600,'{}','{}',$5 FROM geometry RETURNING id`, fixture.actor, fixture.city, fixture.version, wkt, label).Scan(&routeID); err != nil {
		t.Fatal(err)
	}
	if err := tx.QueryRow(ctx, `INSERT INTO walks(actor_id,city_id,route_id,client_request_id,request_fingerprint,status,started_at,finished_at) VALUES($1,$2,$3,gen_random_uuid(),$4,'REVIEW',$5::timestamptz-$6::interval,$5::timestamptz) RETURNING id`, fixture.actor, fixture.city, routeID, label, finishedAt, "30 minutes").Scan(&walkID); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO route_segment_matches(route_id,street_segment_id,classification,matched_length_m,covered_length_m,direct_length_m,required_length_m,coverage_status,provenance,confidence) SELECT $1,s.id,'EXPLORE',s.length_m,s.length_m,s.length_m,LEAST(s.length_m,80),'COMPLETED','DIRECT',1 FROM street_segments s WHERE s.id=$2`, routeID, segmentID); err != nil {
		t.Fatal(err)
	}
	return walkID
}

func readPersistedProgress(t *testing.T, ctx context.Context, db *database.Pool, actorID string) []persistedProgress {
	t.Helper()
	tx, err := db.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	rows, err := tx.Query(ctx, `SELECT street_segment_id,first_explored_at,last_explored_at,visit_count,first_walk_id,last_walk_id FROM user_street_segment_progress WHERE actor_id=$1 ORDER BY street_segment_id`, actorID)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var result []persistedProgress
	for rows.Next() {
		var item persistedProgress
		if err := rows.Scan(&item.segmentID, &item.firstExplored, &item.lastExplored, &item.visitCount, &item.firstWalkID, &item.lastWalkID); err != nil {
			t.Fatal(err)
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return result
}

func assertProgressEquivalent(t *testing.T, before, after []persistedProgress) {
	t.Helper()
	if len(after) != len(before) {
		t.Fatalf("progress size before=%d after=%d", len(before), len(after))
	}
	for index := range before {
		want, got := before[index], after[index]
		if got.segmentID != want.segmentID || got.visitCount != want.visitCount || got.firstWalkID != want.firstWalkID || got.lastWalkID != want.lastWalkID || !got.firstExplored.Equal(want.firstExplored) || !got.lastExplored.Equal(want.lastExplored) {
			t.Fatalf("progress[%d] before=%+v after=%+v", index, want, got)
		}
	}
}

func assertCurrentExplorationEquivalent(t *testing.T, before, after application.CityResult) {
	t.Helper()
	assertClose := func(name string, want, got float64) {
		if math.Abs(want-got) > 0.001 {
			t.Fatalf("%s before=%.6f after=%.6f", name, want, got)
		}
	}
	if before.GeoDataVersion.ID != after.GeoDataVersion.ID || after.State.Status != "READY" || before.City.ExploredSegmentsCount != after.City.ExploredSegmentsCount {
		t.Fatalf("city identity/count before=%+v after=%+v", before, after)
	}
	assertClose("city explored", before.City.ExploredLengthMeters, after.City.ExploredLengthMeters)
	assertClose("city eligible", before.City.EligibleLengthMeters, after.City.EligibleLengthMeters)
	assertClose("city percentage", before.City.Percentage, after.City.Percentage)
	if len(before.Districts) != len(after.Districts) {
		t.Fatalf("district count before=%d after=%d", len(before.Districts), len(after.Districts))
	}
	for index := range before.Districts {
		want, got := before.Districts[index], after.Districts[index]
		if want.DistrictID != got.DistrictID || want.Name != got.Name {
			t.Fatalf("district[%d] before=%+v after=%+v", index, want, got)
		}
		assertClose(want.Name+" explored", want.ExploredLengthMeters, got.ExploredLengthMeters)
		assertClose(want.Name+" eligible", want.EligibleLengthMeters, got.EligibleLengthMeters)
		assertClose(want.Name+" percentage", want.Percentage, got.Percentage)
	}
}

func historicalDeltaCounts(t *testing.T, ctx context.Context, db *database.Pool, actorID string) (int, int) {
	t.Helper()
	tx, err := db.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	var deltas, districts int
	if err := tx.QueryRow(ctx, `SELECT (SELECT COUNT(*) FROM exploration_deltas WHERE actor_id=$1),(SELECT COUNT(*) FROM walk_district_deltas wd JOIN exploration_deltas ed ON ed.walk_id=wd.walk_id WHERE ed.actor_id=$1)`, actorID).Scan(&deltas, &districts); err != nil {
		t.Fatal(err)
	}
	return deltas, districts
}

func cleanupNonEmptyRebuildFixture(t *testing.T, ctx context.Context, db *database.Pool, fixture rebuildFixture) {
	t.Helper()
	tx, err := db.Begin(ctx)
	if err != nil {
		t.Error(err)
		return
	}
	defer tx.Rollback(ctx)
	queries := []struct{ sql, value string }{
		{`DELETE FROM exploration_deltas WHERE actor_id=$1`, fixture.actor},
		{`DELETE FROM exploration_states WHERE actor_id=$1`, fixture.actor},
		{`DELETE FROM user_street_segment_progress WHERE actor_id=$1`, fixture.actor},
		{`DELETE FROM walks WHERE actor_id=$1`, fixture.actor},
		{`DELETE FROM routes WHERE actor_id=$1`, fixture.actor},
		{`DELETE FROM district_data_versions WHERE city_id=$1`, fixture.city},
		{`DELETE FROM geo_data_versions WHERE city_id=$1`, fixture.city},
		{`DELETE FROM cities WHERE id=$1`, fixture.city},
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
