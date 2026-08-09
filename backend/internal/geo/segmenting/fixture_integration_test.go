package segmenting_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/doveva/Gulyaem/backend/internal/geo/domain"
	"github.com/doveva/Gulyaem/backend/internal/geo/importing"
	"github.com/doveva/Gulyaem/backend/internal/geo/segmenting"
	"github.com/doveva/Gulyaem/backend/internal/platform/osm/pbf"
)

type fixtureCollector struct {
	nodes []segmenting.Node
	ways  []segmenting.Way
}

func (collector *fixtureCollector) VisitNode(node importing.SourceNode) error {
	collector.nodes = append(collector.nodes, segmenting.Node{
		SourceID: node.SourceID,
		Point:    domain.Point{Lon: node.Lon, Lat: node.Lat},
		Tags:     node.Tags,
	})
	return nil
}

func (collector *fixtureCollector) VisitWay(way importing.SourceWay) error {
	collector.ways = append(collector.ways, segmenting.Way{
		SourceID: way.SourceID,
		NodeIDs:  way.NodeIDs,
		Tags:     way.Tags,
	})
	return nil
}

func (*fixtureCollector) VisitRelation(importing.SourceRelation) error {
	return nil
}

func TestDenseCenterFixtureInvariants(t *testing.T) {
	path := filepath.Join("..", "..", "..", "..", "data", "test-areas", "spb-dense-center", "spb-dense-center.osm.pbf")
	collector := &fixtureCollector{}
	if _, err := pbf.NewScanner().Scan(context.Background(), path, collector); err != nil {
		t.Fatal(err)
	}

	result, err := segmenting.Build(segmenting.Input{
		Nodes: collector.nodes,
		Ways:  collector.ways,
		BBox: &segmenting.BBox{
			West: 30.3, South: 59.93, East: 30.33, North: 59.945,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Report.SegmentsGenerated != 6558 || result.Report.ExploreSegments != 2649 ||
		result.Report.RoutableOnlySegments != 2338 || result.Report.IgnoreSegments != 1571 {
		t.Fatalf("unexpected fixture report: %+v", result.Report)
	}
	if result.Report.InvalidGeometry != 0 || result.Report.ZeroLengthSegments != 0 {
		t.Fatalf("fixture contains invalid published candidates: %+v", result.Report)
	}
	for _, segment := range result.Segments {
		if len(segment.Geometry) < 2 || segment.LengthMeters <= 0 {
			t.Fatalf("invalid segment: %+v", segment)
		}
		for _, point := range segment.Geometry {
			if point.Lon < 30.3 || point.Lon > 30.33 || point.Lat < 59.93 || point.Lat > 59.945 {
				t.Fatalf("point outside fixture bbox: %+v", point)
			}
		}
	}
}
