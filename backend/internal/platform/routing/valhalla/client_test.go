package valhalla

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/doveva/Gulyaem/backend/internal/routing/port"
)

func TestClientNormalizesSuccessfulRoute(t *testing.T) {
	client := newTestClient(validMetadata())
	client.httpClient.Transport = roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.Method != http.MethodPost || request.URL.Path != "/route" {
			t.Fatalf("request = %s %s", request.Method, request.URL.Path)
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{
			"trip":{"summary":{"length":1.25,"time":900},
			"locations":[{"lat":59.93,"lon":30.31},{"lat":59.94,"lon":30.32}],
			"legs":[{"shape":{"type":"LineString","coordinates":[[30.31,59.93],[30.315,59.935]]}},
			{"shape":{"type":"LineString","coordinates":[[30.315,59.935],[30.32,59.94]]}}]}}
		`)), Header: make(http.Header)}, nil
	})
	result, err := client.Route(context.Background(), port.RouteRequest{
		Profile: "pedestrian", Waypoints: []port.Point{{Lat: 59.93, Lon: 30.31}, {Lat: 59.94, Lon: 30.32}},
	})
	if err != nil {
		t.Fatalf("Route() error = %v", err)
	}
	if result.DistanceMeters != 1250 || result.DurationSeconds != 900 {
		t.Fatalf("summary = %#v", result)
	}
	if len(result.Waypoints) != 2 || result.Waypoints[0].Resolved == nil {
		t.Fatalf("waypoints = %#v", result.Waypoints)
	}
	if got := string(result.Geometry); got != `{"type":"LineString","coordinates":[[30.31,59.93],[30.315,59.935],[30.32,59.94]]}` {
		t.Fatalf("geometry = %s", got)
	}
}

func TestClientNormalizesFailures(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
		want   error
	}{
		{name: "no route", status: http.StatusBadRequest, body: `{}`, want: port.ErrRouteNotFound},
		{name: "upstream unavailable", status: http.StatusBadGateway, body: `{}`, want: port.ErrUnavailable},
		{name: "malformed", status: http.StatusOK, body: `{`, want: port.ErrInvalidResponse},
		{name: "missing geometry", status: http.StatusOK, body: `{"trip":{"summary":{"length":1,"time":2}}}`, want: port.ErrInvalidResponse},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := newTestClient(validMetadata())
			client.httpClient.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: test.status, Body: io.NopCloser(strings.NewReader(test.body)), Header: make(http.Header)}, nil
			})
			_, err := client.Route(context.Background(), port.RouteRequest{Profile: "pedestrian", Waypoints: []port.Point{{}, {}}})
			if !errors.Is(err, test.want) {
				t.Fatalf("error = %v, want %v", err, test.want)
			}
		})
	}
}

func TestDecodeShapeSupportsValhallaPolyline6(t *testing.T) {
	coordinates, err := decodeShape([]byte(`"wkciqBestxx@dAhF"`))
	if err != nil {
		t.Fatalf("decodeShape() error = %v", err)
	}
	if len(coordinates) != 2 || !validCoordinates(coordinates) {
		t.Fatalf("coordinates = %#v", coordinates)
	}
}

func TestClientNormalizesTimeout(t *testing.T) {
	client := newTestClient(validMetadata())
	client.httpClient.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, context.DeadlineExceeded
	})
	_, err := client.Route(context.Background(), port.RouteRequest{Profile: "pedestrian", Waypoints: []port.Point{{}, {}}})
	if !errors.Is(err, port.ErrTimeout) {
		t.Fatalf("error = %v, want timeout", err)
	}
}

func TestMetadataMustBeComplete(t *testing.T) {
	client := newTestClient(port.Metadata{Engine: "valhalla"})
	if _, err := client.Metadata(context.Background()); !errors.Is(err, port.ErrInvalidResponse) {
		t.Fatalf("Metadata() error = %v", err)
	}
}

func TestPingUsesValhallaStatus(t *testing.T) {
	client := newTestClient(validMetadata())
	client.httpClient.Transport = roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.Method != http.MethodGet || request.URL.Path != "/status" {
			t.Fatalf("request = %s %s", request.Method, request.URL.Path)
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{}`)), Header: make(http.Header)}, nil
	})
	if err := client.Ping(context.Background()); err != nil {
		t.Fatalf("Ping() error = %v", err)
	}
}

func validMetadata() port.Metadata {
	return port.Metadata{
		Engine: "valhalla", EngineVersion: "3.7.0", CityID: "01900000-0000-7000-8000-000000000001",
		SourceChecksum: strings.Repeat("a", 64), Profile: "pedestrian",
		GraphArtifact: "valhalla_tiles.tar", GraphChecksum: strings.Repeat("b", 64),
	}
}

func newTestClient(metadata port.Metadata) *Client {
	return New("http://valhalla.test", time.Second, staticMetadataSource{metadata: metadata})
}

type staticMetadataSource struct {
	metadata port.Metadata
	err      error
}

func (source staticMetadataSource) Load(context.Context) (port.Metadata, error) {
	return source.metadata, source.err
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (function roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return function(request)
}
