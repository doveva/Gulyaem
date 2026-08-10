package routeanalysis

import (
	"path/filepath"
	"testing"

	"github.com/doveva/Gulyaem/backend/internal/geo/domain"
)

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

func TestCoverageDoesNotCrossGradeSignature(t *testing.T) {
	candidates := []CandidateSegment{
		{ID: "surface", LengthMeters: 100, RadiusCoveredMeters: 100, Classification: domain.StreetSegmentExplore},
		{ID: "tunnel", LengthMeters: 100, RadiusCoveredMeters: 100, Classification: domain.StreetSegmentExplore,
			Attributes: domain.StreetSegmentAttributes{SourceTags: map[string]string{"tunnel": "yes", "level": "-1"}}},
	}
	coverage, _ := calculateCoverage(candidates, map[string][][2]float64{"surface": {{0, 20}}}, CoverageProfiles["balanced"])
	for _, segment := range coverage {
		if segment.SegmentID == "tunnel" && segment.CoveredMeters != 0 {
			t.Fatalf("tunnel covered meters = %f", segment.CoveredMeters)
		}
	}
}
