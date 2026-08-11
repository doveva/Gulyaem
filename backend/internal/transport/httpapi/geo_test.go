package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/doveva/Gulyaem/backend/internal/geo/domain"
	"github.com/doveva/Gulyaem/backend/internal/geo/querying"
)

const (
	testCityID    = "01900000-0000-7000-8000-000000000001"
	testVersionID = "01900000-0000-7000-8000-000000000002"
	testSegmentID = "01900000-0000-7000-8000-000000000003"
)

type geoRepositoryStub struct {
	version          querying.Version
	districtVersion  querying.DistrictVersion
	segments         []querying.Segment
	segment          querying.Segment
	districts        []querying.District
	segmentDistricts []querying.DistrictSummary
	filter           querying.SegmentFilter
}

func (stub *geoRepositoryStub) CurrentVersion(context.Context, string) (querying.Version, error) {
	if stub.version.ID == "" {
		return querying.Version{}, querying.ErrNotFound
	}
	return stub.version, nil
}

func (stub *geoRepositoryStub) Segments(_ context.Context, filter querying.SegmentFilter, _ int) ([]querying.Segment, error) {
	stub.filter = filter
	return stub.segments, nil
}

func (stub *geoRepositoryStub) Segment(context.Context, string) (querying.Segment, error) {
	if stub.segment.ID == "" {
		return querying.Segment{}, querying.ErrNotFound
	}
	return stub.segment, nil
}

func (stub *geoRepositoryStub) CurrentDistrictVersion(context.Context, string) (querying.DistrictVersion, error) {
	if stub.districtVersion.ID == "" {
		return querying.DistrictVersion{}, querying.ErrNotFound
	}
	return stub.districtVersion, nil
}

func (stub *geoRepositoryStub) Districts(context.Context, querying.DistrictFilter) ([]querying.District, error) {
	return stub.districts, nil
}

func (stub *geoRepositoryStub) SegmentDistricts(context.Context, string) ([]querying.DistrictSummary, error) {
	return stub.segmentDistricts, nil
}

func TestGeoSegmentsReturnsGeoJSONAndPassesFilters(t *testing.T) {
	streetName := "Невский проспект"
	repository := &geoRepositoryStub{
		version: querying.Version{ID: testVersionID, CityID: testCityID, Status: domain.GeoDataVersionReady},
		segments: []querying.Segment{{
			ID: testSegmentID, GeoDataVersionID: testVersionID,
			GeometryJSON: json.RawMessage(`{"type":"LineString","coordinates":[[30.31,59.93],[30.32,59.94]]}`),
			LengthMeters: 42.5, Classification: domain.StreetSegmentExplore,
			Attributes: domain.StreetSegmentAttributes{ReasonCode: "pedestrian_highway"}, StreetName: &streetName,
		}},
	}
	handler := testGeoHandler(repository, "test")
	request := httptest.NewRequest(http.MethodGet,
		"/api/v1/geo/segments?cityId="+testCityID+"&bbox=30.30,59.92,30.33,59.95&classification=EXPLORE&minLength=5&maxLength=100", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", response.Code, response.Body.String())
	}
	var body geoJSONFeatureCollection
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Type != "FeatureCollection" || len(body.Features) != 1 || body.Features[0].ID != testSegmentID {
		t.Fatalf("body = %+v", body)
	}
	if body.Meta.ReturnedCount != 1 || body.Meta.Statistics.ExploreCount != 1 {
		t.Fatalf("meta = %+v", body.Meta)
	}
	if len(repository.filter.Classifications) != 1 || repository.filter.Classifications[0] != domain.StreetSegmentExplore {
		t.Fatalf("filter = %+v", repository.filter)
	}
	if repository.filter.MinLength == nil || *repository.filter.MinLength != 5 || repository.filter.MaxLength == nil || *repository.filter.MaxLength != 100 {
		t.Fatalf("length filter = %+v", repository.filter)
	}
}

func TestGeoDistrictsReturnsGeoJSONWithVersionMetadata(t *testing.T) {
	repository := &geoRepositoryStub{
		districtVersion: querying.DistrictVersion{
			ID: testVersionID, CityID: testCityID, Source: "openstreetmap",
			NormalizationVersion: "stage1-districts-v1", Status: domain.GeoDataVersionReady,
		},
		districts: []querying.District{{
			ID: testSegmentID, CityID: testCityID, DistrictDataVersionID: testVersionID,
			ExternalID: "relation/1114902", Name: "Центральный район", Kind: "administrative_district",
			GeometryJSON:   json.RawMessage(`{"type":"Polygon","coordinates":[]}`),
			LabelPointJSON: json.RawMessage(`{"type":"Point","coordinates":[30.3,59.9]}`),
		}},
	}
	response := httptest.NewRecorder()
	testGeoHandler(repository, "test").ServeHTTP(response, httptest.NewRequest(http.MethodGet,
		"/api/v1/geo/districts?cityId="+testCityID+"&bbox=30.30,59.92,30.31,59.93", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", response.Code, response.Body.String())
	}
	var body districtFeatureCollection
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if len(body.Features) != 1 || body.Features[0].Properties.Name != "Центральный район" || body.Meta.DistrictDataVersionID != testVersionID {
		t.Fatalf("body = %+v", body)
	}
}

func TestGeoSegmentsRejectsInvalidAndOversizedQueries(t *testing.T) {
	repository := &geoRepositoryStub{version: querying.Version{ID: testVersionID}}
	handler := testGeoHandler(repository, "test")
	tests := []struct {
		name       string
		query      string
		wantStatus int
		wantCode   string
	}{
		{name: "invalid bbox", query: "bbox=broken", wantStatus: http.StatusBadRequest, wantCode: "invalid_query"},
		{name: "invalid classification", query: "bbox=30.30,59.92,30.31,59.93&classification=OTHER", wantStatus: http.StatusBadRequest, wantCode: "invalid_query"},
		{name: "oversized bbox", query: "bbox=30,59,31,60", wantStatus: http.StatusUnprocessableEntity, wantCode: "bbox_area_exceeded"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/api/v1/geo/segments?cityId="+testCityID+"&"+test.query, nil)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != test.wantStatus {
				t.Fatalf("status = %d body = %s", response.Code, response.Body.String())
			}
			var body apiErrorBody
			if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if body.Error.Code != test.wantCode {
				t.Fatalf("code = %q", body.Error.Code)
			}
		})
	}
}

