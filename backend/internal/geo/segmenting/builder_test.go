package segmenting

import (
	"math"
	"strings"
	"testing"

	"github.com/doveva/Gulyaem/backend/internal/geo/domain"
)

func TestBuildSimpleLineAndMergeAcrossWayBoundary(t *testing.T) {
	input := Input{
		Nodes: testNodes(
			testNode(1, 0, 0),
			testNode(2, 0.001, 0),
			testNode(3, 0.002, 0),
		),
		Ways: []Way{
			{SourceID: 10, NodeIDs: []int64{1, 2}, Tags: map[string]string{"highway": "footway"}},
			{SourceID: 11, NodeIDs: []int64{2, 3}, Tags: map[string]string{"highway": "footway"}},
		},
	}

	result := mustBuild(t, input)
	if len(result.Segments) != 1 {
		t.Fatalf("segments = %d, want 1", len(result.Segments))
	}
	segment := result.Segments[0]
	if len(segment.Geometry) != 3 || segment.Classification != domain.StreetSegmentExplore {
		t.Fatalf("segment = %+v", segment)
	}
	if len(segment.Attributes.SourceWayIDs) != 2 {
		t.Fatalf("source ways = %v", segment.Attributes.SourceWayIDs)
	}
}

func TestBuildSplitsTJunction(t *testing.T) {
	result := mustBuild(t, Input{
		Nodes: testNodes(testNode(1, 0, 0), testNode(2, 0.001, 0), testNode(3, 0.002, 0), testNode(4, 0.001, 0.001)),
		Ways: []Way{
			{SourceID: 10, NodeIDs: []int64{1, 2, 3}, Tags: map[string]string{"highway": "residential"}},
			{SourceID: 11, NodeIDs: []int64{2, 4}, Tags: map[string]string{"highway": "residential"}},
		},
	})
	if len(result.Segments) != 3 {
		t.Fatalf("segments = %d, want 3", len(result.Segments))
	}
}

func TestGeometryCrossingWithoutSharedNodeDoesNotSplit(t *testing.T) {
	result := mustBuild(t, Input{
		Nodes: testNodes(testNode(1, 0, 0), testNode(2, 0.002, 0.002), testNode(3, 0, 0.002), testNode(4, 0.002, 0)),
		Ways: []Way{
			{SourceID: 10, NodeIDs: []int64{1, 2}, Tags: map[string]string{"highway": "footway"}},
			{SourceID: 11, NodeIDs: []int64{3, 4}, Tags: map[string]string{"highway": "footway"}},
		},
	})
	if len(result.Segments) != 2 {
		t.Fatalf("segments = %d, want 2", len(result.Segments))
	}
}

func TestSemanticChangeAndBarrierSplit(t *testing.T) {
	nodes := testNodes(testNode(1, 0, 0), testNodeWithTags(2, 0.001, 0, map[string]string{"barrier": "gate"}), testNode(3, 0.002, 0), testNode(4, 0.003, 0))
	result := mustBuild(t, Input{
		Nodes: nodes,
		Ways: []Way{
			{SourceID: 10, NodeIDs: []int64{1, 2, 3}, Tags: map[string]string{"highway": "footway"}},
			{SourceID: 11, NodeIDs: []int64{3, 4}, Tags: map[string]string{"highway": "footway", "footway": "crossing"}},
		},
	})
	if len(result.Segments) != 3 {
		t.Fatalf("segments = %d, want 3", len(result.Segments))
	}
}

func TestClosedLoopRemainsOneSegment(t *testing.T) {
	result := mustBuild(t, Input{
		Nodes: testNodes(testNode(1, 0, 0), testNode(2, 0.001, 0), testNode(3, 0.001, 0.001)),
		Ways:  []Way{{SourceID: 10, NodeIDs: []int64{1, 2, 3, 1}, Tags: map[string]string{"highway": "path"}}},
	})
	if len(result.Segments) != 1 {
		t.Fatalf("segments = %d, want 1", len(result.Segments))
	}
	geometry := result.Segments[0].Geometry
	if !pointsEqual(geometry[0], geometry[len(geometry)-1]) {
		t.Fatalf("loop is not closed: %v", geometry)
	}
}

func TestBBoxClippingCreatesBoundarySegment(t *testing.T) {
	bbox := &BBox{West: 0, South: 0, East: 1, North: 1}
	result := mustBuild(t, Input{
		Nodes: testNodes(testNode(1, -1, 0.5), testNode(2, 2, 0.5)),
		Ways:  []Way{{SourceID: 10, NodeIDs: []int64{1, 2}, Tags: map[string]string{"highway": "footway"}}},
		BBox:  bbox,
	})
	if len(result.Segments) != 1 || !result.Segments[0].Attributes.BoundaryClip {
		t.Fatalf("result = %+v", result)
	}
	if !pointsEqual(result.Segments[0].Geometry[0], domain.Point{Lon: 0, Lat: 0.5}) || !pointsEqual(result.Segments[0].Geometry[1], domain.Point{Lon: 1, Lat: 0.5}) {
		t.Fatalf("geometry = %v", result.Segments[0].Geometry)
	}
}

func TestMissingReferencedNodeFailsBuild(t *testing.T) {
	_, err := Build(Input{
		Nodes: testNodes(testNode(1, 0, 0)),
		Ways:  []Way{{SourceID: 10, NodeIDs: []int64{1, 2}, Tags: map[string]string{"highway": "footway"}}},
	})
	if err == nil || !strings.Contains(err.Error(), "missing node 2") {
		t.Fatalf("error = %v", err)
	}
}

