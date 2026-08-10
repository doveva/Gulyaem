package routingspike

import (
	"encoding/json"
	"time"
)

const SchemaVersion = 1

type Point [2]float64

type LineString struct {
	Type        string  `json:"type"`
	Coordinates []Point `json:"coordinates"`
}

type Fixture struct {
	SchemaVersion int `json:"schemaVersion"`
	Dataset       struct {
		GeoFixture   string `json:"geoFixture"`
		PBFFile      string `json:"pbfFile"`
		PBFSHA256    string `json:"pbfSha256"`
		RouteFixture string `json:"routeFixture"`
	} `json:"dataset"`
	Benchmark struct {
		WarmRequests     int     `json:"warmRequests"`
		CorridorMeters   float64 `json:"corridorMeters"`
		SampleStepMeters float64 `json:"sampleStepMeters"`
	} `json:"benchmark"`
	Cases []struct {
		RouteID         string `json:"routeId"`
		WaypointIndexes []int  `json:"waypointIndexes"`
		Note            string `json:"note"`
	} `json:"cases"`
	MapMatchingRouteIDs []string `json:"mapMatchingRouteIds"`
}

type ReferenceRoute struct {
	ID                   string
	Name                 string
	AreaID               string
	Description          string
	IntentionalUnmatched bool
	Geometry             LineString
}

type Engine struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	Version            string `json:"version"`
	Profile            string `json:"profile"`
	BaseURL            string `json:"baseUrl"`
	License            string `json:"license"`
	MapMatchingSurface string `json:"mapMatchingSurface"`
}

type SetupMetric struct {
	ReadySeconds    float64 `json:"readySeconds"`
	PeakMemoryBytes int64   `json:"peakMemoryBytes"`
	IdleMemoryBytes int64   `json:"idleMemoryBytes"`
	GraphBytes      int64   `json:"graphBytes"`
	GraphReused     bool    `json:"graphReused"`
}

type SetupMetrics struct {
	MeasuredAt time.Time              `json:"measuredAt"`
	Host       map[string]string      `json:"host"`
	Engines    map[string]SetupMetric `json:"engines"`
}

type Latency struct {
	FirstMilliseconds float64 `json:"firstMilliseconds"`
	P50Milliseconds   float64 `json:"p50Milliseconds"`
	P95Milliseconds   float64 `json:"p95Milliseconds"`
	WarmRequests      int     `json:"warmRequests"`
}

type CorridorMetrics struct {
	CandidateInsideReferenceRatio float64 `json:"candidateInsideReferenceRatio"`
	ReferenceInsideCandidateRatio float64 `json:"referenceInsideCandidateRatio"`
}

type MatcherMetrics struct {
	RouteMatchedRatio          float64            `json:"routeMatchedRatio"`
	RouteUnmatchedLengthMeters float64            `json:"routeUnmatchedLengthMeters"`
	MatchedReasonMeters        map[string]float64 `json:"matchedReasonMeters"`
}

type RouteResult struct {
	EngineID             string          `json:"engineId"`
	Status               string          `json:"status"`
	Error                string          `json:"error,omitempty"`
	DistanceMeters       float64         `json:"distanceMeters,omitempty"`
	DurationSeconds      float64         `json:"durationSeconds,omitempty"`
	GeometryLengthMeters float64         `json:"geometryLengthMeters,omitempty"`
	ResponseBytes        int             `json:"responseBytes,omitempty"`
	Geometry             *LineString     `json:"geometry,omitempty"`
	Latency              Latency         `json:"latency"`
	Corridor             CorridorMetrics `json:"corridor"`
	Matcher              *MatcherMetrics `json:"matcher,omitempty"`
}

type CaseResult struct {
	RouteID               string        `json:"routeId"`
	Name                  string        `json:"name"`
	AreaID                string        `json:"areaId"`
	Description           string        `json:"description"`
	IntentionalUnmatched  bool          `json:"intentionalUnmatched"`
	Note                  string        `json:"note"`
	Waypoints             []Point       `json:"waypoints"`
	ReferenceGeometry     LineString    `json:"referenceGeometry"`
	ReferenceLengthMeters float64       `json:"referenceLengthMeters"`
	Results               []RouteResult `json:"results"`
}

type MapMatchResult struct {
	RouteID             string          `json:"routeId"`
	EngineID            string          `json:"engineId"`
	Status              string          `json:"status"`
	Error               string          `json:"error,omitempty"`
	Geometry            *LineString     `json:"geometry,omitempty"`
	LatencyMilliseconds float64         `json:"latencyMilliseconds,omitempty"`
	ResponseBytes       int             `json:"responseBytes,omitempty"`
	RouteMatchedRatio   float64         `json:"routeMatchedRatio,omitempty"`
	Corridor            CorridorMetrics `json:"corridor"`
}

type EngineSummary struct {
	EngineID                    string  `json:"engineId"`
	SuccessfulRoutes            int     `json:"successfulRoutes"`
	TotalRoutes                 int     `json:"totalRoutes"`
	MeanCandidateCorridorRatio  float64 `json:"meanCandidateCorridorRatio"`
	MeanReferenceCorridorRatio  float64 `json:"meanReferenceCorridorRatio"`
	MeanStreetSegmentMatchRatio float64 `json:"meanStreetSegmentMatchRatio"`
	MedianWarmLatencyMs         float64 `json:"medianWarmLatencyMs"`
	MapMatchingStatus           string  `json:"mapMatchingStatus"`
}

type Report struct {
	SchemaVersion int              `json:"schemaVersion"`
	Status        string           `json:"status"`
	GeneratedAt   time.Time        `json:"generatedAt"`
	Dataset       FixtureDataset   `json:"dataset"`
	Benchmark     FixtureBenchmark `json:"benchmark"`
	Engines       []Engine         `json:"engines"`
	Setup         *SetupMetrics    `json:"setup,omitempty"`
	Cases         []CaseResult     `json:"cases"`
	MapMatching   []MapMatchResult `json:"mapMatching"`
	Summary       []EngineSummary  `json:"summary"`
	Notes         []string         `json:"notes"`
}

type FixtureDataset struct {
	GeoFixture   string `json:"geoFixture"`
	PBFFile      string `json:"pbfFile"`
	PBFSHA256    string `json:"pbfSha256"`
	RouteFixture string `json:"routeFixture"`
}

type FixtureBenchmark struct {
	WarmRequests     int     `json:"warmRequests"`
	CorridorMeters   float64 `json:"corridorMeters"`
	SampleStepMeters float64 `json:"sampleStepMeters"`
}

type rawRouteResponse struct {
	Geometry        LineString
	DistanceMeters  float64
	DurationSeconds float64
	ResponseBytes   int
}

type Analyzer interface {
	Analyze(routeID string, geometry json.RawMessage) (*MatcherMetrics, error)
}