func TestGeoSegmentsReportsFeatureLimitWithoutTruncation(t *testing.T) {
	repository := &geoRepositoryStub{
		version:  querying.Version{ID: testVersionID},
		segments: make([]querying.Segment, querying.MaximumFeatures+1),
	}
	response := httptest.NewRecorder()
	testGeoHandler(repository, "test").ServeHTTP(response, httptest.NewRequest(http.MethodGet,
		"/api/v1/geo/segments?cityId="+testCityID+"&bbox=30.30,59.92,30.31,59.93", nil))
	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d body = %s", response.Code, response.Body.String())
	}
	var body apiErrorBody
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Error.Code != "feature_limit_exceeded" {
		t.Fatalf("code = %q", body.Error.Code)
	}
}

func TestSegmentDebugSourceIsHiddenInProduction(t *testing.T) {
	wayID := int64(123)
	segment := querying.Segment{
		ID: testSegmentID, CityID: testCityID, GeoDataVersionID: testVersionID,
		GeometryJSON: json.RawMessage(`{"type":"LineString","coordinates":[[30.31,59.93],[30.32,59.94]]}`),
		LengthMeters: 42.5, Classification: domain.StreetSegmentExplore,
		Attributes: domain.StreetSegmentAttributes{
			ReasonCode: "pedestrian_highway", SourceTags: map[string]string{"highway": "footway"}, SourceWayIDs: []int64{wayID},
		},
		VersionStatus: domain.GeoDataVersionSuperseded, NormalizationVersion: "stage1-segments-v1",
	}
	for _, test := range []struct {
		environment string
		wantDebug   bool
	}{{environment: "test", wantDebug: true}, {environment: "production", wantDebug: false}} {
		t.Run(test.environment, func(t *testing.T) {
			response := httptest.NewRecorder()
			testGeoHandler(&geoRepositoryStub{segment: segment}, test.environment).ServeHTTP(response,
				httptest.NewRequest(http.MethodGet, "/api/v1/geo/segments/"+testSegmentID+"?debug=true", nil))
			if response.Code != http.StatusOK {
				t.Fatalf("status = %d body = %s", response.Code, response.Body.String())
			}
			var body segmentDetailResponse
			if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if (body.DebugSource != nil) != test.wantDebug {
				t.Fatalf("debug source = %+v", body.DebugSource)
			}
		})
	}
}

func TestSegmentDetailKeepsOSMVocabularyOutOfNormalization(t *testing.T) {
	segment := querying.Segment{
		ID: testSegmentID, CityID: testCityID, GeoDataVersionID: testVersionID,
		GeometryJSON:   json.RawMessage(`{"type":"LineString","coordinates":[[30.31,59.93],[30.32,59.94]]}`),
		LengthMeters:   42.5,
		Classification: domain.StreetSegmentExplore,
		Attributes: domain.StreetSegmentAttributes{
			ReasonCode: "pedestrian_highway", SourceTags: map[string]string{"highway": "footway", "surface": "paving_stones"},
			BoundaryClip: true, Warnings: []string{"boundary_clip"},
		},
		VersionStatus: domain.GeoDataVersionReady, NormalizationVersion: "stage1-segments-v1",
	}
	response := httptest.NewRecorder()
	testGeoHandler(&geoRepositoryStub{segment: segment}, "test").ServeHTTP(response,
		httptest.NewRequest(http.MethodGet, "/api/v1/geo/segments/"+testSegmentID, nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", response.Code, response.Body.String())
	}
	if bytes.Contains(response.Body.Bytes(), []byte(`"normalizedAttributes"`)) ||
		bytes.Contains(response.Body.Bytes(), []byte(`"highway":`)) ||
		bytes.Contains(response.Body.Bytes(), []byte(`"surface":`)) {
		t.Fatalf("ordinary detail leaks OSM vocabulary: %s", response.Body.String())
	}
	var body segmentDetailResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if !body.Normalization.BoundaryClipped || len(body.Normalization.Warnings) != 1 ||
		body.Normalization.Warnings[0] != "boundary_clip" {
		t.Fatalf("normalization = %+v", body.Normalization)
	}
}

func testGeoHandler(repository querying.Repository, environment string) http.Handler {
	return NewHandler(Dependencies{
		Database: healthCheckerStub{}, Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		Environment: environment, Geo: querying.NewService(repository),
	})
}
