package pbf

import (
	"context"
	"math"
	"path/filepath"
	"testing"

	"github.com/doveva/Gulyaem/backend/internal/geo/importing"
)

type countingVisitor struct {
	nodes     int64
	ways      int64
	relations int64
}

func (visitor *countingVisitor) VisitNode(importing.SourceNode) error {
	visitor.nodes++
	return nil
}

func (visitor *countingVisitor) VisitWay(importing.SourceWay) error {
	visitor.ways++
	return nil
}

func (visitor *countingVisitor) VisitRelation(importing.SourceRelation) error {
	visitor.relations++
	return nil
}

func TestScannerReadsCommittedDenseCenterFixture(t *testing.T) {
	path := filepath.Join("..", "..", "..", "..", "..", "data", "test-areas", "spb-dense-center", "spb-dense-center.osm.pbf")
	visitor := &countingVisitor{}
	metadata, err := NewScanner().Scan(context.Background(), path, visitor)
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}
	if visitor.nodes != 40041 || visitor.ways != 8667 || visitor.relations != 1520 {
		t.Fatalf("unexpected counts: nodes=%d ways=%d relations=%d", visitor.nodes, visitor.ways, visitor.relations)
	}
	if metadata.BBox == nil || !closeCoordinate(metadata.BBox.West, 30.3) || !closeCoordinate(metadata.BBox.South, 59.93) || !closeCoordinate(metadata.BBox.East, 30.33) || !closeCoordinate(metadata.BBox.North, 59.945) {
		t.Fatalf("unexpected bbox: %+v", metadata.BBox)
	}
}

func closeCoordinate(got, want float64) bool {
	return math.Abs(got-want) < 1e-9
}
