package geoquery

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/doveva/Gulyaem/backend/internal/geo/domain"
	"github.com/doveva/Gulyaem/backend/internal/geo/querying"
	"github.com/doveva/Gulyaem/backend/internal/platform/database"
)

func TestRepositoryQueriesCurrentViewportAndHistoricalDetailAgainstPostGIS(t *testing.T) {
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

	cityCode := fmt.Sprintf("geoquery-%d", time.Now().UnixNano())
	cityID, oldVersionID, currentVersionID, oldSegmentID := seedQueryFixture(t, ctx, db, cityCode)
	defer deleteQueryFixture(t, ctx, db, cityID)

	repository := New(db)
	version, err := repository.CurrentVersion(ctx, cityID)
	if err != nil {
		t.Fatal(err)
	}
	if version.ID != currentVersionID || version.Status != domain.GeoDataVersionReady {
		t.Fatalf("current version = %+v", version)
	}

	minimum := 20.0
	segments, err := repository.Segments(ctx, querying.SegmentFilter{
		CityID:          cityID,
		BBox:            querying.BBox{West: 30.30, South: 59.93, East: 30.315, North: 59.945},
		Classifications: []domain.StreetSegmentClassification{domain.StreetSegmentExplore},
		MinLength:       &minimum,
	}, 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(segments) != 1 || segments[0].GeoDataVersionID != currentVersionID || segments[0].Attributes.ReasonCode != "integration_explore" {
		t.Fatalf("viewport segments = %+v", segments)
	}

	historical, err := repository.Segment(ctx, oldSegmentID)
	if err != nil {
		t.Fatal(err)
	}
	if historical.GeoDataVersionID != oldVersionID || historical.IsCurrent || historical.VersionStatus != domain.GeoDataVersionSuperseded {
		t.Fatalf("historical segment = %+v", historical)
	}
}

func seedQueryFixture(t *testing.T, ctx context.Context, db *database.Pool, code string) (cityID, oldVersionID, currentVersionID, oldSegmentID string) {
	t.Helper()
	tx, err := db.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := tx.QueryRow(ctx, `
		INSERT INTO cities (code, name, country_code, timezone)
		VALUES ($1, 'Geo Query Test', 'RU', 'Europe/Moscow')
		RETURNING id
	`, code).Scan(&cityID); err != nil {
		t.Fatal(err)
	}
	insertVersion := func(status domain.GeoDataVersionStatus, checksum, normalization string) string {
		var id string
		if err := tx.QueryRow(ctx, `
			INSERT INTO geo_data_versions (
				city_id, source, source_checksum, source_file_name, source_size_bytes,
				normalization_version, status, import_finished_at, imported_at, import_report
			) VALUES ($1, 'integration-test', $2, 'fixture.osm.pbf', 1, $3, $4, now(), now(), '{"outcome":"imported"}')
			RETURNING id
		`, cityID, checksum, normalization, status).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	oldVersionID = insertVersion(domain.GeoDataVersionSuperseded, strings.Repeat("a", 64), "query-v1")
	currentVersionID = insertVersion(domain.GeoDataVersionReady, strings.Repeat("b", 64), "query-v2")
	insertSegment := func(versionID, geometry string, length float64, classification domain.StreetSegmentClassification, reason string) string {
		var id string
		if err := tx.QueryRow(ctx, `
			INSERT INTO street_segments (city_id, geo_data_version_id, geometry, length_m, classification, attributes)
			VALUES ($1, $2, ST_GeomFromText($3, 4326), $4, $5, jsonb_build_object('reasonCode', $6::text))
			RETURNING id
		`, cityID, versionID, geometry, length, classification, reason).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	oldSegmentID = insertSegment(oldVersionID, "LINESTRING(30.305 59.935,30.306 59.936)", 60, domain.StreetSegmentExplore, "historical")
	insertSegment(currentVersionID, "LINESTRING(30.305 59.935,30.306 59.936)", 60, domain.StreetSegmentExplore, "integration_explore")
	insertSegment(currentVersionID, "LINESTRING(30.325 59.94,30.326 59.941)", 80, domain.StreetSegmentIgnore, "outside_bbox")
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	return cityID, oldVersionID, currentVersionID, oldSegmentID
}

func deleteQueryFixture(t *testing.T, ctx context.Context, db *database.Pool, cityID string) {
	t.Helper()
	tx, err := db.Begin(ctx)
	if err != nil {
		t.Errorf("begin cleanup: %v", err)
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `DELETE FROM geo_data_versions WHERE city_id = $1`, cityID); err != nil {
		t.Errorf("delete versions: %v", err)
		return
	}
	if _, err := tx.Exec(ctx, `DELETE FROM cities WHERE id = $1`, cityID); err != nil {
		t.Errorf("delete city: %v", err)
		return
	}
	if err := tx.Commit(ctx); err != nil {
		t.Errorf("commit cleanup: %v", err)
	}
}
