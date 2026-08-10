package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/doveva/Gulyaem/backend/internal/geo/domain"
	"github.com/doveva/Gulyaem/backend/internal/geo/querying"
	"github.com/go-chi/chi/v5"
)

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-8][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`)
var routeIDPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

type apiErrorBody struct {
	Error apiError `json:"error"`
}

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type geoVersionResponse struct {
	ID                   string                      `json:"id"`
	CityID               string                      `json:"cityId"`
	Source               string                      `json:"source"`
	SourceTimestamp      *time.Time                  `json:"sourceTimestamp"`
	SourceChecksum       string                      `json:"sourceChecksum"`
	NormalizationVersion string                      `json:"normalizationVersion"`
	Status               domain.GeoDataVersionStatus `json:"status"`
	ImportedAt           *time.Time                  `json:"importedAt"`
	ImportReport         domain.ImportReport         `json:"importReport"`
}

type geoJSONFeatureCollection struct {
	Type     string                `json:"type"`
	Features []geoJSONFeature      `json:"features"`
	Meta     segmentCollectionMeta `json:"meta"`
}

type geoJSONFeature struct {
	Type       string                   `json:"type"`
	ID         string                   `json:"id"`
	Geometry   json.RawMessage          `json:"geometry"`
	Properties geoJSONFeatureProperties `json:"properties"`
}

type geoJSONFeatureProperties struct {
	ID               string                             `json:"id"`
	GeoDataVersionID string                             `json:"geoDataVersionId"`
	Classification   domain.StreetSegmentClassification `json:"classification"`
	LengthMeters     float64                            `json:"lengthMeters"`
	StreetName       *string                            `json:"streetName"`
	ReasonCode       string                             `json:"reasonCode"`
	BoundaryClip     bool                               `json:"boundaryClip"`
}

type segmentCollectionMeta struct {
	GeoDataVersionID string              `json:"geoDataVersionId"`
	ReturnedCount    int                 `json:"returnedCount"`
	BBox             [4]float64          `json:"bbox"`
	Statistics       querying.Statistics `json:"statistics"`
}

type segmentDetailResponse struct {
	ID                   string                             `json:"id"`
	CityID               string                             `json:"cityId"`
	GeoDataVersionID     string                             `json:"geoDataVersionId"`
	VersionStatus        domain.GeoDataVersionStatus        `json:"versionStatus"`
	NormalizationVersion string                             `json:"normalizationVersion"`
	IsCurrent            bool                               `json:"isCurrent"`
	Geometry             json.RawMessage                    `json:"geometry"`
	LengthMeters         float64                            `json:"lengthMeters"`
	Classification       domain.StreetSegmentClassification `json:"classification"`
	ReasonCode           string                             `json:"reasonCode"`
	NormalizedAttributes map[string]any                     `json:"normalizedAttributes"`
	Street               *segmentStreetResponse             `json:"street"`
	Districts            []querying.DistrictSummary         `json:"districts"`
	DebugSource          *segmentDebugSource                `json:"debugSource,omitempty"`
}

type districtFeatureCollection struct {
	Type     string            `json:"type"`
	Features []districtFeature `json:"features"`
	Meta     districtMeta      `json:"meta"`
}

type districtFeature struct {
	Type       string             `json:"type"`
	ID         string             `json:"id"`
	Geometry   json.RawMessage    `json:"geometry"`
	Properties districtProperties `json:"properties"`
}

type districtProperties struct {
	ID                    string          `json:"id"`
	DistrictDataVersionID string          `json:"districtDataVersionId"`
	ExternalID            string          `json:"externalId"`
	Name                  string          `json:"name"`
	Kind                  string          `json:"kind"`
	LabelPoint            json.RawMessage `json:"labelPoint"`
	Source                string          `json:"source"`
	SourceTimestamp       *time.Time      `json:"sourceTimestamp"`
	NormalizationVersion  string          `json:"normalizationVersion"`
}

type districtMeta struct {
	DistrictDataVersionID string     `json:"districtDataVersionId"`
	ReturnedCount         int        `json:"returnedCount"`
	BBox                  [4]float64 `json:"bbox"`
}

type segmentStreetResponse struct {
	ID   string  `json:"id"`
	Name *string `json:"name"`
}

type segmentDebugSource struct {
	Tags        map[string]string `json:"tags,omitempty"`
	WayIDs      []int64           `json:"wayIds,omitempty"`
	StartNodeID *int64            `json:"startNodeId,omitempty"`
	EndNodeID   *int64            `json:"endNodeId,omitempty"`
}

func registerGeoRoutes(router chi.Router, deps Dependencies) {
	if deps.Geo == nil {
		return
	}
	router.Get("/api/v1/cities/{cityId}/geo-version", currentGeoVersionHandler(deps))
	router.Get("/api/v1/geo/segments", segmentsHandler(deps))
	router.Get("/api/v1/geo/segments/{segmentId}", segmentDetailHandler(deps))
	router.Get("/api/v1/geo/districts", districtsHandler(deps))
}

func districtsHandler(deps Dependencies) http.HandlerFunc {
	return func(response http.ResponseWriter, request *http.Request) {
		query := request.URL.Query()
		cityID := strings.TrimSpace(query.Get("cityId"))
		if !uuidPattern.MatchString(cityID) {
			writeAPIError(response, http.StatusBadRequest, "invalid_query", "cityId must be a UUID")
			return
		}
		bbox, err := parseBBox(query.Get("bbox"))
		if err != nil {
			writeAPIError(response, http.StatusBadRequest, "invalid_query", err.Error())
			return
		}
		collection, err := deps.Geo.Districts(request.Context(), querying.DistrictFilter{CityID: cityID, BBox: bbox})
		if err != nil {
			handleGeoError(response, deps, err)
			return
		}
		features := make([]districtFeature, len(collection.Districts))
		for index, district := range collection.Districts {
			features[index] = districtFeature{
				Type: "Feature", ID: district.ID, Geometry: district.GeometryJSON,
				Properties: districtProperties{
					ID: district.ID, DistrictDataVersionID: district.DistrictDataVersionID,
					ExternalID: district.ExternalID, Name: district.Name, Kind: district.Kind,
					LabelPoint: district.LabelPointJSON, Source: collection.Version.Source,
					SourceTimestamp:      collection.Version.SourceTimestamp,
					NormalizationVersion: collection.Version.NormalizationVersion,
				},
			}
		}
		writeJSON(response, http.StatusOK, districtFeatureCollection{
			Type: "FeatureCollection", Features: features,
			Meta: districtMeta{
				DistrictDataVersionID: collection.Version.ID, ReturnedCount: len(features),
				BBox: [4]float64{bbox.West, bbox.South, bbox.East, bbox.North},
			},
		})
	}
}

func currentGeoVersionHandler(deps Dependencies) http.HandlerFunc {
	return func(response http.ResponseWriter, request *http.Request) {
		cityID := chi.URLParam(request, "cityId")
		if !uuidPattern.MatchString(cityID) {
			writeAPIError(response, http.StatusBadRequest, "invalid_city_id", "cityId must be a UUID")
			return
		}
		version, err := deps.Geo.CurrentVersion(request.Context(), cityID)
		if err != nil {
			handleGeoError(response, deps, err)
			return
		}
		writeJSON(response, http.StatusOK, versionResponse(version))
	}
}

func segmentsHandler(deps Dependencies) http.HandlerFunc {
	return func(response http.ResponseWriter, request *http.Request) {
		filter, err := parseSegmentFilter(request)
		if err != nil {
			writeAPIError(response, http.StatusBadRequest, "invalid_query", err.Error())
			return
		}
		collection, err := deps.Geo.Segments(request.Context(), filter)
		if err != nil {
			handleGeoError(response, deps, err)
			return
		}
		features := make([]geoJSONFeature, len(collection.Segments))
		for index, segment := range collection.Segments {
			features[index] = geoJSONFeature{
				Type:     "Feature",
				ID:       segment.ID,
				Geometry: segment.GeometryJSON,
				Properties: geoJSONFeatureProperties{
					ID:               segment.ID,
					GeoDataVersionID: segment.GeoDataVersionID,
					Classification:   segment.Classification,
					LengthMeters:     segment.LengthMeters,
					StreetName:       segment.StreetName,
					ReasonCode:       segment.Attributes.ReasonCode,
					BoundaryClip:     segment.Attributes.BoundaryClip,
				},
			}
		}
		writeJSON(response, http.StatusOK, geoJSONFeatureCollection{
			Type:     "FeatureCollection",
			Features: features,
			Meta: segmentCollectionMeta{
				GeoDataVersionID: collection.Version.ID,
				ReturnedCount:    len(features),
				BBox:             [4]float64{filter.BBox.West, filter.BBox.South, filter.BBox.East, filter.BBox.North},
				Statistics:       collection.Statistics,
			},
		})
	}
}

func segmentDetailHandler(deps Dependencies) http.HandlerFunc {
	return func(response http.ResponseWriter, request *http.Request) {
		segmentID := chi.URLParam(request, "segmentId")
		if !uuidPattern.MatchString(segmentID) {
			writeAPIError(response, http.StatusBadRequest, "invalid_segment_id", "segmentId must be a UUID")
			return
		}
		segment, err := deps.Geo.Segment(request.Context(), segmentID)
		if err != nil {
			handleGeoError(response, deps, err)
			return
		}
		includeDebug := request.URL.Query().Get("debug") == "true" && deps.Environment != "production"
		writeJSON(response, http.StatusOK, detailResponse(segment, includeDebug))
	}
}

func parseSegmentFilter(request *http.Request) (querying.SegmentFilter, error) {
	query := request.URL.Query()
	cityID := strings.TrimSpace(query.Get("cityId"))
	if !uuidPattern.MatchString(cityID) {
		return querying.SegmentFilter{}, errors.New("cityId must be a UUID")
	}
	bbox, err := parseBBox(query.Get("bbox"))
	if err != nil {
		return querying.SegmentFilter{}, err
	}
	classifications, err := parseClassifications(query.Get("classification"))
	if err != nil {
		return querying.SegmentFilter{}, err
	}
	minimum, err := parseOptionalLength(query.Get("minLength"), "minLength")
	if err != nil {
		return querying.SegmentFilter{}, err
	}
	maximum, err := parseOptionalLength(query.Get("maxLength"), "maxLength")
	if err != nil {
		return querying.SegmentFilter{}, err
	}
	if minimum != nil && maximum != nil && *minimum > *maximum {
		return querying.SegmentFilter{}, errors.New("minLength cannot exceed maxLength")
	}
	return querying.SegmentFilter{
		CityID: cityID, BBox: bbox, Classifications: classifications,
		MinLength: minimum, MaxLength: maximum,
	}, nil
}

func parseBBox(value string) (querying.BBox, error) {
	parts := strings.Split(value, ",")
	if len(parts) != 4 {
		return querying.BBox{}, errors.New("bbox must contain west,south,east,north")
	}
	coordinates := [4]float64{}
	for index, part := range parts {
		coordinate, err := strconv.ParseFloat(strings.TrimSpace(part), 64)
		if err != nil || math.IsNaN(coordinate) || math.IsInf(coordinate, 0) {
			return querying.BBox{}, errors.New("bbox coordinates must be finite numbers")
		}
		coordinates[index] = coordinate
	}
	bbox := querying.BBox{West: coordinates[0], South: coordinates[1], East: coordinates[2], North: coordinates[3]}
	if !bbox.Valid() {
		return querying.BBox{}, errors.New("bbox coordinates are outside EPSG:4326 or have invalid order")
	}
	return bbox, nil
}

func parseClassifications(value string) ([]domain.StreetSegmentClassification, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	seen := make(map[domain.StreetSegmentClassification]bool)
	result := make([]domain.StreetSegmentClassification, 0)
	for _, part := range strings.Split(value, ",") {
		classification := domain.StreetSegmentClassification(strings.ToUpper(strings.TrimSpace(part)))
		switch classification {
		case domain.StreetSegmentExplore, domain.StreetSegmentRoutableOnly, domain.StreetSegmentIgnore:
		default:
			return nil, errors.New("classification contains an unsupported value")
		}
		if !seen[classification] {
			seen[classification] = true
			result = append(result, classification)
		}
	}
	return result, nil
}

func parseOptionalLength(value, name string) (*float64, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil || math.IsNaN(parsed) || math.IsInf(parsed, 0) || parsed < 0 {
		return nil, errors.New(name + " must be a finite non-negative number")
	}
	return &parsed, nil
}

func versionResponse(version querying.Version) geoVersionResponse {
	return geoVersionResponse{
		ID: version.ID, CityID: version.CityID, Source: version.Source,
		SourceTimestamp: version.SourceTimestamp, SourceChecksum: version.SourceChecksum,
		NormalizationVersion: version.NormalizationVersion, Status: version.Status,
		ImportedAt: version.ImportedAt, ImportReport: version.ImportReport,
	}
}

func detailResponse(segment querying.Segment, includeDebug bool) segmentDetailResponse {
	var street *segmentStreetResponse
	if segment.StreetID != nil {
		street = &segmentStreetResponse{ID: *segment.StreetID, Name: segment.StreetName}
	}
	response := segmentDetailResponse{
		ID: segment.ID, CityID: segment.CityID, GeoDataVersionID: segment.GeoDataVersionID,
		VersionStatus: segment.VersionStatus, NormalizationVersion: segment.NormalizationVersion,
		IsCurrent: segment.IsCurrent, Geometry: segment.GeometryJSON,
		LengthMeters: segment.LengthMeters, Classification: segment.Classification,
		ReasonCode:           segment.Attributes.ReasonCode,
		NormalizedAttributes: normalizedAttributes(segment.Attributes), Street: street,
		Districts: segment.Districts,
	}
	if includeDebug {
		response.DebugSource = &segmentDebugSource{
			Tags: segment.Attributes.SourceTags, WayIDs: segment.Attributes.SourceWayIDs,
			StartNodeID: segment.Attributes.SourceStartNodeID, EndNodeID: segment.Attributes.SourceEndNodeID,
		}
	}
	return response
}

func normalizedAttributes(attributes domain.StreetSegmentAttributes) map[string]any {
	result := make(map[string]any)
	for _, key := range []string{
		"highway", "footway", "service", "surface", "access", "foot", "sidewalk",
		"bridge", "tunnel", "indoor", "level", "oneway:foot", "foot:forward", "foot:backward",
	} {
		if value := attributes.SourceTags[key]; value != "" {
			result[key] = value
		}
	}
	if attributes.BoundaryClip {
		result["boundaryClip"] = true
	}
	if len(attributes.Warnings) > 0 {
		result["warnings"] = attributes.Warnings
	}
	return result
}

func handleGeoError(response http.ResponseWriter, deps Dependencies, err error) {
	switch {
	case errors.Is(err, context.Canceled):
		return
	case errors.Is(err, querying.ErrNotFound):
		writeAPIError(response, http.StatusNotFound, "not_found", "geo resource was not found")
	case errors.Is(err, querying.ErrBBoxAreaLimit):
		writeAPIError(response, http.StatusUnprocessableEntity, "bbox_area_exceeded", "bbox area exceeds 25 square kilometers")
	case errors.Is(err, querying.ErrFeatureLimit):
		writeAPIError(response, http.StatusUnprocessableEntity, "feature_limit_exceeded", "viewport contains more than 10000 segments; zoom in")
	default:
		deps.Logger.Error("geo API request failed", "error", err)
		writeAPIError(response, http.StatusInternalServerError, "internal_error", "internal server error")
	}
}

func writeAPIError(response http.ResponseWriter, status int, code, message string) {
	writeJSON(response, status, apiErrorBody{Error: apiError{Code: code, Message: message}})
}
