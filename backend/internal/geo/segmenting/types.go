package segmenting

import "github.com/doveva/Gulyaem/backend/internal/geo/domain"

type Node struct {
	SourceID int64
	Point    domain.Point
	Tags     map[string]string
}

type Way struct {
	SourceID int64
	NodeIDs  []int64
	Tags     map[string]string
}

type BBox struct {
	West  float64
	South float64
	East  float64
	North float64
}

func (bbox BBox) Valid() bool {
	return bbox.West >= -180 && bbox.East <= 180 && bbox.South >= -90 && bbox.North <= 90 &&
		bbox.West < bbox.East && bbox.South < bbox.North
}

type Input struct {
	Nodes            []Node
	Ways             []Way
	BBox             *BBox
	BBoxes           []BBox
	MaxSegmentLength float64
}

type Report struct {
	CandidateWays                int64
	UnsupportedPedestrianAreas   int64
	SegmentsGenerated            int64
	SegmentsRejected             int64
	SegmentsClipped              int64
	SegmentsDeduplicated         int64
	DuplicateGeometry            int64
	ConflictingDuplicateGeometry int64
	InvalidGeometry              int64
	ZeroLengthSegments           int64
	ShortSegments                int64
	LongSegments                 int64
	ExploreSegments              int64
	RoutableOnlySegments         int64
	IgnoreSegments               int64
	TotalLengthMeters            float64
	ExplorableLengthMeters       float64
	MinLengthMeters              float64
	MedianLengthMeters           float64
	P95LengthMeters              float64
	MaxLengthMeters              float64
	Warnings                     []string
}

type Result struct {
	Segments []domain.StreetSegmentDraft
	Report   Report
}