func TestExactDuplicateIsMerged(t *testing.T) {
	result := mustBuild(t, Input{
		Nodes: testNodes(testNode(1, 0, 0), testNode(2, 0.001, 0)),
		Ways: []Way{
			{SourceID: 10, NodeIDs: []int64{1, 2}, Tags: map[string]string{"highway": "footway"}},
			{SourceID: 11, NodeIDs: []int64{2, 1}, Tags: map[string]string{"highway": "footway"}},
		},
	})
	if len(result.Segments) != 1 || result.Report.SegmentsDeduplicated != 1 || result.Report.DuplicateGeometry != 1 {
		t.Fatalf("result = %+v", result)
	}
}

func TestConflictingDuplicateRemainsInspectable(t *testing.T) {
	result := mustBuild(t, Input{
		Nodes: testNodes(testNode(1, 0, 0), testNode(2, 0.001, 0)),
		Ways: []Way{
			{SourceID: 10, NodeIDs: []int64{1, 2}, Tags: map[string]string{"highway": "footway"}},
			{SourceID: 11, NodeIDs: []int64{2, 1}, Tags: map[string]string{"highway": "footway", "footway": "crossing"}},
		},
	})
	if len(result.Segments) != 2 || result.Report.ConflictingDuplicateGeometry != 1 {
		t.Fatalf("result = %+v", result)
	}
}

func TestSharedNodeIntersectionCreatesFourSegments(t *testing.T) {
	result := mustBuild(t, Input{
		Nodes: testNodes(
			testNode(1, -0.001, 0), testNode(2, 0, 0), testNode(3, 0.001, 0),
			testNode(4, 0, -0.001), testNode(5, 0, 0.001),
		),
		Ways: []Way{
			{SourceID: 10, NodeIDs: []int64{1, 2, 3}, Tags: map[string]string{"highway": "residential"}},
			{SourceID: 11, NodeIDs: []int64{4, 2, 5}, Tags: map[string]string{"highway": "residential"}},
		},
	})
	if len(result.Segments) != 4 {
		t.Fatalf("segments = %d, want 4", len(result.Segments))
	}
}

func TestDegenerateCandidateIsRejected(t *testing.T) {
	result := mustBuild(t, Input{
		Nodes: testNodes(testNode(1, 0, 0)),
		Ways:  []Way{{SourceID: 10, NodeIDs: []int64{1}, Tags: map[string]string{"highway": "path"}}},
	})
	if len(result.Segments) != 0 || result.Report.SegmentsRejected != 1 || result.Report.InvalidGeometry != 1 {
		t.Fatalf("result = %+v", result)
	}
}

func TestMaximumLengthHandlesExistingVertexAtBoundary(t *testing.T) {
	firstLegDegrees := 100 / 111195.0802335329
	result := mustBuild(t, Input{
		Nodes:            testNodes(testNode(1, 0, 0), testNode(2, firstLegDegrees, 0), testNode(3, firstLegDegrees+0.0005, 0)),
		Ways:             []Way{{SourceID: 10, NodeIDs: []int64{1, 2, 3}, Tags: map[string]string{"highway": "path"}}},
		MaxSegmentLength: 100,
	})
	for _, segment := range result.Segments {
		for index := 1; index < len(segment.Geometry); index++ {
			if pointsEqual(segment.Geometry[index-1], segment.Geometry[index]) {
				t.Fatalf("duplicate adjacent coordinate: %v", segment.Geometry)
			}
		}
	}
}

func TestMaximumLengthSplitsWithoutChangingClassification(t *testing.T) {
	result := mustBuild(t, Input{
		Nodes:            testNodes(testNode(1, 0, 0), testNode(2, 0.003, 0)),
		Ways:             []Way{{SourceID: 10, NodeIDs: []int64{1, 2}, Tags: map[string]string{"highway": "steps", "oneway:foot": "yes"}}},
		MaxSegmentLength: 100,
	})
	if len(result.Segments) != 4 {
		t.Fatalf("segments = %d, want 4", len(result.Segments))
	}
	for _, segment := range result.Segments {
		if segment.LengthMeters > 100.001 {
			t.Fatalf("length = %f", segment.LengthMeters)
		}
		if segment.Classification != domain.StreetSegmentExplore || segment.Attributes.SourceTags["oneway:foot"] != "yes" {
			t.Fatalf("segment metadata changed: %+v", segment)
		}
	}
}

func mustBuild(t *testing.T, input Input) Result {
	t.Helper()
	result, err := Build(input)
	if err != nil {
		t.Fatal(err)
	}
	for _, segment := range result.Segments {
		if len(segment.Geometry) < 2 || segment.LengthMeters <= 0 || math.IsNaN(segment.LengthMeters) {
			t.Fatalf("invalid segment: %+v", segment)
		}
	}
	return result
}

func testNodes(nodes ...Node) []Node {
	return nodes
}

func testNode(id int64, lon, lat float64) Node {
	return Node{SourceID: id, Point: domain.Point{Lon: lon, Lat: lat}}
}

func testNodeWithTags(id int64, lon, lat float64, tags map[string]string) Node {
	return Node{SourceID: id, Point: domain.Point{Lon: lon, Lat: lat}, Tags: tags}
}
