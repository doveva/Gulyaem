package routeanalysis

import (
	"context"
	"encoding/json"
	"time"

	"github.com/doveva/Gulyaem/backend/internal/geo/domain"
	"github.com/doveva/Gulyaem/backend/internal/geo/querying"
)

const AnalysisContextRadiusMeters = 75.0

type MatchingParameters struct {
	SampleStepMeters        float64 `json:"sampleStepMeters"`
	CandidateRadiusMeters   float64 `json:"candidateRadiusMeters"`
	MaxDirectionDegrees     float64 `json:"maxDirectionDegrees"`
	EndpointToleranceMeters float64 `json:"endpointToleranceMeters"`
}

func DefaultMatchingParameters() MatchingParameters {
	return MatchingParameters{SampleStepMeters: 5, CandidateRadiusMeters: 12, MaxDirectionDegrees: 55, EndpointToleranceMeters: 2}
}

type CoverageProfile struct {
	Name              string  `json:"name"`
	RadiusMeters      float64 `json:"radiusMeters"`
	CoverageRatio     float64 `json:"coverageRatio"`
	MinRequiredMeters float64 `json:"minRequiredMeters"`
	MaxRequiredMeters float64 `json:"maxRequiredMeters"`
}

var CoverageProfiles = map[string]CoverageProfile{
	"strict":   {Name: "strict", RadiusMeters: 10, CoverageRatio: .8, MinRequiredMeters: 20, MaxRequiredMeters: 120},
	"balanced": {Name: "balanced", RadiusMeters: 20, CoverageRatio: .6, MinRequiredMeters: 15, MaxRequiredMeters: 80},
	"generous": {Name: "generous", RadiusMeters: 35, CoverageRatio: .4, MinRequiredMeters: 10, MaxRequiredMeters: 50},
}

type AnalyzeRequest struct {
	Matching MatchingParameters `json:"matching"`
	Coverage CoverageProfile    `json:"coverage"`
}

type Route struct {
	ID                   string          `json:"id"`
	Name                 string          `json:"name"`
	AreaID               string          `json:"areaId"`
	Description          string          `json:"description"`
	IntentionalUnmatched bool            `json:"intentionalUnmatched"`
	Geometry             json.RawMessage `json:"geometry"`
	Points               []domain.Point  `json:"-"`
}

type RouteCollection struct {
	Routes                       []Route          `json:"routes"`
	GeoDataVersion               VersionReference `json:"geoDataVersion"`
	ExpectedSourceChecksum       string           `json:"expectedSourceChecksum"`
	ExpectedNormalizationVersion string           `json:"expectedNormalizationVersion"`
	Warnings                     []string         `json:"warnings"`
}

type VersionReference struct {
	ID                   string                      `json:"id"`
	CityID               string                      `json:"cityId"`
	SourceChecksum       string                      `json:"sourceChecksum"`
	NormalizationVersion string                      `json:"normalizationVersion"`
	Status               domain.GeoDataVersionStatus `json:"status"`
	ImportedAt           *time.Time                  `json:"importedAt"`
}

type CandidateSegment struct {
	ID                  string
	Geometry            []domain.Point
	GeometryJSON        json.RawMessage
	LengthMeters        float64
	Classification      domain.StreetSegmentClassification
	Attributes          domain.StreetSegmentAttributes
	RadiusCoveredMeters float64
}

type Repository interface {
	CurrentVersion(context.Context, string) (querying.Version, error)
	CandidateSegments(context.Context, string, json.RawMessage, float64) ([]CandidateSegment, error)
	CoverageSegments(context.Context, string, json.RawMessage, float64, float64) ([]CandidateSegment, error)
}

type Score struct {
	DistanceScore   float64 `json:"distanceScore"`
	DirectionScore  float64 `json:"directionScore"`
	ContinuityScore float64 `json:"continuityScore"`
	Confidence      float64 `json:"confidence"`
	Reason          string  `json:"reason"`
}

type MatchedFragment struct {
	SegmentID        string                             `json:"segmentId"`
	Classification   domain.StreetSegmentClassification `json:"classification"`
	Geometry         json.RawMessage                    `json:"geometry"`
	RouteStartMeters float64                            `json:"routeStartMeters"`
	RouteEndMeters   float64                            `json:"routeEndMeters"`
	Score            Score                              `json:"score"`
}

type UnmatchedFragment struct {
	Reason      string          `json:"reason"`
	Geometry    json.RawMessage `json:"geometry"`
	StartMeters float64         `json:"startMeters"`
	EndMeters   float64         `json:"endMeters"`
}

type CoverageSegment struct {
	SegmentID      string                             `json:"segmentId"`
	Classification domain.StreetSegmentClassification `json:"classification"`
	Geometry       json.RawMessage                    `json:"geometry"`
	LengthMeters   float64                            `json:"lengthMeters"`
	CoveredMeters  float64                            `json:"coveredMeters"`
	DirectMeters   float64                            `json:"directMeters"`
	RequiredMeters float64                            `json:"requiredMeters"`
	Status         string                             `json:"status"`
	Provenance     string                             `json:"provenance,omitempty"`
}

type Metrics struct {
	GeometricCoveredLengthMeters  float64 `json:"geometricCoveredLengthMeters"`
	CompletedNetworkLengthMeters  float64 `json:"completedNetworkLengthMeters"`
	ContextExplorableLengthMeters float64 `json:"contextExplorableLengthMeters"`
	CompletedNetworkRatio         float64 `json:"completedNetworkRatio"`
	RouteMatchedRatio             float64 `json:"routeMatchedRatio"`
	RouteUnmatchedLengthMeters    float64 `json:"routeUnmatchedLengthMeters"`
}

type Analysis struct {
	RouteID             string              `json:"routeId"`
	GeoDataVersion      VersionReference    `json:"geoDataVersion"`
	Warnings            []string            `json:"warnings"`
	Matching            MatchingParameters  `json:"matching"`
	CoverageProfile     CoverageProfile     `json:"coverageProfile"`
	ContextRadiusMeters float64             `json:"contextRadiusMeters"`
	SourceRoute         json.RawMessage     `json:"sourceRoute"`
	NormalizedRoute     json.RawMessage     `json:"normalizedRoute"`
	MatchedFragments    []MatchedFragment   `json:"matchedFragments"`
	UnmatchedFragments  []UnmatchedFragment `json:"unmatchedFragments"`
	CoverageSegments    []CoverageSegment   `json:"coverageSegments"`
	Metrics             Metrics             `json:"metrics"`
}
