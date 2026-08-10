package districtversion

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/doveva/Gulyaem/backend/internal/geo/districting"
	"github.com/doveva/Gulyaem/backend/internal/geo/domain"
	"github.com/doveva/Gulyaem/backend/internal/platform/database"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Repository struct{ database *database.Pool }

func New(database *database.Pool) *Repository { return &Repository{database: database} }

func (repository *Repository) BeginImport(ctx context.Context, input domain.BeginDistrictImport) (domain.BeginDistrictImportResult, error) {
	tx, err := repository.database.Begin(ctx)
	if err != nil {
		return domain.BeginDistrictImportResult{}, fmt.Errorf("begin district import transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var cityID string
	if err := tx.QueryRow(ctx, `SELECT id FROM cities WHERE code = $1`, input.CityCode).Scan(&cityID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.BeginDistrictImportResult{}, fmt.Errorf("%w: %s", districting.ErrCityNotFound, input.CityCode)
		}
		return domain.BeginDistrictImportResult{}, fmt.Errorf("select city: %w", err)
	}
	existing, err := findReady(ctx, tx, cityID, input.SourceChecksum, input.NormalizationVersion)
	if err == nil {
		if err := tx.Commit(ctx); err != nil {
			return domain.BeginDistrictImportResult{}, fmt.Errorf("commit idempotent district import lookup: %w", err)
		}
		return domain.BeginDistrictImportResult{Version: existing, AlreadyReady: true}, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return domain.BeginDistrictImportResult{}, fmt.Errorf("find existing district data version: %w", err)
	}
	version, err := insertImporting(ctx, tx, cityID, input)
	if err != nil {
		var postgresError *pgconn.PgError
		if errors.As(err, &postgresError) && postgresError.Code == "23505" {
			return domain.BeginDistrictImportResult{}, districting.ErrImportInProgress
		}
		return domain.BeginDistrictImportResult{}, fmt.Errorf("insert importing district data version: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.BeginDistrictImportResult{}, fmt.Errorf("commit importing district data version: %w", err)
	}
	return domain.BeginDistrictImportResult{Version: version}, nil
}

func (repository *Repository) CompleteImport(ctx context.Context, versionID string, sourceTimestamp *time.Time, report domain.DistrictImportReport, districts []domain.DistrictDraft) (domain.DistrictDataVersion, error) {
	if len(districts) == 0 {
		return domain.DistrictDataVersion{}, errors.New("publish districts: generated set is empty")
	}
	reportJSON, err := json.Marshal(report)
	if err != nil {
		return domain.DistrictDataVersion{}, fmt.Errorf("encode district import report: %w", err)
	}
	tx, err := repository.database.Begin(ctx)
	if err != nil {
		return domain.DistrictDataVersion{}, fmt.Errorf("begin district publish transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var cityID string
	var status domain.GeoDataVersionStatus
	if err := tx.QueryRow(ctx, `SELECT city_id, status FROM district_data_versions WHERE id = $1 FOR UPDATE`, versionID).Scan(&cityID, &status); err != nil {
		return domain.DistrictDataVersion{}, fmt.Errorf("lock district data version: %w", err)
	}
	if status != domain.GeoDataVersionImporting {
		return domain.DistrictDataVersion{}, fmt.Errorf("cannot publish district data version in status %s", status)
	}
	if err := insertDistricts(ctx, tx, versionID, cityID, districts); err != nil {
		return domain.DistrictDataVersion{}, err
	}
	if _, err := tx.Exec(ctx, `UPDATE district_data_versions SET status = 'SUPERSEDED' WHERE city_id = $1 AND status = 'READY'`, cityID); err != nil {
		return domain.DistrictDataVersion{}, fmt.Errorf("supersede current district data version: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE cities
		SET boundary = (
			SELECT ST_Multi(ST_UnaryUnion(ST_Collect(boundary)))
			FROM districts
			WHERE district_data_version_id = $1
		)
		WHERE id = $2
	`, versionID, cityID); err != nil {
		return domain.DistrictDataVersion{}, fmt.Errorf("update city boundary from districts: %w", err)
	}
	version, err := updateReady(ctx, tx, versionID, sourceTimestamp, reportJSON)
	if err != nil {
		return domain.DistrictDataVersion{}, fmt.Errorf("publish district data version: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.DistrictDataVersion{}, fmt.Errorf("commit published district data version: %w", err)
	}
	return version, nil
}

func insertDistricts(ctx context.Context, tx pgx.Tx, versionID, cityID string, districts []domain.DistrictDraft) error {
	if _, err := tx.Exec(ctx, `
		CREATE TEMP TABLE district_import (
			external_id text NOT NULL,
			name text NOT NULL,
			kind text NOT NULL,
			geometry_json text NOT NULL,
			attributes_json text NOT NULL
		) ON COMMIT DROP
	`); err != nil {
		return fmt.Errorf("create district import table: %w", err)
	}
	rows := make([][]any, 0, len(districts))
	for index, district := range districts {
		attributes, err := json.Marshal(district.Attributes)
		if err != nil {
			return fmt.Errorf("encode district %d attributes: %w", index, err)
		}
		rows = append(rows, []any{district.ExternalID, district.Name, district.Kind, string(district.GeometryJSON), string(attributes)})
	}
	copied, err := tx.CopyFrom(ctx, pgx.Identifier{"district_import"}, []string{"external_id", "name", "kind", "geometry_json", "attributes_json"}, pgx.CopyFromRows(rows))
	if err != nil {
		return fmt.Errorf("copy district import rows: %w", err)
	}
	if copied != int64(len(districts)) {
		return fmt.Errorf("copy district import rows: copied %d, want %d", copied, len(districts))
	}
	if _, err := tx.Exec(ctx, `
		WITH parsed AS (
			SELECT external_id, name, kind,
			       ST_Multi(ST_Force2D(ST_GeomFromGeoJSON(geometry_json)))::geometry(MultiPolygon, 4326) AS boundary,
			       attributes_json::jsonb AS attributes
			FROM district_import
		), valid AS (
			SELECT *, ST_IsValid(boundary) AND NOT ST_IsEmpty(boundary) AS is_valid
			FROM parsed
		)
		INSERT INTO districts (
			city_id, district_data_version_id, external_id, name, kind, boundary, label_point, attributes
		)
		SELECT $1, $2, external_id, name, kind, boundary,
		       ST_PointOnSurface(boundary)::geometry(Point, 4326), attributes
		FROM valid
		WHERE is_valid
	`, cityID, versionID); err != nil {
		return fmt.Errorf("insert districts: %w", err)
	}
	var published int64
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM districts WHERE district_data_version_id = $1`, versionID).Scan(&published); err != nil {
		return fmt.Errorf("validate published districts: %w", err)
	}
	if published != int64(len(districts)) {
		return fmt.Errorf("validate published districts: published=%d expected=%d", published, len(districts))
	}
	return nil
}

func (repository *Repository) FailImport(ctx context.Context, versionID string, report domain.DistrictImportReport, importError error) error {
	reportJSON, err := json.Marshal(report)
	if err != nil {
		return fmt.Errorf("encode failed district import report: %w", err)
	}
	message := importError.Error()
	if len(message) > 4000 {
		message = message[:4000]
	}
	tx, err := repository.database.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	result, err := tx.Exec(ctx, `
		UPDATE district_data_versions
		SET status = 'FAILED', import_finished_at = now(), import_report = $2, import_error = $3
		WHERE id = $1 AND status = 'IMPORTING'
	`, versionID, reportJSON, message)
	if err != nil {
		return fmt.Errorf("mark district data version failed: %w", err)
	}
	if result.RowsAffected() != 1 {
		return errors.New("mark district data version failed: version is not IMPORTING")
	}
	return tx.Commit(ctx)
}

func findReady(ctx context.Context, tx pgx.Tx, cityID, checksum, normalizationVersion string) (domain.DistrictDataVersion, error) {
	return scanVersion(tx.QueryRow(ctx, versionSelect+`
		WHERE ddv.city_id = $1 AND ddv.source_checksum = $2
		  AND ddv.normalization_version = $3 AND ddv.status = 'READY'
	`, cityID, checksum, normalizationVersion))
}

func insertImporting(ctx context.Context, tx pgx.Tx, cityID string, input domain.BeginDistrictImport) (domain.DistrictDataVersion, error) {
	return scanVersion(tx.QueryRow(ctx, `
		INSERT INTO district_data_versions (
			city_id, source, source_url, source_checksum, source_file_name,
			source_size_bytes, normalization_version, status
		) VALUES ($1, $2, NULLIF($3, ''), $4, $5, $6, $7, 'IMPORTING')
		RETURNING id, city_id, $8::text, source, COALESCE(source_url, ''), source_timestamp,
		          source_checksum, source_file_name, source_size_bytes, normalization_version,
		          status, import_started_at, import_finished_at, imported_at,
		          import_report, COALESCE(import_error, '')
	`, cityID, input.Source, input.SourceURL, input.SourceChecksum, input.SourceFileName,
		input.SourceSizeBytes, input.NormalizationVersion, input.CityCode))
}

func updateReady(ctx context.Context, tx pgx.Tx, versionID string, sourceTimestamp *time.Time, reportJSON []byte) (domain.DistrictDataVersion, error) {
	return scanVersion(tx.QueryRow(ctx, `
		UPDATE district_data_versions
		SET status = 'READY', source_timestamp = $2, import_finished_at = now(),
		    imported_at = now(), import_report = $3, import_error = NULL
		WHERE id = $1
		RETURNING id, city_id, (SELECT code FROM cities WHERE id = district_data_versions.city_id),
		          source, COALESCE(source_url, ''), source_timestamp, source_checksum,
		          source_file_name, source_size_bytes, normalization_version, status,
		          import_started_at, import_finished_at, imported_at,
		          import_report, COALESCE(import_error, '')
	`, versionID, sourceTimestamp, reportJSON))
}

const versionSelect = `
	SELECT ddv.id, ddv.city_id, c.code, ddv.source, COALESCE(ddv.source_url, ''),
	       ddv.source_timestamp, ddv.source_checksum, ddv.source_file_name,
	       ddv.source_size_bytes, ddv.normalization_version, ddv.status,
	       ddv.import_started_at, ddv.import_finished_at, ddv.imported_at,
	       ddv.import_report, COALESCE(ddv.import_error, '')
	FROM district_data_versions ddv
	JOIN cities c ON c.id = ddv.city_id
`

type rowScanner interface{ Scan(...any) error }

func scanVersion(row rowScanner) (domain.DistrictDataVersion, error) {
	var version domain.DistrictDataVersion
	var reportJSON []byte
	if err := row.Scan(
		&version.ID, &version.CityID, &version.CityCode, &version.Source, &version.SourceURL,
		&version.SourceTimestamp, &version.SourceChecksum, &version.SourceFileName,
		&version.SourceSizeBytes, &version.NormalizationVersion, &version.Status,
		&version.ImportStartedAt, &version.ImportFinishedAt, &version.ImportedAt,
		&reportJSON, &version.ImportError,
	); err != nil {
		return domain.DistrictDataVersion{}, err
	}
	if len(reportJSON) > 0 {
		if err := json.Unmarshal(reportJSON, &version.ImportReport); err != nil {
			return domain.DistrictDataVersion{}, fmt.Errorf("decode district import report: %w", err)
		}
	}
	return version, nil
}

var _ districting.VersionStore = (*Repository)(nil)
