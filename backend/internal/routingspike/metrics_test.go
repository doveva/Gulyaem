package routingspike

import (
	"math"
	"testing"
)

func TestCorridorMetricsDistinguishNearbyAndDivergingLines(t *testing.T) {
	reference := LineString{Type: "LineString", Coordinates: []Point{{30, 60}, {30.01, 60}}}
	nearby := LineString{Type: "LineString", Coordinates: []Point{{30, 60.00005}, {30.01, 60.00005}}}
	far := LineString{Type: "LineString", Coordinates: []Point{{30, 60.001}, {30.01, 60.001}}}

	nearMetrics := corridorMetrics(nearby, reference, 20, 5)
	farMetrics := corridorMetrics(far, reference, 20, 5)
	if nearMetrics.CandidateInsideReferenceRatio < .99 || nearMetrics.ReferenceInsideCandidateRatio < .99 {
		t.Fatalf("nearby line should fit corridor: %+v", nearMetrics)
	}
	if farMetrics.CandidateInsideReferenceRatio > .01 || farMetrics.ReferenceInsideCandidateRatio > .01 {
		t.Fatalf("far line should not fit corridor: %+v", farMetrics)
	}
}

func TestGeometryLength(t *testing.T) {
	line := LineString{Type: "LineString", Coordinates: []Point{{30, 60}, {30, 60.001}}}
	if got := geometryLength(line); math.Abs(got-111.2) > 1 {
		t.Fatalf("unexpected length %.2f", got)
	}
}

func TestSelectWaypointsRequiresEndpoints(t *testing.T) {
	points := []Point{{1, 1}, {2, 2}, {3, 3}}
	if _, err := selectWaypoints(points, []int{0, 2}); err != nil {
		t.Fatal(err)
	}
	if _, err := selectWaypoints(points, []int{0, 1}); err == nil {
		t.Fatal("expected missing finish to fail")
	}
}
