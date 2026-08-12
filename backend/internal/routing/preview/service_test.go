package preview

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/doveva/Gulyaem/backend/internal/geo/domain"
	"github.com/doveva/Gulyaem/backend/internal/geo/querying"
	"github.com/doveva/Gulyaem/backend/internal/geo/routeanalysis"
	"github.com/doveva/Gulyaem/backend/internal/routing/port"
)

type engineStub struct {
	metadata  port.Metadata
	result    port.RouteResult
	err       error
	calls     int
	routeHook func()
}

func (stub *engineStub) Metadata(context.Context) (port.Metadata, error) { return stub.metadata, nil }
func (stub *engineStub) Route(context.Context, port.RouteRequest) (port.RouteResult, error) {
	stub.calls++
	if stub.routeHook != nil {
		stub.routeHook()
	}
	return stub.result, stub.err
}

type analyzerStub struct {
	version  querying.Version
	analysis routeanalysis.Analysis
	err      error
}

func (stub analyzerStub) CurrentVersion(context.Context, string) (querying.Version, error) {
	return stub.version, stub.err
}
func (stub analyzerStub) AnalyzeGeometryForVersion(context.Context, querying.Version, string, json.RawMessage, routeanalysis.AnalyzeRequest) (routeanalysis.Analysis, error) {
	return stub.analysis, stub.err
}

type switchingAnalyzerStub struct {
	current         querying.Version
	analyzedVersion querying.Version
}

func (stub *switchingAnalyzerStub) CurrentVersion(context.Context, string) (querying.Version, error) {
	return stub.current, nil
}

func (stub *switchingAnalyzerStub) AnalyzeGeometryForVersion(
	_ context.Context, version querying.Version, _ string, _ json.RawMessage, _ routeanalysis.AnalyzeRequest,
) (routeanalysis.Analysis, error) {
	stub.analyzedVersion = version
	return routeanalysis.Analysis{
		GeoDataVersion: versionReferenceForTest(version),
		Metrics:        routeanalysis.Metrics{RouteMatchedRatio: 1},
	}, nil
}

func TestCreateComposesStatelessPreview(t *testing.T) {
	version := testVersion("checksum")
	engine := &engineStub{metadata: testMetadata("checksum"), result: port.RouteResult{
		DistanceMeters: 1200, DurationSeconds: 900,
		Geometry: json.RawMessage(`{"type":"LineString","coordinates":[[30.31,59.93],[30.32,59.94]]}`),
	}}
	analyzer := analyzerStub{version: version, analysis: routeanalysis.Analysis{
		GeoDataVersion: versionReferenceForTest(version), CoverageProfile: routeanalysis.CoverageProfiles["balanced"],
		Metrics:          routeanalysis.Metrics{RouteMatchedRatio: .95},
		MatchedFragments: []routeanalysis.MatchedFragment{{Classification: domain.StreetSegmentExplore, RouteStartMeters: 0, RouteEndMeters: 800}},
		CoverageSegments: []routeanalysis.CoverageSegment{{Status: "COMPLETED"}, {Status: "PARTIAL"}, {Status: "NOT_COVERED"}},
	}}
	service := NewService(engine, analyzer, slog.New(slog.NewTextHandler(io.Discard, nil)))
	result, err := service.Create(context.Background(), validRequest())
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if result.Routing.Engine != "valhalla" || result.ExplorationPreview.Metrics.CompletedSegmentCount != 1 ||
		result.ExplorationPreview.Metrics.PartialSegmentCount != 1 {
		t.Fatalf("result = %#v", result)
	}
	if len(result.Warnings) != 0 {
		t.Fatalf("warnings = %v", result.Warnings)
	}
	if len(result.ExplorationPreview.CoverageSegments) != 2 {
		t.Fatalf("coverage segments = %#v", result.ExplorationPreview.CoverageSegments)
	}
}

