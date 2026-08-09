package geoversion

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/doveva/Gulyaem/backend/internal/geo/domain"
	"github.com/doveva/Gulyaem/backend/internal/geo/importing"
	"github.com/doveva/Gulyaem/backend/internal/platform/database"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Repository struct {
	database *database.Pool
}

func New(database *database.Pool) *Repository {
	return &Repository{database: database}
}

func (repository *Repository) BeginImport(ctx context.Context, input domain.BeginImport) (domain.BeginImportResult, error) {
	tx, err := repository.database.Begin(ctx)
	if err != nil {
		return domain.BeginImportResult{}, fmt.Errorf("begin geo import transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var cityID string
	if err := tx.QueryRow(ctx, `SELECT id FROM cities WHERE code = $1`, input.CityCode).Scan(&cityID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.BeginImportResult{}, fmt.Errorf("%w: %s", importing.ErrCityNotFound, input.CityCode)
		}
		return domain.BeginImportResult{}, fmt.Errorf("select city: %w", err)
	}

	existing, err := findReady(ctx, tx, cityID, input.SourceChecksum, input.NormalizationVersion)
	if err == nil {
		if err := tx.Commit(ctx); err != nil {
			return domain.BeginImportResult{}, fmt.Errorf("commit idempotent import lookup: %w", err)
		}
		return domain.BeginImportResult{Version: existing, AlreadyReady: true}, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return domain.BeginImportResult{}, fmt.Errorf("find existing geo data version: %w", err)
	}

	version, err := insertImporting(ctx, tx, cityID, input)
	if err != nil {
		var postgresError *pgconn.PgError
		if errors.As(err, &postgresError) && postgresError.Code == "23505" {
			return domain.BeginImportResult{}, importing.ErrImportInProgress
		}
		return domain.BeginImportResult{}, fmt.Errorf("insert importing geo data version: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.BeginImportResult{}, fmt.Errorf("commit importing geo data version: %w", err)
	}
	return domain.BeginImportResult{Version: version}, nil
}

func (repository *Repository) CompleteImport(
	ctx context.Context,
	versionID string,
	sourceTimestamp *time.Time,
	report domain.ImportReport,
	segments []domain.StreetSegmentDraft,
) (domain.GeoDataVersion, error) {
	reportJSON, err := json.Marshal(report)
	if err != nil {
		return domain.GeoDataVersion{}, fmt.Errorf("encode import report: %w", err)
	}

	tx, err := repository.database.Begin(ctx)
	if err != nil {
		return domain.GeoDataVersion{}, fmt.Errorf("begin publish transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var cityID string
	var status domain.GeoDataVersionStatus
	if err := tx.QueryRow(ctx, `
		SELECT city_id, status
		FROM geo_data_versions
		WHERE id = $1
		FOR UPDATE
	`, versionID).Scan(&cityID, &status); err != nil {
		return domain.GeoDataVersion{}, fmt.Errorf("lock geo data version: %w", err)
	}
	if status != domain.GeoDataVersionImporting {
		return domain.GeoDataVersion{}, fmt.Errorf("cannot publish geo data version in status %s", status)
	}
	if err := insertStreetSegments(ctx, tx, versionID, cityID, segments); err != nil {
		return domain.GeoDataVersion{}, err
	}

	if _, err := tx.Exec(ctx, `
		UPDATE geo_data_versions
		SET status = 'SUPERSEDED'
		WHERE city_id = $1 AND status = 'READY'
	`, cityID); err != nil {
		return domain.GeoDataVersion{}, fmt.Errorf("supersede current geo data version: %w", err)
	}

	version, err := updateReady(ctx, tx, versionID, sourceTimestamp, reportJSON)
	if err != nil {
		return domain.GeoDataVersion{}, fmt.Errorf("publish geo data version: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.GeoDataVersion{}, fmt.Errorf("commit published geo data version: %w", err)
	}
	return version, nil
}

func insertStreetSegments(ctx context.Context, tx pgx.Tx, versionID, cityID string, segments []domain.StreetSegmentDraft) error {
	if len(segments) == 0 {
		return errors.New("publish street segments: generated set is empty")
	}
	if _, err := tx.Exec(ctx, `
		CREATE TEMP TABLE geo_segment_import (
			geometry_wkt text NOT NULL,
			length_m double precision NOT NULL,
			classification text NOT NULL,
			attributes_json text NOT NULL
		) ON COMMIT DROP
	`); err != nil {
		return fmt.Errorf("create street segment import table: %w", err)
	}

	rows := make([][]any, 0, len(segments))
	for index, segment := range segments {
		if err := validateSegmentDraft(segment); err != nil {
			return fmt.Errorf("validate street segment %d: %w", index, err)
		}
		attributesJSON, err := json.Marshal(segment.Attributes)
		if err != nil {
			return fmt.Errorf("encode street segment %d attributes: %w", index, err)
		}
		rows = append(rows, []any{
			lineStringWKT(segment.Geometry),
			segment.LengthMeters,
			string(segment.Classification),
			string(attributesJSON),
		})
	}

	copied, err := tx.CopyFrom(
		ctx,
		pgx.Identifier{"geo_segment_import"},
		[]string{"geometry_wkt", "length_m", "classification", "attributes_json"},
		pgx.CopyFromRows(rows),
	)
	if err != nil {
		return fmt.Errorf("copy street segment import rows: %w", err)
	}
	if copied != int64(len(segments)) {
		return fmt.Errorf("copy street segment import rows: copied %d, want %d", copied, len(segments))
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO street_segments (
			city_id, geo_data_version_id, geometry, length_m, classification, attributes
		)
		SELECT $1, $2, ST_GeomFromText(geometry_wkt, 4326), length_m,
		       classification::street_segment_classification, attributes_json::jsonb
		FROM geo_segment_import
	`, cityID, versionID); err != nil {
		return fmt.Errorf("insert street segments: %w", err)
	}

	var published int64
	var invalid int64
	if err := tx.QueryRow(ctx, `
		SELECT count(*), count(*) FILTER (
			WHERE ST_IsEmpty(geometry)
			   OR NOT ST_IsValid(geometry)
			   OR ST_NPoints(geometry) < 2
			   OR length_m <= 0
			   OR ST_Length(geometry::geography) <= 0
		)
		FROM street_segments
		WHERE geo_data_version_id = $1
	`, versionID).Scan(&published, &invalid); err != nil {
		return fmt.Errorf("validate published street segments: %w", err)
	}
	if published != int64(len(segments)) || invalid != 0 {
		return fmt.Errorf("validate published street segments: published=%d expected=%d invalid=%d", published, len(segments), invalid)
	}
	return nil
}

func validateSegmentDraft(segment domain.StreetSegmentDraft) error {
	if len(segment.Geometry) < 2 {
		return errors.New("geometry must contain at least two points")
	}
	if segment.LengthMeters <= 0 || math.IsNaN(segment.LengthMeters) || math.IsInf(segment.LengthMeters, 0) {
		return errors.New("length must be finite and positive")
	}
	switch segment.Classification {
	case domain.StreetSegmentExplore, domain.StreetSegmentRoutableOnly, domain.StreetSegmentIgnore:
	default:
		return fmt.Errorf("unsupported classification %q", segment.Classification)
	}
	for _, point := range segment.Geometry {
		if point.Lon < -180 || point.Lon > 180 || point.Lat < -90 || point.Lat > 90 ||
			math.IsNaN(point.Lon) || math.IsNaN(point.Lat) || math.IsInf(point.Lon, 0) || math.IsInf(point.Lat, 0) {
			return errors.New("geometry contains an invalid coordinate")
		}
	}
	return nil
}

func lineStringWKT(points []domain.Point) string {
	coordinates := make([]string, len(points))
	for index, point := range points {
		coordinates[index] = strconv.FormatFloat(point.Lon, 'f', -1, 64) + " " + strconv.FormatFloat(point.Lat, 'f', -1, 64)
	}
	return "LINESTRING(" + strings.Join(coordinates, ",") + ")"
}

func (repository *Repository) FailImport(ctx context.Context, versionID string, report domain.ImportReport, importError error) error {
	reportJSON, err := json.Marshal(report)
	if err != nil {
		return fmt.Errorf("encode failed import report: %w", err)
	}
	errorMessage := importError.Error()
	if len(errorMessage) > 4000 {
		errorMessage = errorMessage[:4000]
	}

	result, err := repository.databaseExec(ctx, `
		UPDATE geo_data_versions
		SET status = 'FAILED',
		    import_finished_at = now(),
		    import_report = $2,
		    import_error = $3
		WHERE id = $1 AND status = 'IMPORTING'
	`, versionID, reportJSON, errorMessage)
	if err != nil {
		return fmt.Errorf("mark geo data version failed: %w", err)
	}
	if result.RowsAffected() != 1 {
		return fmt.Errorf("mark geo data version failed: version is not IMPORTING")
	}
	return nil
}

func (repository *Repository) databaseExec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	tx, err := repository.database.Begin(ctx)
	if err != nil {
		return pgconn.CommandTag{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	result, err := tx.Exec(ctx, sql, arguments...)
	if err != nil {
		return pgconn.CommandTag{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return pgconn.CommandTag{}, err
	}
	return result, nil
}

func findReady(ctx context.Context, tx pgx.Tx, cityID, checksum, normalizationVersion string) (domain.GeoDataVersion, error) {
	row := tx.QueryRow(ctx, versionSelect+`
		WHERE gdv.city_id = $1
		  AND gdv.source_checksum = $2
		  AND gdv.normalization_version = $3
		  AND gdv.status = 'READY'
	`, cityID, checksum, normalizationVersion)
	return scanVersion(row)
}

func insertImporting(ctx context.Context, tx pgx.Tx, cityID string, input domain.BeginImport) (domain.GeoDataVersion, error) {
	row := tx.QueryRow(ctx, `
		INSERT INTO geo_data_versions (
			city_id, source, source_url, source_checksum, source_file_name,
			source_size_bytes, normalization_version, status
		)
		VALUES ($1, $2, NULLIF($3, ''), $4, $5, $6, $7, 'IMPORTING')
		RETURNING id, city_id, $8::text, source, COALESCE(source_url, ''), source_timestamp,
		          source_checksum, source_file_name, source_size_bytes, normalization_version,
		          status, import_started_at, import_finished_at, imported_at,
		          import_report, COALESCE(import_error, '')
	`, cityID, input.Source, input.SourceURL, input.SourceChecksum, input.SourceFileName,
		input.SourceSizeBytes, input.NormalizationVersion, input.CityCode)
	return scanVersion(row)
}

func updateReady(ctx context.Context, tx pgx.Tx, versionID string, sourceTimestamp *time.Time, reportJSON []byte) (domain.GeoDataVersion, error) {
	row := tx.QueryRow(ctx, `
		UPDATE geo_data_versions
		SET status = 'READY',
		    source_timestamp = $2,
		    import_finished_at = now(),
		    imported_at = now(),
		    import_report = $3,
		    import_error = NULL
		WHERE id = $1
		RETURNING id, city_id,
		          (SELECT code FROM cities WHERE id = geo_data_versions.city_id),
		          source, COALESCE(source_url, ''), source_timestamp, source_checksum,
		          source_file_name, source_size_bytes, normalization_version, status,
		          import_started_at, import_finished_at, imported_at,
		          import_report, COALESCE(import_error, '')
	`, versionID, sourceTimestamp, reportJSON)
	return scanVersion(row)
}

const versionSelect = `
	SELECT gdv.id, gdv.city_id, c.code, gdv.source, COALESCE(gdv.source_url, ''),
	       gdv.source_timestamp, gdv.source_checksum, gdv.source_file_name,
	       gdv.source_size_bytes, gdv.normalization_version, gdv.status,
	       gdv.import_started_at, gdv.import_finished_at, gdv.imported_at,
	       gdv.import_report, COALESCE(gdv.import_error, '')
	FROM geo_data_versions gdv
	JOIN cities c ON c.id = gdv.city_id
`

type rowScanner interface {
	Scan(...any) error
}

func scanVersion(row rowScanner) (domain.GeoDataVersion, error) {
	var version domain.GeoDataVersion
	var reportJSON []byte
	err := row.Scan(
		&version.ID,
		&version.CityID,
		&version.CityCode,
		&version.Source,
		&version.SourceURL,
		&version.SourceTimestamp,
		&version.SourceChecksum,
		&version.SourceFileName,
		&version.SourceSizeBytes,
		&version.NormalizationVersion,
		&version.Status,
		&version.ImportStartedAt,
		&version.ImportFinishedAt,
		&version.ImportedAt,
		&reportJSON,
		&version.ImportError,
	)
	if err != nil {
		return domain.GeoDataVersion{}, err
	}
	if len(reportJSON) > 0 {
		if err := json.Unmarshal(reportJSON, &version.ImportReport); err != nil {
			return domain.GeoDataVersion{}, fmt.Errorf("decode import report: %w", err)
		}
	}
	return version, nil
}
