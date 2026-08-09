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
	Outcome            string `json:"outcome"`
	NodesProcessed     int64  `json:"nodesProcessed"`
	WaysProcessed      int64  `json:"waysProcessed"`
	RelationsProcessed int64  `json:"relationsProcessed"`
	ObjectsProcessed   int64  `json:"objectsProcessed"`
	DurationMillis     int64  `json:"durationMillis"`
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
