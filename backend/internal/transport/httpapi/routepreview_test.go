package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/doveva/Gulyaem/backend/internal/geo/domain"
	"github.com/doveva/Gulyaem/backend/internal/geo/querying"
	"github.com/doveva/Gulyaem/backend/internal/geo/routeanalysis"
	"github.com/doveva/Gulyaem/backend/internal/routing/port"
	"github.com/doveva/Gulyaem/backend/internal/routing/preview"
)

type previewEngineStub struct {
	metadata port.Metadata
	result   port.RouteResult
	err      error
}

func (stub previewEngineStub) Metadata(context.Context) (port.Metadata, error) {
	return stub.metadata, nil
}
func (stub previewEngineStub) Route(context.Context, port.RouteRequest) (port.RouteResult, error) {
	return stub.result, stub.err
}

type previewAnalyzerStub struct {
	version  querying.Version
	analysis routeanalysis.Analysis
	err      error
}

func (stub previewAnalyzerStub) CurrentVersion(context.Context, string) (querying.Version, error) {
	return stub.version, stub.err
}
func (stub previewAnalyzerStub) AnalyzeGeometryForVersion(context.Context, querying.Version, string, json.RawMessage, routeanalysis.AnalyzeRequest) (routeanalysis.Analysis, error) {
	return stub.analysis, stub.err
}

func TestRoutePreviewEndpointSuccess(t *testing.T) {
	handler := routePreviewTestHandler(nil, "checksum", "checksum")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/route-previews", strings.NewReader(validPreviewBody())))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `"engine":"valhalla"`) || !strings.Contains(response.Body.String(), `"geoDataVersion"`) {
		t.Fatalf("body = %s", response.Body.String())
	}
}

func TestRoutePreviewEndpointRejectsMalformedAndUnknownFields(t *testing.T) {
	handler := routePreviewTestHandler(nil, "checksum", "checksum")
	for _, body := range []string{`{`, strings.TrimSuffix(validPreviewBody(), "}") + `,"extra":true}`} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/route-previews", strings.NewReader(body)))
		if response.Code != http.StatusBadRequest {
			t.Fatalf("body %q status = %d", body, response.Code)
		}
	}
}

func TestRoutePreviewEndpointNormalizesErrors(t *testing.T) {
	tests := []struct {
		name            string
		err             error
		routingChecksum string
		geoChecksum     string
		status          int
		code            string
	}{
		{name: "no route", err: port.ErrRouteNotFound, routingChecksum: "same", geoChecksum: "same", status: 422, code: "route_not_found"},
		{name: "unavailable", err: port.ErrUnavailable, routingChecksum: "same", geoChecksum: "same", status: 503, code: "routing_unavailable"},
		{name: "timeout", err: port.ErrTimeout, routingChecksum: "same", geoChecksum: "same", status: 504, code: "routing_timeout"},
		{name: "mismatch", routingChecksum: "routing", geoChecksum: "geo", status: 409, code: "routing_geo_version_mismatch"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler := routePreviewTestHandler(test.err, test.routingChecksum, test.geoChecksum)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/route-previews", strings.NewReader(validPreviewBody())))
			if response.Code != test.status || !strings.Contains(response.Body.String(), `"code":"`+test.code+`"`) {
				t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
			}
		})
	}
}

func TestRoutePreviewEndpointRejectsInvalidWaypoints(t *testing.T) {
	handler := routePreviewTestHandler(nil, "checksum", "checksum")
	body := `{"cityId":"01900000-0000-7000-8000-000000000001","profile":"pedestrian","waypoints":[{"lat":59.93,"lon":30.31}]}`
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/route-previews", strings.NewReader(body)))
	if response.Code != http.StatusUnprocessableEntity || !strings.Contains(response.Body.String(), `"code":"invalid_waypoints"`) {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
}

func routePreviewTestHandler(routeError error, routingChecksum, geoChecksum string) http.Handler {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	version := querying.Version{ID: "02900000-0000-7000-8000-000000000001", CityID: "01900000-0000-7000-8000-000000000001", SourceChecksum: geoChecksum, Status: domain.GeoDataVersionReady}
	geometry := json.RawMessage(`{"type":"LineString","coordinates":[[30.31,59.93],[30.32,59.94]]}`)
	service := preview.NewService(previewEngineStub{
		metadata: port.Metadata{
			Engine: "valhalla", EngineVersion: "3.7.0", CityID: version.CityID,
			SourceChecksum: routingChecksum, Profile: "pedestrian",
			GraphArtifact: "valhalla_tiles.tar", GraphChecksum: "graph",
		},
		result: port.RouteResult{DistanceMeters: 1000, DurationSeconds: 800, Geometry: geometry}, err: routeError,
	}, previewAnalyzerStub{version: version, analysis: routeanalysis.Analysis{
		GeoDataVersion:  routeanalysis.VersionReference{ID: version.ID, CityID: version.CityID, SourceChecksum: version.SourceChecksum, Status: version.Status},
		CoverageProfile: routeanalysis.CoverageProfiles["balanced"], Metrics: routeanalysis.Metrics{RouteMatchedRatio: 1},
	}}, logger)
	return NewHandler(Dependencies{Database: healthCheckerStub{}, Logger: logger, Environment: "test", RoutePreview: service})
}

func validPreviewBody() string {
	return `{"cityId":"01900000-0000-7000-8000-000000000001","profile":"pedestrian","waypoints":[{"lat":59.93,"lon":30.31},{"lat":59.94,"lon":30.32}]}`
}

var _ = errors.New
