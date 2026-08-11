package routeanalysis

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/doveva/Gulyaem/backend/internal/geo/domain"
	geoanalysis "github.com/doveva/Gulyaem/backend/internal/geo/routeanalysis"
	"github.com/doveva/Gulyaem/backend/internal/platform/database"
	"github.com/doveva/Gulyaem/backend/internal/platform/database/geoquery"
)

func TestCoverageSegmentsKeepsMixedGradesLocalAgainstPostGIS(t *testing.T) {
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

	cityID, segmentIDs := seedMixedGradeFixture(t, ctx, db)
	defer deleteMixedGradeFixture(t, ctx, db, cityID)
	repository := New(db, geoquery.New(db))
	surfaceGeometry := []domain.Point{{Lon: 30, Lat: 60}, {Lon: 30.001, Lat: 60}}
	tunnelGeometry := []domain.Point{{Lon: 30.01, Lat: 60}, {Lon: 30.011, Lat: 60}}
	fragments := []geoanalysis.NormalizedRouteFragment{
		{Geometry: surfaceGeometry, GradeSignature: "surface"},
		// Duplicate same-grade geometry verifies that its buffer is unioned rather than counted twice.
		{Geometry: surfaceGeometry, GradeSignature: "surface"},
		{Geometry: tunnelGeometry, GradeSignature: "surface;tunnel=yes;level=-1"},
	}
	candidates, err := repository.CoverageSegments(ctx, cityID, fragments, 35, 225)
	if err != nil {
		t.Fatal(err)
	}
	byID := make(map[string]geoanalysis.CandidateSegment, len(candidates))
	for _, candidate := range candidates {
		byID[candidate.ID] = candidate
	}
	if len(candidates) != 3 {
		t.Fatalf("context candidates = %d, want 3", len(candidates))
	}
	for name, segmentID := range segmentIDs {
		if _, found := byID[segmentID]; !found {
			t.Fatalf("context candidate %s is missing", name)
		}
	}
	surface := byID[segmentIDs["surface-a"]]
	parallelTunnel := byID[segmentIDs["parallel-tunnel-a"]]
	tunnel := byID[segmentIDs["tunnel-b"]]
	if surface.RadiusCoveredMeters <= 0 || math.Abs(surface.RadiusCoveredMeters-surface.LengthMeters) > .5 {
		t.Fatalf("surface coverage = %.3f of %.3f", surface.RadiusCoveredMeters, surface.LengthMeters)
	}
	if parallelTunnel.RadiusCoveredMeters != 0 {
		t.Fatalf("parallel tunnel near surface coverage = %.3f, want 0", parallelTunnel.RadiusCoveredMeters)
	}
	if tunnel.RadiusCoveredMeters <= 0 || math.Abs(tunnel.RadiusCoveredMeters-tunnel.LengthMeters) > .5 {
		t.Fatalf("matched tunnel coverage = %.3f of %.3f", tunnel.RadiusCoveredMeters, tunnel.LengthMeters)
	}
}

func seedMixedGradeFixture(t *testing.T, ctx context.Context, db *database.Pool) (string, map[string]string) {
	t.Helper()
	tx, err := db.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var cityID, versionID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO cities (code, name, country_code, timezone)
		VALUES ($1, 'Mixed Grade Test', 'RU', 'Europe/Moscow')
		RETURNING id
	`, fmt.Sprintf("mixed-grade-%d", time.Now().UnixNano())).Scan(&cityID); err != nil {
		t.Fatal(err)
	}
	if err := tx.QueryRow(ctx, `
		INSERT INTO geo_data_versions (
			city_id, source, source_checksum, source_file_name, source_size_bytes,
			normalization_version, status, import_finished_at, imported_at, import_report
		) VALUES ($1, 'integration-test', $2, 'mixed-grade.osm.pbf', 1,
		          'mixed-grade-v1', 'READY', now(), now(), '{"outcome":"imported"}')
		RETURNING id
	`, cityID, strings.Repeat("d", 64)).Scan(&versionID); err != nil {
		t.Fatal(err)
	}
	segmentIDs := make(map[string]string)
	insertSegment := func(name, geometry string, tags map[string]string) {
		attributes, marshalErr := json.Marshal(domain.StreetSegmentAttributes{ReasonCode: name, SourceTags: tags})
		if marshalErr != nil {
			t.Fatal(marshalErr)
		}
		var segmentID string
		if err := tx.QueryRow(ctx, `
			WITH segment AS (SELECT ST_GeomFromText($3, 4326) AS geometry)
			INSERT INTO street_segments (
				city_id, geo_data_version_id, geometry, length_m, classification, attributes
			)
			SELECT $1, $2, geometry, ST_Length(geometry::geography), 'EXPLORE', $4::jsonb
			FROM segment
			RETURNING id
		`, cityID, versionID, geometry, string(attributes)).Scan(&segmentID); err != nil {
			t.Fatal(err)
		}
		segmentIDs[name] = segmentID
	}
	insertSegment("surface-a", "LINESTRING(30 60,30.001 60)", nil)
	insertSegment("parallel-tunnel-a", "LINESTRING(30 60.00005,30.001 60.00005)", map[string]string{"tunnel": "yes", "level": "-1"})
	insertSegment("tunnel-b", "LINESTRING(30.01 60,30.011 60)", map[string]string{"tunnel": "yes", "level": "-1"})
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	return cityID, segmentIDs
}

func deleteMixedGradeFixture(t *testing.T, ctx context.Context, db *database.Pool, cityID string) {
	t.Helper()
	tx, err := db.Begin(ctx)
	if err != nil {
		t.Errorf("begin mixed-grade cleanup: %v", err)
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `DELETE FROM geo_data_versions WHERE city_id = $1`, cityID); err != nil {
		t.Errorf("delete mixed-grade version: %v", err)
		return
	}
	if _, err := tx.Exec(ctx, `DELETE FROM cities WHERE id = $1`, cityID); err != nil {
		t.Errorf("delete mixed-grade city: %v", err)
		return
	}
	if err := tx.Commit(ctx); err != nil {
		t.Errorf("commit mixed-grade cleanup: %v", err)
	}
}
