package domain

type StreetSegmentClassification string

const (
	StreetSegmentExplore      StreetSegmentClassification = "EXPLORE"
	StreetSegmentRoutableOnly StreetSegmentClassification = "ROUTABLE_ONLY"
	StreetSegmentIgnore       StreetSegmentClassification = "IGNORE"
)

type Point struct {
	Lon float64 `json:"lon"`
	Lat float64 `json:"lat"`
}

type StreetSegmentAttributes struct {
	ReasonCode        string            `json:"reasonCode"`
	SourceTags        map[string]string `json:"sourceTags,omitempty"`
	SourceWayIDs      []int64           `json:"sourceWayIds,omitempty"`
	SourceStartNodeID *int64            `json:"sourceStartNodeId,omitempty"`
	SourceEndNodeID   *int64            `json:"sourceEndNodeId,omitempty"`
	BoundaryClip      bool              `json:"boundaryClip,omitempty"`
	Warnings          []string          `json:"warnings,omitempty"`
}

type StreetSegmentDraft struct {
	Geometry       []Point
	LengthMeters   float64
	Classification StreetSegmentClassification
	Attributes     StreetSegmentAttributes
}
