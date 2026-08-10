package domain

import (
	"encoding/json"
	"time"
)

type DistrictDataVersion struct {
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
	ImportReport         DistrictImportReport
	ImportError          string
}

type DistrictImportReport struct {
	Outcome            string `json:"outcome"`
	FeaturesProcessed  int64  `json:"featuresProcessed"`
	DistrictsPublished int64  `json:"districtsPublished"`
	DurationMillis     int64  `json:"durationMillis"`
}

type DistrictDraft struct {
	ExternalID   string
	Name         string
	Kind         string
	GeometryJSON json.RawMessage
	Attributes   map[string]any
}

type BeginDistrictImport struct {
	CityCode             string
	Source               string
	SourceURL            string
	SourceChecksum       string
	SourceFileName       string
	SourceSizeBytes      int64
	NormalizationVersion string
}

type BeginDistrictImportResult struct {
	Version      DistrictDataVersion
	AlreadyReady bool
}
