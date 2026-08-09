package domain

import "time"

type GeoDataVersionStatus string

const (
	GeoDataVersionImporting  GeoDataVersionStatus = "IMPORTING"
	GeoDataVersionReady      GeoDataVersionStatus = "READY"
	GeoDataVersionFailed     GeoDataVersionStatus = "FAILED"
	GeoDataVersionSuperseded GeoDataVersionStatus = "SUPERSEDED"
)

type GeoDataVersion struct {
	ID                   string
	CityID               string
	CityCode             string
	Source               string
	SourceURL            string
	SourceTimestamp      *time.Time
	SourceChecksum       string
	SourceFileName       string
	SourceSizeBytes      int64
	NormalizationVersion string
	Status               GeoDataVersionStatus
	ImportStartedAt      time.Time
	ImportFinishedAt     *time.Time
	ImportedAt           *time.Time
	ImportReport         ImportReport
	ImportError          string
}

type ImportReport struct {
	Outcome                      string   `json:"outcome"`
	NodesProcessed               int64    `json:"nodesProcessed"`
	WaysProcessed                int64    `json:"waysProcessed"`
	RelationsProcessed           int64    `json:"relationsProcessed"`
	ObjectsProcessed             int64    `json:"objectsProcessed"`
	CandidateWays                int64    `json:"candidateWays"`
	UnsupportedPedestrianAreas   int64    `json:"unsupportedPedestrianAreas"`
	SegmentsGenerated            int64    `json:"segmentsGenerated"`
	SegmentsRejected             int64    `json:"segmentsRejected"`
	SegmentsClipped              int64    `json:"segmentsClipped"`
	SegmentsDeduplicated         int64    `json:"segmentsDeduplicated"`
	DuplicateGeometry            int64    `json:"duplicateGeometry"`
	ConflictingDuplicateGeometry int64    `json:"conflictingDuplicateGeometry"`
	InvalidGeometries            int64    `json:"invalidGeometries"`
	ZeroLengthSegments           int64    `json:"zeroLengthSegments"`
	ShortSegments                int64    `json:"shortSegments"`
	LongSegments                 int64    `json:"longSegments"`
	ExploreSegments              int64    `json:"exploreSegments"`
	RoutableOnlySegments         int64    `json:"routableOnlySegments"`
	IgnoreSegments               int64    `json:"ignoreSegments"`
	TotalLengthMeters            float64  `json:"totalLengthMeters"`
	ExplorableLengthMeters       float64  `json:"explorableLengthMeters"`
	MinSegmentLengthMeters       float64  `json:"minSegmentLengthMeters"`
	MedianSegmentLengthMeters    float64  `json:"medianSegmentLengthMeters"`
	P95SegmentLengthMeters       float64  `json:"p95SegmentLengthMeters"`
	MaxSegmentLengthMeters       float64  `json:"maxSegmentLengthMeters"`
	Warnings                     []string `json:"warnings,omitempty"`
	DurationMillis               int64    `json:"durationMillis"`
}

type BeginImport struct {
	CityCode             string
	Source               string
	SourceURL            string
	SourceChecksum       string
	SourceFileName       string
	SourceSizeBytes      int64
	NormalizationVersion string
}

type BeginImportResult struct {
	Version      GeoDataVersion
	AlreadyReady bool
}
