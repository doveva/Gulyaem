package routeanalysis

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"testing"

	"github.com/doveva/Gulyaem/backend/internal/geo/domain"
	"github.com/doveva/Gulyaem/backend/internal/geo/querying"
)

type pinningRepositoryStub struct {
	currentVersionCalls int
	candidateVersionID  string
	coverageVersionID   string
}

func (stub *pinningRepositoryStub) CurrentVersion(context.Context, string) (querying.Version, error) {
	stub.currentVersionCalls++
	return querying.Version{ID: "version-b", CityID: "city", Status: domain.GeoDataVersionReady}, nil
}

func (stub *pinningRepositoryStub) CandidateSegments(
	_ context.Context, _, versionID string, _ json.RawMessage, _ float64,
) ([]CandidateSegment, error) {
	stub.candidateVersionID = versionID
	return []CandidateSegment{{
		ID: "segment-a", Geometry: []domain.Point{{Lon: 30, Lat: 60}, {Lon: 30.001, Lat: 60}},
		LengthMeters: 56, Classification: domain.StreetSegmentExplore,
	}}, nil
}

func (stub *pinningRepositoryStub) CoverageSegments(
	_ context.Context, _, versionID string, _ []NormalizedRouteFragment, _, _ float64,
) ([]CandidateSegment, error) {
	stub.coverageVersionID = versionID
	return []CandidateSegment{{
		ID: "segment-a", Geometry: []domain.Point{{Lon: 30, Lat: 60}, {Lon: 30.001, Lat: 60}},
		LengthMeters: 56, Classification: domain.StreetSegmentExplore, RadiusCoveredMeters: 56,
	}}, nil
}

func TestAnalyzeGeometryForVersionPinsEveryRepositoryQuery(t *testing.T) {
	repository := &pinningRepositoryStub{}
	analyzer := NewAnalyzer(repository)
	pinned := querying.Version{
		ID: "version-a", CityID: "city", SourceChecksum: "checksum-a", Status: domain.GeoDataVersionReady,
	}
	analysis, err := analyzer.AnalyzeGeometryForVersion(
		context.Background(), pinned, "version-switch",
		json.RawMessage(`{"type":"LineString","coordinates":[[30,60],[30.001,60]]}`),
		AnalyzeRequest{Matching: DefaultMatchingParameters(), Coverage: CoverageProfiles["balanced"]},
	)
	if err != nil {
		t.Fatal(err)
	}
	if repository.currentVersionCalls != 0 {
		t.Fatalf("CurrentVersion calls during pinned analysis = %d, want 0", repository.currentVersionCalls)
	}
	if repository.candidateVersionID != pinned.ID || repository.coverageVersionID != pinned.ID {
		t.Fatalf("query versions = candidate %q, coverage %q; want %q",
			repository.candidateVersionID, repository.coverageVersionID, pinned.ID)
	}
	if analysis.GeoDataVersion.ID != pinned.ID {
		t.Fatalf("analysis version = %q, want %q", analysis.GeoDataVersion.ID, pinned.ID)
	}
}

func TestCoverageRadiusSupportedRange(t *testing.T) {
	request := AnalyzeRequest{Matching: DefaultMatchingParameters(), Coverage: CoverageProfile{
		Name: "custom", RadiusMeters: MaxCoverageRadiusMeters,
		CoverageRatio: .6, MinRequiredMeters: 15, MaxRequiredMeters: 80,
	}}
	if err := validateAnalyzeRequest(request); err != nil {
		t.Fatalf("maximum supported radius rejected: %v", err)
	}
	request.Coverage.RadiusMeters = MaxCoverageRadiusMeters + 1
	if err := validateAnalyzeRequest(request); !errors.Is(err, ErrInvalidParameters) {
		t.Fatalf("radius above supported range error = %v", err)
	}
}

func TestCoverageProfilePresets(t *testing.T) {
	tests := map[string]CoverageProfile{
		"strict":   {Name: "strict", RadiusMeters: 50, CoverageRatio: .8, MinRequiredMeters: 20, MaxRequiredMeters: 120},
		"balanced": {Name: "balanced", RadiusMeters: 100, CoverageRatio: .4, MinRequiredMeters: 15, MaxRequiredMeters: 80},
		"generous": {Name: "generous", RadiusMeters: 200, CoverageRatio: .4, MinRequiredMeters: 10, MaxRequiredMeters: 50},
	}
	for name, want := range tests {
		if got := CoverageProfiles[name]; got != want {
			t.Errorf("CoverageProfiles[%q] = %+v, want %+v", name, got, want)
		}
	}
}

func TestCommittedFixtureContainsFiveValidationRoutes(t *testing.T) {
	set, err := loadFixtureSet(filepath.Join("..", "..", "..", "..", "data"))
	if err != nil {
		t.Fatalf("loadFixtureSet() error = %v", err)
	}
	if len(set.routes) != 5 {
		t.Fatalf("routes = %d, want 5", len(set.routes))
	}
	if !set.byID["konyushennaya-capella-moyka"].IntentionalUnmatched {
		t.Fatal("courtyard fixture must document its intentional unmatched fragment")
	}
}