func TestCreateBlocksDatasetMismatchBeforeRouting(t *testing.T) {
	version := testVersion("geo")
	engine := &engineStub{metadata: testMetadata("routing")}
	service := NewService(engine, analyzerStub{version: version}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	_, err := service.Create(context.Background(), validRequest())
	if !errors.Is(err, ErrDatasetMismatch) {
		t.Fatalf("error = %v", err)
	}
	if engine.calls != 0 {
		t.Fatalf("routing calls = %d, want 0", engine.calls)
	}
}

func TestCreateKeepsResolvedVersionPinnedWhenCurrentVersionSwitchesDuringRouting(t *testing.T) {
	versionA := testVersion("checksum")
	versionB := versionA
	versionB.ID = "03900000-0000-7000-8000-000000000002"
	versionB.SourceChecksum = "replacement"
	analyzer := &switchingAnalyzerStub{current: versionA}
	engine := &engineStub{
		metadata: testMetadata(versionA.SourceChecksum),
		result: port.RouteResult{
			Geometry: json.RawMessage(`{"type":"LineString","coordinates":[[30.31,59.93],[30.32,59.94]]}`),
		},
		routeHook: func() { analyzer.current = versionB },
	}
	service := NewService(engine, analyzer, slog.New(slog.NewTextHandler(io.Discard, nil)))
	result, err := service.Create(context.Background(), validRequest())
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if analyzer.current.ID != versionB.ID || analyzer.analyzedVersion.ID != versionA.ID ||
		result.GeoDataVersion.ID != versionA.ID {
		t.Fatalf("current=%s analyzed=%s result=%s; want B/A/A",
			analyzer.current.ID, analyzer.analyzedVersion.ID, result.GeoDataVersion.ID)
	}
}

func TestReadyRequiresCompatibleGraphMetadataAndGeoVersion(t *testing.T) {
	version := testVersion("checksum")
	service := NewService(&engineStub{metadata: testMetadata("checksum")}, analyzerStub{version: version}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err := service.Ready(context.Background()); err != nil {
		t.Fatalf("Ready() error = %v", err)
	}

	mismatched := NewService(&engineStub{metadata: testMetadata("routing")}, analyzerStub{version: testVersion("geo")}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err := mismatched.Ready(context.Background()); !errors.Is(err, ErrDatasetMismatch) {
		t.Fatalf("Ready() error = %v, want dataset mismatch", err)
	}
}

func TestCreateBlocksRoutingCityMismatch(t *testing.T) {
	metadata := testMetadata("checksum")
	metadata.CityID = "02900000-0000-7000-8000-000000000002"
	engine := &engineStub{metadata: metadata}
	service := NewService(engine, analyzerStub{version: testVersion("checksum")}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	_, err := service.Create(context.Background(), validRequest())
	if !errors.Is(err, ErrDatasetMismatch) || engine.calls != 0 {
		t.Fatalf("error = %v, routing calls = %d", err, engine.calls)
	}
}

func TestCreateValidatesWaypoints(t *testing.T) {
	service := NewService(&engineStub{}, analyzerStub{}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	requests := []Request{
		{CityID: "city", Profile: "pedestrian", Waypoints: []port.Point{{}}},
		{CityID: "city", Profile: "bicycle", Waypoints: []port.Point{{}, {}}},
		{CityID: "city", Profile: "pedestrian", Waypoints: []port.Point{{Lat: 91}, {}}},
	}
	for _, request := range requests {
		if _, err := service.Create(context.Background(), request); !errors.Is(err, ErrInvalidRequest) {
			t.Fatalf("Create(%#v) error = %v", request, err)
		}
	}
}

func TestCreateAppliesEvidenceBasedLowMatchWarningThreshold(t *testing.T) {
	tests := []struct {
		name         string
		matchedRatio float64
		wantWarning  bool
	}{
		{name: "ambiguous Stage 1 evidence", matchedRatio: 0.9144705690360309, wantWarning: true},
		{name: "just below boundary", matchedRatio: lowRouteMatchWarningThreshold - 0.0001, wantWarning: true},
		{name: "exact boundary", matchedRatio: lowRouteMatchWarningThreshold, wantWarning: false},
		{name: "normal route", matchedRatio: 0.99, wantWarning: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			version := testVersion("checksum")
			service := NewService(&engineStub{metadata: testMetadata("checksum"), result: port.RouteResult{
				Geometry: json.RawMessage(`{"type":"LineString","coordinates":[[30.31,59.93],[30.32,59.94]]}`),
			}}, analyzerStub{version: version, analysis: routeanalysis.Analysis{
				GeoDataVersion: versionReferenceForTest(version),
				Metrics:        routeanalysis.Metrics{RouteMatchedRatio: test.matchedRatio},
			}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
			result, err := service.Create(context.Background(), validRequest())
			if err != nil {
				t.Fatalf("Create() error = %v", err)
			}
			gotWarning := len(result.Warnings) == 1 && result.Warnings[0] == "low_route_match"
			if gotWarning != test.wantWarning {
				t.Fatalf("warnings = %v for ratio %.9f, want warning %t",
					result.Warnings, test.matchedRatio, test.wantWarning)
			}
		})
	}
}

func validRequest() Request {
	return Request{CityID: "01900000-0000-7000-8000-000000000001", Profile: "pedestrian", Waypoints: []port.Point{{Lat: 59.93, Lon: 30.31}, {Lat: 59.94, Lon: 30.32}}}
}

func testVersion(checksum string) querying.Version {
	return querying.Version{ID: "02900000-0000-7000-8000-000000000001", CityID: "01900000-0000-7000-8000-000000000001", SourceChecksum: checksum, Status: domain.GeoDataVersionReady}
}

func versionReferenceForTest(version querying.Version) routeanalysis.VersionReference {
	return routeanalysis.VersionReference{ID: version.ID, CityID: version.CityID, SourceChecksum: version.SourceChecksum, Status: version.Status}
}

func testMetadata(checksum string) port.Metadata {
	return port.Metadata{
		Engine: "valhalla", EngineVersion: "3.7.0", CityID: "01900000-0000-7000-8000-000000000001",
		SourceChecksum: checksum, Profile: "pedestrian", GraphArtifact: "valhalla_tiles.tar", GraphChecksum: "graph",
	}
}
