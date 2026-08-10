package routingspike

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

type Options struct {
	RepositoryRoot   string
	FixturePath      string
	OutputPath       string
	SetupMetricsPath string
	Analyzer         Analyzer
	Now              func() time.Time
	Clients          []engineClient
}

func Run(ctx context.Context, options Options) (Report, error) {
	if options.RepositoryRoot == "" {
		options.RepositoryRoot = "."
	}
	if options.Now == nil {
		options.Now = time.Now
	}
	if options.Clients == nil {
		options.Clients = clientsFromEnvironment()
	}
	fixture, routes, err := loadInputs(options.RepositoryRoot, options.FixturePath)
	if err != nil {
		return Report{}, err
	}
	report := Report{
		SchemaVersion: SchemaVersion, Status: "complete", GeneratedAt: options.Now().UTC(),
		Dataset: FixtureDataset{GeoFixture: fixture.Dataset.GeoFixture, PBFFile: fixture.Dataset.PBFFile,
			PBFSHA256: fixture.Dataset.PBFSHA256, RouteFixture: fixture.Dataset.RouteFixture},
		Benchmark: FixtureBenchmark{WarmRequests: fixture.Benchmark.WarmRequests,
			CorridorMeters: fixture.Benchmark.CorridorMeters, SampleStepMeters: fixture.Benchmark.SampleStepMeters},
		Notes: []string{
			"Docker Desktop resource values are relative local measurements, not production sizing.",
			"Routing-engine edge identifiers are intentionally discarded before StreetSegment matching.",
		},
	}
	for _, client := range options.Clients {
		report.Engines = append(report.Engines, client.Metadata())
	}
	if options.SetupMetricsPath != "" {
		setup, loadErr := loadSetupMetrics(options.SetupMetricsPath)
		if loadErr != nil && !errors.Is(loadErr, os.ErrNotExist) {
			return Report{}, loadErr
		}
		report.Setup = setup
	}
	for _, configured := range fixture.Cases {
		reference, found := routes[configured.RouteID]
		if !found {
			return Report{}, fmt.Errorf("routing case references unknown route %q", configured.RouteID)
		}
		waypoints, waypointErr := selectWaypoints(reference.Geometry.Coordinates, configured.WaypointIndexes)
		if waypointErr != nil {
			return Report{}, fmt.Errorf("routing case %q: %w", configured.RouteID, waypointErr)
		}
		caseResult := CaseResult{
			RouteID: configured.RouteID, Name: reference.Name, AreaID: reference.AreaID,
			Description: reference.Description, IntentionalUnmatched: reference.IntentionalUnmatched,
			Note: configured.Note, Waypoints: waypoints, ReferenceGeometry: reference.Geometry,
			ReferenceLengthMeters: geometryLength(reference.Geometry),
		}
		for _, client := range options.Clients {
			result := benchmarkRoute(ctx, client, waypoints, reference.Geometry, fixture, options.Analyzer, configured.RouteID)
			caseResult.Results = append(caseResult.Results, result)
			if result.Status != "ok" {
				report.Status = "complete_with_failures"
			}
		}
		report.Cases = append(report.Cases, caseResult)
	}
	matchSet := make(map[string]bool, len(fixture.MapMatchingRouteIDs))
	for _, routeID := range fixture.MapMatchingRouteIDs {
		matchSet[routeID] = true
	}
	for _, routeCase := range report.Cases {
		if !matchSet[routeCase.RouteID] {
			continue
		}
		for _, client := range options.Clients {
			report.MapMatching = append(report.MapMatching,
				benchmarkMapMatch(ctx, client, routeCase.RouteID, routeCase.ReferenceGeometry, fixture, options.Analyzer))
		}
	}
	report.Summary = buildSummaries(report.Engines, report.Cases, report.MapMatching)
	if options.OutputPath != "" {
		if err := writeReport(options.OutputPath, report); err != nil {
			return Report{}, err
		}
	}
	return report, nil
}

func benchmarkRoute(
	ctx context.Context, client engineClient, waypoints []Point, reference LineString, fixture Fixture,
	analyzer Analyzer, routeID string,
) RouteResult {
	result := RouteResult{EngineID: client.Metadata().ID, Status: "error"}
	started := time.Now()
	route, err := client.Route(ctx, waypoints)
	result.Latency.FirstMilliseconds = milliseconds(time.Since(started))
	if err != nil {
		result.Error = err.Error()
		return result
	}
	latencies := make([]float64, 0, fixture.Benchmark.WarmRequests)
	for range fixture.Benchmark.WarmRequests {
		started = time.Now()
		if _, err = client.Route(ctx, waypoints); err != nil {
			result.Error = "warm request failed: " + err.Error()
			return result
		}
		latencies = append(latencies, milliseconds(time.Since(started)))
	}
	result.Status = "ok"
	result.DistanceMeters = route.DistanceMeters
	result.DurationSeconds = route.DurationSeconds
	result.GeometryLengthMeters = geometryLength(route.Geometry)
	result.ResponseBytes = route.ResponseBytes
	result.Geometry = &route.Geometry
	result.Latency.WarmRequests = len(latencies)
	result.Latency.P50Milliseconds = percentile(latencies, .5)
	result.Latency.P95Milliseconds = percentile(latencies, .95)
	result.Corridor = corridorMetrics(route.Geometry, reference,
		fixture.Benchmark.CorridorMeters, fixture.Benchmark.SampleStepMeters)
	if analyzer != nil {
		geometry, _ := json.Marshal(route.Geometry)
		matcher, analysisErr := analyzer.Analyze(routeID+"-"+client.Metadata().ID, geometry)
		if analysisErr != nil {
			result.Error = "StreetSegment matcher: " + analysisErr.Error()
		} else {
			result.Matcher = matcher
		}
	}
	return result
}

