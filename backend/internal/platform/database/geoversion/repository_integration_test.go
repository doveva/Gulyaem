package geoversion

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

	cityCode := fmt.Sprintf("integration-%d", time.Now().UnixNano())
	cityID := createTestCity(t, ctx, db, cityCode)
	defer deleteTestCity(t, ctx, db, cityID)

	repository := New(db)
	first := beginTestImport(t, ctx, repository, cityCode, strings.Repeat("a", 64), "integration-v1")
	firstReady, err := repository.CompleteImport(ctx, first.Version.ID, nil, domain.ImportReport{Outcome: "imported", ObjectsProcessed: 10})
	if err != nil {
		t.Fatal(err)
	}
	if firstReady.Status != domain.GeoDataVersionReady {
		t.Fatalf("first status = %s", firstReady.Status)
	}

	idempotent := beginTestImport(t, ctx, repository, cityCode, strings.Repeat("a", 64), "integration-v1")
	if !idempotent.AlreadyReady || idempotent.Version.ID != firstReady.ID {
		t.Fatalf("idempotent result = %+v", idempotent)
	}

	second := beginTestImport(t, ctx, repository, cityCode, strings.Repeat("b", 64), "integration-v2")
	secondReady, err := repository.CompleteImport(ctx, second.Version.ID, nil, domain.ImportReport{Outcome: "imported", ObjectsProcessed: 11})
	if err != nil {
		t.Fatal(err)
	}
	if got := versionStatus(t, ctx, db, firstReady.ID); got != domain.GeoDataVersionSuperseded {
		t.Fatalf("first status after second publish = %s", got)
	}
	if got := versionStatus(t, ctx, db, secondReady.ID); got != domain.GeoDataVersionReady {
		t.Fatalf("second status = %s", got)
	}

	failed := beginTestImport(t, ctx, repository, cityCode, strings.Repeat("c", 64), "integration-invalid")
	if err := repository.FailImport(ctx, failed.Version.ID, domain.ImportReport{Outcome: "failed"}, errors.New("invalid fixture")); err != nil {
		t.Fatal(err)
	}
	if got := versionStatus(t, ctx, db, failed.Version.ID); got != domain.GeoDataVersionFailed {
		t.Fatalf("failed status = %s", got)
	}
	if got := versionStatus(t, ctx, db, secondReady.ID); got != domain.GeoDataVersionReady {
		t.Fatalf("failed import changed current READY status to %s", got)
	}
}

func beginTestImport(t *testing.T, ctx context.Context, repository *Repository, cityCode, checksum, normalization string) domain.BeginImportResult {
	t.Helper()
	result, err := repository.BeginImport(ctx, domain.BeginImport{
		CityCode:             cityCode,
		Source:               "integration-test",
		SourceChecksum:       checksum,
		SourceFileName:       "fixture.osm.pbf",
		SourceSizeBytes:      1,
		NormalizationVersion: normalization,
	})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func createTestCity(t *testing.T, ctx context.Context, db *database.Pool, code string) string {
	t.Helper()
	tx, err := db.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var id string
	if err := tx.QueryRow(ctx, `
		INSERT INTO cities (code, name, country_code, timezone)
		VALUES ($1, 'Integration Test City', 'RU', 'Europe/Moscow')
		RETURNING id
	`, code).Scan(&id); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	return id
}

func deleteTestCity(t *testing.T, ctx context.Context, db *database.Pool, cityID string) {
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

func versionStatus(t *testing.T, ctx context.Context, db *database.Pool, versionID string) domain.GeoDataVersionStatus {
	t.Helper()
	tx, err := db.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var status domain.GeoDataVersionStatus
	if err := tx.QueryRow(ctx, `SELECT status FROM geo_data_versions WHERE id = $1`, versionID).Scan(&status); err != nil {
		t.Fatal(err)
	}
	return status
}
