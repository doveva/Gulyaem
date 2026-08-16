package walks

import (
	"encoding/json"
	"time"

	"github.com/doveva/Gulyaem/backend/internal/geo/routeanalysis"
	"github.com/doveva/Gulyaem/backend/internal/routing/port"
)

type Status string

const (
	StatusDraft     Status = "DRAFT"
	StatusActive    Status = "ACTIVE"
	StatusReview    Status = "REVIEW"
	StatusCompleted Status = "COMPLETED"
	StatusCancelled Status = "CANCELLED"
)

type Route struct {
	ID                         string          `json:"id"`
	ActorID                    string          `json:"-"`
	CityID                     string          `json:"cityId"`
	GeoDataVersionID           string          `json:"geoDataVersionId"`
	Profile                    string          `json:"profile"`
	Waypoints                  []port.Point    `json:"waypoints"`
	Geometry                   json.RawMessage `json:"geometry"`
	NormalizedGeometry         json.RawMessage `json:"normalizedGeometry,omitempty"`
	DistanceMeters             float64         `json:"distanceMeters"`
	EstimatedDurationSeconds   int             `json:"estimatedDurationSeconds"`
	RoutingProvenance          json.RawMessage `json:"-"`
	AnalysisProvenance         json.RawMessage `json:"-"`
	MaterializationFingerprint string          `json:"-"`
	Revision                   int             `json:"revision"`
	FinalizedAt                *time.Time      `json:"finalizedAt,omitempty"`
	CreatedAt                  time.Time       `json:"createdAt"`
	UpdatedAt                  time.Time       `json:"updatedAt"`
}

type SegmentMatch struct {
	SegmentID      string
	Classification string
	MatchedMeters  float64
	CoveredMeters  float64
	DirectMeters   float64
	RequiredMeters float64
	Status         string
	Provenance     string
	Confidence     *float64
}

type Walk struct {
	ID                 string     `json:"id"`
	ActorID            string     `json:"-"`
	CityID             string     `json:"cityId"`
	RouteID            string     `json:"routeId"`
	ClientRequestID    string     `json:"-"`
	RequestFingerprint string     `json:"-"`
	Status             Status     `json:"status"`
	StartedAt          *time.Time `json:"startedAt,omitempty"`
	FinishedAt         *time.Time `json:"finishedAt,omitempty"`
	CompletedAt        *time.Time `json:"completedAt,omitempty"`
	DurationSeconds    *int       `json:"durationSeconds,omitempty"`
	DistanceMeters     *float64   `json:"distanceMeters,omitempty"`
	CreatedAt          time.Time  `json:"createdAt"`
	UpdatedAt          time.Time  `json:"updatedAt"`
}

type Aggregate struct {
	Walk  Walk  `json:"walk"`
	Route Route `json:"route"`
}

type CreateRequest struct {
	ClientRequestID            string       `json:"clientRequestId"`
	CityID                     string       `json:"cityId"`
	Profile                    string       `json:"profile"`
	ExpectedPreviewFingerprint string       `json:"expectedPreviewFingerprint"`
	Waypoints                  []port.Point `json:"waypoints"`
}

type CorrectRouteRequest struct {
	Profile                    string       `json:"profile"`
	ExpectedPreviewFingerprint string       `json:"expectedPreviewFingerprint"`
	Waypoints                  []port.Point `json:"waypoints"`
}

type Materialization struct {
	Route   Route
	Matches []SegmentMatch
}

type CompletionSummary struct {
	GeoDataVersionID       string          `json:"geoDataVersionId"`
	NewSegmentsCount       int             `json:"newSegmentsCount"`
	RevisitedSegmentsCount int             `json:"revisitedSegmentsCount"`
	NewNetworkLengthMeters float64         `json:"newNetworkLengthMeters"`
	NewSegments            json.RawMessage `json:"newSegments"`
	Districts              []DistrictDelta `json:"districts"`
}

type DistrictDelta struct {
	DistrictID       string  `json:"districtId"`
	Name             string  `json:"name"`
	PercentageBefore float64 `json:"percentageBefore"`
	PercentageAfter  float64 `json:"percentageAfter"`
	NewLengthMeters  float64 `json:"newLengthMeters"`
}

type CompletionResult struct {
	Walk        Walk              `json:"walk"`
	Exploration CompletionSummary `json:"exploration"`
}

type analysisProvenance struct {
	Version         string                           `json:"analysisVersion"`
	Matching        routeanalysis.MatchingParameters `json:"matchingParameters"`
	CoverageProfile routeanalysis.CoverageProfile    `json:"coverageProfile"`
	Normalization   string                           `json:"normalizationVersion"`
}