func benchmarkMapMatch(
	ctx context.Context, client engineClient, routeID string, trace LineString, fixture Fixture, analyzer Analyzer,
) MapMatchResult {
	result := MapMatchResult{RouteID: routeID, EngineID: client.Metadata().ID, Status: "unavailable"}
	started := time.Now()
	matched, err := client.MapMatch(ctx, trace.Coordinates)
	result.LatencyMilliseconds = milliseconds(time.Since(started))
	if err != nil {
		result.Error = err.Error()
		return result
	}
	result.Status = "ok"
	result.Geometry = &matched.Geometry
	result.ResponseBytes = matched.ResponseBytes
	result.Corridor = corridorMetrics(matched.Geometry, trace,
		fixture.Benchmark.CorridorMeters, fixture.Benchmark.SampleStepMeters)
	if analyzer != nil {
		geometry, _ := json.Marshal(matched.Geometry)
		metrics, analysisErr := analyzer.Analyze(routeID+"-map-match-"+client.Metadata().ID, geometry)
		if analysisErr != nil {
			result.Error = "StreetSegment matcher: " + analysisErr.Error()
		} else {
			result.RouteMatchedRatio = metrics.RouteMatchedRatio
		}
	}
	return result
}

func loadInputs(repositoryRoot, fixturePath string) (Fixture, map[string]ReferenceRoute, error) {
	fixtureBytes, err := os.ReadFile(resolvePath(repositoryRoot, fixturePath))
	if err != nil {
		return Fixture{}, nil, fmt.Errorf("read routing fixture: %w", err)
	}
	var fixture Fixture
	if err := json.Unmarshal(fixtureBytes, &fixture); err != nil {
		return Fixture{}, nil, fmt.Errorf("decode routing fixture: %w", err)
	}
	if fixture.SchemaVersion != SchemaVersion || len(fixture.Cases) == 0 || fixture.Benchmark.WarmRequests < 1 ||
		fixture.Benchmark.CorridorMeters <= 0 || fixture.Benchmark.SampleStepMeters <= 0 {
		return Fixture{}, nil, errors.New("routing fixture is invalid")
	}
	pbfPath := resolvePath(repositoryRoot, fixture.Dataset.PBFFile)
	pbf, err := os.Open(pbfPath)
	if err != nil {
		return Fixture{}, nil, fmt.Errorf("open routing PBF: %w", err)
	}
	defer pbf.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, pbf); err != nil {
		return Fixture{}, nil, fmt.Errorf("checksum routing PBF: %w", err)
	}
	if hex.EncodeToString(hash.Sum(nil)) != fixture.Dataset.PBFSHA256 {
		return Fixture{}, nil, errors.New("routing PBF checksum mismatch")
	}
	routeBytes, err := os.ReadFile(resolvePath(repositoryRoot, fixture.Dataset.RouteFixture))
	if err != nil {
		return Fixture{}, nil, fmt.Errorf("read reference routes: %w", err)
	}
	var collection struct {
		Type     string `json:"type"`
		Features []struct {
			ID         string `json:"id"`
			Properties struct {
				ID                   string `json:"id"`
				Name                 string `json:"name"`
				AreaID               string `json:"areaId"`
				Description          string `json:"description"`
				IntentionalUnmatched bool   `json:"intentionalUnmatched"`
			} `json:"properties"`
			Geometry LineString `json:"geometry"`
		} `json:"features"`
	}
	if err := json.Unmarshal(routeBytes, &collection); err != nil || collection.Type != "FeatureCollection" {
		return Fixture{}, nil, errors.New("reference route fixture is invalid")
	}
	routes := make(map[string]ReferenceRoute, len(collection.Features))
	for _, feature := range collection.Features {
		if feature.ID == "" || feature.ID != feature.Properties.ID || feature.Geometry.Type != "LineString" ||
			len(feature.Geometry.Coordinates) < 2 {
			return Fixture{}, nil, errors.New("reference route feature is invalid")
		}
		routes[feature.ID] = ReferenceRoute{ID: feature.ID, Name: feature.Properties.Name,
			AreaID: feature.Properties.AreaID, Description: feature.Properties.Description,
			IntentionalUnmatched: feature.Properties.IntentionalUnmatched, Geometry: feature.Geometry}
	}
	return fixture, routes, nil
}

func loadSetupMetrics(path string) (*SetupMetrics, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var metrics SetupMetrics
	if err := json.Unmarshal(data, &metrics); err != nil {
		return nil, fmt.Errorf("decode setup metrics: %w", err)
	}
	return &metrics, nil
}

func selectWaypoints(points []Point, indexes []int) ([]Point, error) {
	if len(indexes) < 2 || len(indexes) > 4 {
		return nil, errors.New("each case must have two to four waypoints")
	}
	result := make([]Point, len(indexes))
	previous := -1
	for position, index := range indexes {
		if index <= previous || index < 0 || index >= len(points) {
			return nil, errors.New("waypoint indexes must be unique, ordered and inside the route")
		}
		result[position] = points[index]
		previous = index
	}
	if indexes[0] != 0 || indexes[len(indexes)-1] != len(points)-1 {
		return nil, errors.New("waypoint indexes must include route start and finish")
	}
	return result, nil
}

func writeReport(path string, report Report) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	temporary := path + ".tmp"
	if err := os.WriteFile(temporary, data, 0o644); err != nil {
		return err
	}
	if err := os.Rename(temporary, path); err != nil {
		return err
	}
	return nil
}

func resolvePath(root, path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(root, path)
}

func milliseconds(duration time.Duration) float64 {
	return float64(duration.Microseconds()) / 1000
}
