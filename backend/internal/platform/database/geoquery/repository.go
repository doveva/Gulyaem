package geoquery

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/doveva/Gulyaem/backend/internal/geo/querying"
	"github.com/doveva/Gulyaem/backend/internal/platform/database"
	"github.com/jackc/pgx/v5"
)

type Repository struct {
	database *database.Pool
}

func New(database *database.Pool) *Repository {
	return &Repository{database: database}
}

func (repository *Repository) CurrentVersion(ctx context.Context, cityID string) (querying.Version, error) {
	tx, err := repository.database.Begin(ctx)
	if err != nil {
		return querying.Version{}, fmt.Errorf("begin current geo version query: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	row := tx.QueryRow(ctx, `
		SELECT id, city_id, source, source_timestamp, source_checksum,
		       normalization_version, status, imported_at, import_report
		FROM geo_data_versions
		WHERE city_id = $1 AND status = 'READY'
	`, cityID)
	version, err := scanVersion(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return querying.Version{}, querying.ErrNotFound
	}
	if err != nil {
		return querying.Version{}, fmt.Errorf("query current geo version: %w", err)
	}
	return version, nil
}

func (repository *Repository) Segments(ctx context.Context, filter querying.SegmentFilter, limit int) ([]querying.Segment, error) {
	tx, err := repository.database.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin street segment query: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	classifications := make([]string, len(filter.Classifications))
	for index, classification := range filter.Classifications {
		classifications[index] = string(classification)
	}
	var minimum any
	if filter.MinLength != nil {
		minimum = *filter.MinLength
	}
	var maximum any
	if filter.MaxLength != nil {
		maximum = *filter.MaxLength
	}

	rows, err := tx.Query(ctx, `
		SELECT ss.id, ss.city_id, ss.geo_data_version_id,
		       ST_AsGeoJSON(ss.geometry), ss.length_m, ss.classification,
		       ss.attributes, ss.street_id, st.name,
		       gdv.status, gdv.normalization_version, true
		FROM street_segments ss
		JOIN geo_data_versions gdv ON gdv.id = ss.geo_data_version_id
		LEFT JOIN streets st ON st.id = ss.street_id
		WHERE ss.city_id = $1
		  AND gdv.status = 'READY'
		  AND ss.geometry && ST_MakeEnvelope($2, $3, $4, $5, 4326)
		  AND ST_Intersects(ss.geometry, ST_MakeEnvelope($2, $3, $4, $5, 4326))
		  AND (cardinality($6::text[]) = 0 OR ss.classification::text = ANY($6::text[]))
		  AND ($7::double precision IS NULL OR ss.length_m >= $7)
		  AND ($8::double precision IS NULL OR ss.length_m <= $8)
		ORDER BY ss.id
		LIMIT $9
	`, filter.CityID, filter.BBox.West, filter.BBox.South, filter.BBox.East, filter.BBox.North,
		classifications, minimum, maximum, limit)
	if err != nil {
		return nil, fmt.Errorf("query street segments: %w", err)
	}
	defer rows.Close()

	segments := make([]querying.Segment, 0)
	for rows.Next() {
		segment, err := scanSegment(rows)
		if err != nil {
			return nil, fmt.Errorf("scan street segment: %w", err)
		}
		segments = append(segments, segment)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate street segments: %w", err)
	}
	return segments, nil
}

func (repository *Repository) Segment(ctx context.Context, segmentID string) (querying.Segment, error) {
	tx, err := repository.database.Begin(ctx)
	if err != nil {
		return querying.Segment{}, fmt.Errorf("begin street segment detail query: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	row := tx.QueryRow(ctx, `
		SELECT ss.id, ss.city_id, ss.geo_data_version_id,
		       ST_AsGeoJSON(ss.geometry), ss.length_m, ss.classification,
		       ss.attributes, ss.street_id, st.name,
		       gdv.status, gdv.normalization_version, gdv.status = 'READY'
		FROM street_segments ss
		JOIN geo_data_versions gdv ON gdv.id = ss.geo_data_version_id
		LEFT JOIN streets st ON st.id = ss.street_id
		WHERE ss.id = $1
	`, segmentID)
	segment, err := scanSegment(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return querying.Segment{}, querying.ErrNotFound
	}
	if err != nil {
		return querying.Segment{}, fmt.Errorf("query street segment detail: %w", err)
	}
	return segment, nil
}

func (repository *Repository) CurrentDistrictVersion(ctx context.Context, cityID string) (querying.DistrictVersion, error) {
	tx, err := repository.database.Begin(ctx)
	if err != nil {
		return querying.DistrictVersion{}, fmt.Errorf("begin current district version query: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var version querying.DistrictVersion
	err = tx.QueryRow(ctx, `
		SELECT id, city_id, source, source_timestamp, source_checksum,
		       normalization_version, status, imported_at
		FROM district_data_versions
		WHERE city_id = $1 AND status = 'READY'
	`, cityID).Scan(
		&version.ID, &version.CityID, &version.Source, &version.SourceTimestamp,
		&version.SourceChecksum, &version.NormalizationVersion, &version.Status, &version.ImportedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return querying.DistrictVersion{}, querying.ErrNotFound
	}
	if err != nil {
		return querying.DistrictVersion{}, fmt.Errorf("query current district version: %w", err)
	}
	return version, nil
}

func (repository *Repository) Districts(ctx context.Context, filter querying.DistrictFilter) ([]querying.District, error) {
	tx, err := repository.database.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin district bbox query: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	rows, err := tx.Query(ctx, `
		SELECT d.id, d.city_id, d.district_data_version_id, d.external_id,
		       d.name, d.kind,
		       ST_AsGeoJSON(ST_SimplifyPreserveTopology(d.boundary, 0.00005)),
		       ST_AsGeoJSON(d.label_point), d.attributes
		FROM districts d
		JOIN district_data_versions ddv ON ddv.id = d.district_data_version_id
		WHERE d.city_id = $1 AND ddv.status = 'READY'
		  AND d.boundary && ST_MakeEnvelope($2, $3, $4, $5, 4326)
		  AND ST_Intersects(d.boundary, ST_MakeEnvelope($2, $3, $4, $5, 4326))
		ORDER BY d.name
	`, filter.CityID, filter.BBox.West, filter.BBox.South, filter.BBox.East, filter.BBox.North)
	if err != nil {
		return nil, fmt.Errorf("query districts: %w", err)
	}
	defer rows.Close()
	result := make([]querying.District, 0)
	for rows.Next() {
		var district querying.District
		var geometry, label string
		var attributes []byte
		if err := rows.Scan(
			&district.ID, &district.CityID, &district.DistrictDataVersionID,
			&district.ExternalID, &district.Name, &district.Kind,
			&geometry, &label, &attributes,
		); err != nil {
			return nil, fmt.Errorf("scan district: %w", err)
		}
		district.GeometryJSON = json.RawMessage(geometry)
		district.LabelPointJSON = json.RawMessage(label)
		if err := json.Unmarshal(attributes, &district.Attributes); err != nil {
			return nil, fmt.Errorf("decode district attributes: %w", err)
		}
		result = append(result, district)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate districts: %w", err)
	}
	return result, nil
}

func (repository *Repository) SegmentDistricts(ctx context.Context, segmentID string) ([]querying.DistrictSummary, error) {
	tx, err := repository.database.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin segment district query: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	rows, err := tx.Query(ctx, `
		SELECT d.id, d.district_data_version_id, d.name, d.kind
		FROM street_segments ss
		JOIN districts d ON d.city_id = ss.city_id AND ST_Intersects(d.boundary, ss.geometry)
		JOIN district_data_versions ddv ON ddv.id = d.district_data_version_id AND ddv.status = 'READY'
		WHERE ss.id = $1
		ORDER BY d.name
	`, segmentID)
	if err != nil {
		return nil, fmt.Errorf("query segment districts: %w", err)
	}
	defer rows.Close()
	result := make([]querying.DistrictSummary, 0)
	for rows.Next() {
		var district querying.DistrictSummary
		if err := rows.Scan(&district.ID, &district.DistrictDataVersionID, &district.Name, &district.Kind); err != nil {
			return nil, fmt.Errorf("scan segment district: %w", err)
		}
		result = append(result, district)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate segment districts: %w", err)
	}
	return result, nil
}

type rowScanner interface {
	Scan(...any) error
}

func scanVersion(row rowScanner) (querying.Version, error) {
	var version querying.Version
	var reportJSON []byte
	if err := row.Scan(
		&version.ID,
		&version.CityID,
		&version.Source,
		&version.SourceTimestamp,
		&version.SourceChecksum,
		&version.NormalizationVersion,
		&version.Status,
		&version.ImportedAt,
		&reportJSON,
	); err != nil {
		return querying.Version{}, err
	}
	if len(reportJSON) > 0 {
		if err := json.Unmarshal(reportJSON, &version.ImportReport); err != nil {
			return querying.Version{}, fmt.Errorf("decode import report: %w", err)
		}
	}
	return version, nil
}

func scanSegment(row rowScanner) (querying.Segment, error) {
	var segment querying.Segment
	var geometryJSON string
	var attributesJSON []byte
	if err := row.Scan(
		&segment.ID,
		&segment.CityID,
		&segment.GeoDataVersionID,
		&geometryJSON,
		&segment.LengthMeters,
		&segment.Classification,
		&attributesJSON,
		&segment.StreetID,
		&segment.StreetName,
		&segment.VersionStatus,
		&segment.NormalizationVersion,
		&segment.IsCurrent,
	); err != nil {
		return querying.Segment{}, err
	}
	segment.GeometryJSON = json.RawMessage(geometryJSON)
	if len(attributesJSON) > 0 {
		if err := json.Unmarshal(attributesJSON, &segment.Attributes); err != nil {
			return querying.Segment{}, fmt.Errorf("decode street segment attributes: %w", err)
		}
	}
	return segment, nil
}

var _ querying.Repository = (*Repository)(nil)
