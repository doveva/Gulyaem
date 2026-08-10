package routeanalysis

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/doveva/Gulyaem/backend/internal/geo/domain"
	"github.com/doveva/Gulyaem/backend/internal/geo/querying"
	geoanalysis "github.com/doveva/Gulyaem/backend/internal/geo/routeanalysis"
	"github.com/doveva/Gulyaem/backend/internal/platform/database"
	"github.com/doveva/Gulyaem/backend/internal/platform/database/geoquery"
	"github.com/jackc/pgx/v5"
)

type Repository struct {
	database *database.Pool
	geo      *geoquery.Repository
}

func New(database *database.Pool, geo *geoquery.Repository) *Repository {
	return &Repository{database: database, geo: geo}
}

func (repository *Repository) CurrentVersion(ctx context.Context, cityID string) (querying.Version, error) {
	return repository.geo.CurrentVersion(ctx, cityID)
}

func (repository *Repository) CandidateSegments(
	ctx context.Context, cityID string, route json.RawMessage, contextRadius float64,
) ([]geoanalysis.CandidateSegment, error) {
	tx, err := repository.database.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin route match candidate query: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	rows, err := tx.Query(ctx, `
		WITH route AS (SELECT ST_SetSRID(ST_GeomFromGeoJSON($2), 4326) AS geometry)
		SELECT ss.id, ST_AsGeoJSON(ss.geometry), ss.length_m, ss.classification,
		       ss.attributes, 0::double precision
		FROM street_segments ss
		JOIN geo_data_versions gdv ON gdv.id = ss.geo_data_version_id AND gdv.status = 'READY'
		CROSS JOIN route
		WHERE ss.city_id = $1
		  AND ss.classification IN ('EXPLORE', 'ROUTABLE_ONLY')
		  AND ST_DWithin(ss.geometry::geography, route.geometry::geography, $3)
		ORDER BY ss.id
	`, cityID, string(route), contextRadius)
	if err != nil {
		return nil, fmt.Errorf("query route match candidates: %w", err)
	}
	defer rows.Close()
	return scanSegments(rows)
}

func (repository *Repository) CoverageSegments(
	ctx context.Context, cityID string, route json.RawMessage, radius, contextRadius float64,
) ([]geoanalysis.CandidateSegment, error) {
	tx, err := repository.database.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin route coverage query: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	rows, err := tx.Query(ctx, `
		WITH route AS (
			SELECT ST_SetSRID(ST_GeomFromGeoJSON($2), 4326) AS geometry
		), halo AS (
			SELECT ST_Buffer(geometry::geography, $3)::geometry AS geometry FROM route
		)
		SELECT ss.id, ST_AsGeoJSON(ss.geometry), ss.length_m, ss.classification, ss.attributes,
		       CASE WHEN ss.classification = 'EXPLORE' THEN
		         ST_Length(ST_CollectionExtract(ST_Intersection(ss.geometry, halo.geometry), 2)::geography)
		       ELSE 0 END
		FROM street_segments ss
		JOIN geo_data_versions gdv ON gdv.id = ss.geo_data_version_id AND gdv.status = 'READY'
		CROSS JOIN route
		CROSS JOIN halo
		WHERE ss.city_id = $1
		  AND ss.classification IN ('EXPLORE', 'ROUTABLE_ONLY')
		  AND ST_DWithin(ss.geometry::geography, route.geometry::geography, $4)
		ORDER BY ss.id
	`, cityID, string(route), radius, contextRadius)
	if err != nil {
		return nil, fmt.Errorf("query route coverage: %w", err)
	}
	defer rows.Close()
	return scanSegments(rows)
}

func scanSegments(queryRows pgx.Rows) ([]geoanalysis.CandidateSegment, error) {
	result := make([]geoanalysis.CandidateSegment, 0)
	for queryRows.Next() {
		var segment geoanalysis.CandidateSegment
		var geometryBytes, attributesBytes []byte
		if err := queryRows.Scan(
			&segment.ID, &geometryBytes, &segment.LengthMeters, &segment.Classification,
			&attributesBytes, &segment.RadiusCoveredMeters,
		); err != nil {
			return nil, fmt.Errorf("scan route analysis segment: %w", err)
		}
		var geometry struct {
			Coordinates [][]float64 `json:"coordinates"`
		}
		if err := json.Unmarshal(geometryBytes, &geometry); err != nil {
			return nil, fmt.Errorf("decode route analysis geometry: %w", err)
		}
		segment.Geometry = make([]domain.Point, len(geometry.Coordinates))
		for index, coordinate := range geometry.Coordinates {
			segment.Geometry[index] = domain.Point{Lon: coordinate[0], Lat: coordinate[1]}
		}
		segment.GeometryJSON = json.RawMessage(geometryBytes)
		if err := json.Unmarshal(attributesBytes, &segment.Attributes); err != nil {
			return nil, fmt.Errorf("decode route analysis attributes: %w", err)
		}
		segment.ReasonCode = segment.Attributes.ReasonCode
		result = append(result, segment)
	}
	if err := queryRows.Err(); err != nil {
		return nil, fmt.Errorf("iterate route analysis segments: %w", err)
	}
	return result, nil
}

var _ geoanalysis.Repository = (*Repository)(nil)