func TestSequentialMatcherHandlesUndirectedStraightSegment(t *testing.T) {
	route := []domain.Point{{Lon: 30, Lat: 60}, {Lon: 30.001, Lat: 60}}
	candidates := []CandidateSegment{{
		ID: "segment", Geometry: []domain.Point{{Lon: 30.001, Lat: 60.00001}, {Lon: 30, Lat: 60.00001}},
		LengthMeters: 56, Classification: domain.StreetSegmentExplore,
	}}
	matched, unmatched, normalized, _, matchedMeters := matchRoute(route, candidates, DefaultMatchingParameters())
	if len(matched) == 0 || len(unmatched) != 0 || len(normalized) != 1 {
		t.Fatalf("matched=%d unmatched=%d normalized=%d", len(matched), len(unmatched), len(normalized))
	}
	if matchedMeters < 50 {
		t.Fatalf("matched meters = %f", matchedMeters)
	}
}

func TestCoverageUsesThresholdAndKeepsConnectorSeparate(t *testing.T) {
	profile := CoverageProfiles["balanced"]
	candidates := []CandidateSegment{
		{ID: "complete", LengthMeters: 100, RadiusCoveredMeters: 70, Classification: domain.StreetSegmentExplore},
		{ID: "partial", LengthMeters: 100, RadiusCoveredMeters: 30, Classification: domain.StreetSegmentExplore},
		{ID: "none", LengthMeters: 100, Classification: domain.StreetSegmentExplore},
		{ID: "connector", LengthMeters: 20, Classification: domain.StreetSegmentRoutableOnly},
	}
	coverage, metrics := calculateCoverage(candidates, map[string][][2]float64{
		"connector": {{0, 10}},
	}, profile)
	statuses := make(map[string]string)
	for _, segment := range coverage {
		statuses[segment.SegmentID] = segment.Status
	}
	if statuses["complete"] != "COMPLETED" || statuses["partial"] != "PARTIAL" ||
		statuses["none"] != "NOT_COVERED" || statuses["connector"] != "CONNECTOR" {
		t.Fatalf("statuses = %v", statuses)
	}
	if metrics.ContextExplorableLengthMeters != 300 || metrics.CompletedNetworkLengthMeters != 100 {
		t.Fatalf("metrics = %+v", metrics)
	}
}

func TestMixedGradeRouteKeepsCoverageLocal(t *testing.T) {
	surface := &CandidateSegment{ID: "surface-a", LengthMeters: 100, Classification: domain.StreetSegmentExplore}
	tunnelAttributes := domain.StreetSegmentAttributes{SourceTags: map[string]string{"tunnel": "yes", "level": "-1"}}
	tunnel := &CandidateSegment{ID: "tunnel-b", LengthMeters: 100, Classification: domain.StreetSegmentExplore, Attributes: tunnelAttributes}
	samples := []routeSample{
		{point: domain.Point{Lon: 30, Lat: 60}, measure: 0},
		{point: domain.Point{Lon: 30.001, Lat: 60}, measure: 50},
		{point: domain.Point{Lon: 30.01, Lat: 60}, measure: 500},
		{point: domain.Point{Lon: 30.011, Lat: 60}, measure: 550},
	}
	matches := []sampleMatch{
		{segment: surface, nearest: nearestPoint{point: samples[0].point, measure: 0}},
		{segment: surface, nearest: nearestPoint{point: samples[1].point, measure: 50}},
		{segment: tunnel, nearest: nearestPoint{point: samples[2].point, measure: 0}},
		{segment: tunnel, nearest: nearestPoint{point: samples[3].point, measure: 50}},
	}
	_, _, fragments, direct, _ := assembleMatchResult(samples, matches, 5)
	if len(fragments) != 2 || fragments[0].GradeSignature != "surface" ||
		fragments[1].GradeSignature != "surface;tunnel=yes;level=-1" {
		t.Fatalf("normalized fragments = %+v", fragments)
	}

	candidates := []CandidateSegment{
		{ID: "surface-a", LengthMeters: 100, RadiusCoveredMeters: 100, Classification: domain.StreetSegmentExplore},
		{ID: "parallel-tunnel-a", LengthMeters: 100, Classification: domain.StreetSegmentExplore, Attributes: tunnelAttributes},
		{ID: "tunnel-b", LengthMeters: 100, RadiusCoveredMeters: 100, Classification: domain.StreetSegmentExplore, Attributes: tunnelAttributes},
	}
	coverage, _ := calculateCoverage(candidates, direct, CoverageProfiles["balanced"])
	statuses := make(map[string]string)
	for _, segment := range coverage {
		statuses[segment.SegmentID] = segment.Status
	}
	if statuses["surface-a"] != "COMPLETED" || statuses["parallel-tunnel-a"] != "NOT_COVERED" ||
		statuses["tunnel-b"] != "COMPLETED" {
		t.Fatalf("mixed-grade statuses = %v", statuses)
	}
}
