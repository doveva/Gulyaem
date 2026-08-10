package districtversion

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/doveva/Gulyaem/backend/internal/geo/domain"
	"github.com/doveva/Gulyaem/backend/internal/platform/database"
)

func TestRepositoryLifecycleAgainstPostGIS(t *testing.T) {
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
	cityCode := fmt.Sprintf("district-%d", time.Now().UnixNano())
	cityID := createCity(t, ctx, db, cityCode)
	defer deleteCity(t, ctx, db, cityID)
	repository := New(db)

	first := beginImport(t, ctx, repository, cityCode, strings.Repeat("a", 64), "district-v1")
	firstReady, err := repository.CompleteImport(ctx, first.Version.ID, nil,
		domain.DistrictImportReport{Outcome: "imported", FeaturesProcessed: 1, DistrictsPublished: 1}, testDistricts())
	if err != nil {
		t.Fatal(err)
	}
	if firstReady.Status != domain.GeoDataVersionReady || districtCount(t, ctx, db, firstReady.ID) != 1 {
		t.Fatalf("first version = %+v", firstReady)
	}
	idempotent := beginImport(t, ctx, repository, cityCode, strings.Repeat("a", 64), "district-v1")
	if !idempotent.AlreadyReady || idempotent.Version.ID != firstReady.ID {
		t.Fatalf("idempotent result = %+v", idempotent)
	}

	second := beginImport(t, ctx, repository, cityCode, strings.Repeat("b", 64), "district-v2")
	secondReady, err := repository.CompleteImport(ctx, second.Version.ID, nil,
		domain.DistrictImportReport{Outcome: "imported", FeaturesProcessed: 1, DistrictsPublished: 1}, testDistricts())
	if err != nil {
		t.Fatal(err)
	}
	if versionStatus(t, ctx, db, firstReady.ID) != domain.GeoDataVersionSuperseded || versionStatus(t, ctx, db, secondReady.ID) != domain.GeoDataVersionReady {
		t.Fatal("district publication did not rotate READY version")
	}
	if !cityBoundaryValid(t, ctx, db, cityID) {
		t.Fatal("city boundary was not rebuilt from districts")
	}

	failed := beginImport(t, ctx, repository, cityCode, strings.Repeat("c", 64), "district-invalid")
	if _, err := repository.CompleteImport(ctx, failed.Version.ID, nil, domain.DistrictImportReport{}, nil); err == nil {
		t.Fatal("empty district publication unexpectedly succeeded")
	}
	if err := repository.FailImport(ctx, failed.Version.ID, domain.DistrictImportReport{Outcome: "failed"}, errors.New("invalid fixture")); err != nil {
		t.Fatal(err)
	}
	if versionStatus(t, ctx, db, failed.Version.ID) != domain.GeoDataVersionFailed || versionStatus(t, ctx, db, secondReady.ID) != domain.GeoDataVersionReady {
		t.Fatal("failed import affected the current district version")
	}
}

func testDistricts() []domain.DistrictDraft {
	return []domain.DistrictDraft{{
		ExternalID: "relation/1", Name: "Test District", Kind: "administrative_district",
		GeometryJSON: []byte(`{"type":"Polygon","coordinates":[[[30.30,59.93],[30.31,59.93],[30.31,59.94],[30.30,59.94],[30.30,59.93]]]}`),
		Attributes:   map[string]any{"adminLevel": 5},
	}}
}

func beginImport(t *testing.T, ctx context.Context, repository *Repository, cityCode, checksum, normalization string) domain.BeginDistrictImportResult {
	t.Helper()
	result, err := repository.BeginImport(ctx, domain.BeginDistrictImport{
		CityCode: cityCode, Source: "integration-test", SourceChecksum: checksum,
		SourceFileName: "districts.geojson", SourceSizeBytes: 1, NormalizationVersion: normalization,
	})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func createCity(t *testing.T, ctx context.Context, db *database.Pool, code string) string {
	t.Helper()
	tx, err := db.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var id string
	if err := tx.QueryRow(ctx, `INSERT INTO cities (code,name,country_code,timezone) VALUES ($1,'District Test','RU','Europe/Moscow') RETURNING id`, code).Scan(&id); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	return id
}

func deleteCity(t *testing.T, ctx context.Context, db *database.Pool, cityID string) {
	t.Helper()
	tx, err := db.Begin(ctx)
	if err != nil {
		t.Errorf("begin cleanup: %v", err)
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `DELETE FROM district_data_versions WHERE city_id=$1`, cityID); err != nil {
		t.Errorf("delete versions: %v", err)
		return
	}
	if _, err := tx.Exec(ctx, `DELETE FROM cities WHERE id=$1`, cityID); err != nil {
		t.Errorf("delete city: %v", err)
		return
	}
	if err := tx.Commit(ctx); err != nil {
		t.Errorf("commit cleanup: %v", err)
	}
}

func versionStatus(t *testing.T, ctx context.Context, db *database.Pool, versionID string) domain.GeoDataVersionStatus {
	t.Helper()
	tx, err := db.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var status domain.GeoDataVersionStatus
	if err := tx.QueryRow(ctx, `SELECT status FROM district_data_versions WHERE id=$1`, versionID).Scan(&status); err != nil {
		t.Fatal(err)
	}
	return status
}

func districtCount(t *testing.T, ctx context.Context, db *database.Pool, versionID string) int64 {
	t.Helper()
	tx, err := db.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var count int64
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM districts WHERE district_data_version_id=$1`, versionID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	return count
}

func cityBoundaryValid(t *testing.T, ctx context.Context, db *database.Pool, cityID string) bool {
	t.Helper()
	tx, err := db.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var valid bool
	if err := tx.QueryRow(ctx, `SELECT boundary IS NOT NULL AND ST_IsValid(boundary) FROM cities WHERE id=$1`, cityID).Scan(&valid); err != nil {
		t.Fatal(err)
	}
	return valid
}
