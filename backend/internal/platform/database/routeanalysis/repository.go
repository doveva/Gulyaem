package routeanalysis

import (
	"context"
	"encoding/json"
	"errors"
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
	ctx context.Context, cityID, geoDataVersionID string, route json.RawMessage, contextRadius float64,
) ([]geoanalysis.CandidateSegment, error) {
	tx, err := repository.database.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin route match candidate query: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	rows, err := tx.Query(ctx, `
		WITH route AS (SELECT ST_SetSRID(ST_GeomFromGeoJSON($3), 4326) AS geometry)
		SELECT ss.id, ST_AsGeoJSON(ss.geometry), ss.length_m, ss.classification,
		       ss.attributes, 0::double precision
		FROM street_segments ss
		CROSS JOIN route
		WHERE ss.city_id = $1
		  AND ss.geo_data_version_id = $2
		  AND ss.classification IN ('EXPLORE', 'ROUTABLE_ONLY')
		  AND ST_DWithin(ss.geometry::geography, route.geometry::geography, $4)
		ORDER BY ss.id
	`, cityID, geoDataVersionID, string(route), contextRadius)
	if err != nil {
		return nil, fmt.Errorf("query route match candidates: %w", err)
	}
	defer rows.Close()
	return scanSegments(rows)
}

func (repository *Repository) CoverageSegments(
	ctx context.Context, cityID, geoDataVersionID string,
	fragments []geoanalysis.NormalizedRouteFragment, radius, contextRadius float64,
) ([]geoanalysis.CandidateSegment, error) {
	contextRoute, err := fragmentGeometryJSON(fragments)
	if err != nil {
		return nil, err
	}
	tx, err := repository.database.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin route coverage query: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	rows, err := tx.Query(ctx, `
		WITH route AS (
			SELECT ST_SetSRID(ST_GeomFromGeoJSON($3), 4326) AS geometry
		)
		SELECT ss.id, ST_AsGeoJSON(ss.geometry), ss.length_m, ss.classification, ss.attributes,
		       0::double precision
		FROM street_segments ss
		CROSS JOIN route
		WHERE ss.city_id = $1
		  AND ss.geo_data_version_id = $2
		  AND ss.classification IN ('EXPLORE', 'ROUTABLE_ONLY')
		  AND ST_DWithin(ss.geometry::geography, route.geometry::geography, $4)
		ORDER BY ss.id
	`, cityID, geoDataVersionID, string(contextRoute), contextRadius)
	if err != nil {
		return nil, fmt.Errorf("query route coverage: %w", err)
	}
	candidates, err := scanSegments(rows)
	rows.Close()
	if err != nil {
		return nil, err
	}

	fragmentsByGrade := make(map[string][]geoanalysis.NormalizedRouteFragment)
	for _, fragment := range fragments {
		fragmentsByGrade[fragment.GradeSignature] = append(fragmentsByGrade[fragment.GradeSignature], fragment)
	}
	candidateIDsByGrade := make(map[string][]string)
	candidateByID := make(map[string]*geoanalysis.CandidateSegment, len(candidates))
	for index := range candidates {
		candidate := &candidates[index]
		candidateByID[candidate.ID] = candidate
		if candidate.Classification == domain.StreetSegmentExplore {
			grade := geoanalysis.GradeSignature(candidate.Attributes)
			candidateIDsByGrade[grade] = append(candidateIDsByGrade[grade], candidate.ID)
		}
	}
	for grade, gradeFragments := range fragmentsByGrade {
		candidateIDs := candidateIDsByGrade[grade]
		if len(candidateIDs) == 0 {
			continue
		}
		gradeRoute, encodeErr := fragmentGeometryJSON(gradeFragments)
		if encodeErr != nil {
			return nil, encodeErr
		}
		coverageRows, queryErr := tx.Query(ctx, `
			WITH route AS (
				SELECT ST_SetSRID(ST_GeomFromGeoJSON($3), 4326) AS geometry
			), halo AS (
				SELECT ST_Buffer(geometry::geography, $4)::geometry AS geometry FROM route
			)
			SELECT ss.id,
			       ST_Length(ST_CollectionExtract(ST_Intersection(ss.geometry, halo.geometry), 2)::geography)
			FROM street_segments ss
			CROSS JOIN halo
			WHERE ss.city_id = $1
			  AND ss.geo_data_version_id = $2
			  AND ss.classification = 'EXPLORE'
			  AND ss.id::text = ANY($5::text[])
			ORDER BY ss.id
		`, cityID, geoDataVersionID, string(gradeRoute), radius, candidateIDs)
		if queryErr != nil {
			return nil, fmt.Errorf("query route coverage for grade %q: %w", grade, queryErr)
		}
		for coverageRows.Next() {
			var segmentID string
			var coveredMeters float64
			if scanErr := coverageRows.Scan(&segmentID, &coveredMeters); scanErr != nil {
				coverageRows.Close()
				return nil, fmt.Errorf("scan route coverage for grade %q: %w", grade, scanErr)
			}
			if candidate := candidateByID[segmentID]; candidate != nil {
				candidate.RadiusCoveredMeters = coveredMeters
			}
		}
		if rowsErr := coverageRows.Err(); rowsErr != nil {
			coverageRows.Close()
			return nil, fmt.Errorf("iterate route coverage for grade %q: %w", grade, rowsErr)
		}
		coverageRows.Close()
	}
	return candidates, nil
}

func fragmentGeometryJSON(fragments []geoanalysis.NormalizedRouteFragment) (json.RawMessage, error) {
	coordinates := make([][][2]float64, 0, len(fragments))
	for _, fragment := range fragments {
		if len(fragment.Geometry) < 2 {
			continue
		}
		line := make([][2]float64, len(fragment.Geometry))
		for index, point := range fragment.Geometry {
			line[index] = [2]float64{point.Lon, point.Lat}
		}
		coordinates = append(coordinates, line)
	}
	if len(coordinates) == 0 {
		return nil, errors.New("route coverage requires at least one normalized fragment")
	}
	result, err := json.Marshal(struct {
		Type        string         `json:"type"`
		Coordinates [][][2]float64 `json:"coordinates"`
	}{Type: "MultiLineString", Coordinates: coordinates})
	if err != nil {
		return nil, fmt.Errorf("encode route coverage fragments: %w", err)
	}
	return result, nil
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
