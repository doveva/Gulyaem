package preview

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"time"

	"github.com/doveva/Gulyaem/backend/internal/geo/querying"
	"github.com/doveva/Gulyaem/backend/internal/geo/routeanalysis"
	"github.com/doveva/Gulyaem/backend/internal/routing/port"
)

// lowRouteMatchWarningThreshold is the Stage 2 diagnostic boundary selected
// from validation evidence: normal routes match at about 99-100%, while the
// intentionally ambiguous fixture matches at about 91.4%.
const lowRouteMatchWarningThreshold = 0.95

var (
	ErrInvalidRequest  = errors.New("invalid route preview request")
	ErrGeoUnavailable  = errors.New("geo data unavailable")
	ErrDatasetMismatch = errors.New("routing and geo datasets are incompatible")
)

type Request struct {
	CityID    string       `json:"cityId"`
	Profile   string       `json:"profile"`
	Waypoints []port.Point `json:"waypoints"`
}

type GeoAnalyzer interface {
	CurrentVersion(context.Context, string) (querying.Version, error)
	AnalyzeGeometryForVersion(context.Context, querying.Version, string, json.RawMessage, routeanalysis.AnalyzeRequest) (routeanalysis.Analysis, error)
}

type Routing struct {
	Engine          string          `json:"engine"`
	Profile         string          `json:"profile"`
	DistanceMeters  float64         `json:"distanceMeters"`
	DurationSeconds float64         `json:"durationSeconds"`
	Geometry        json.RawMessage `json:"geometry"`
	Waypoints       []port.Waypoint `json:"waypoints"`
}

type Metrics struct {
	GeometricCoveredLengthMeters         float64 `json:"geometricCoveredLengthMeters"`
	CompletedNetworkLengthMeters         float64 `json:"completedNetworkLengthMeters"`
	ContextExplorableLengthMeters        float64 `json:"contextExplorableLengthMeters"`
	CompletedNetworkRatio                float64 `json:"completedNetworkRatio"`
	RouteMatchedRatio                    float64 `json:"routeMatchedRatio"`
	RouteUnmatchedLengthMeters           float64 `json:"routeUnmatchedLengthMeters"`
	MatchedExplorableRouteLengthMeters   float64 `json:"matchedExplorableRouteLengthMeters"`
	MatchedRoutableOnlyRouteLengthMeters float64 `json:"matchedRoutableOnlyRouteLengthMeters"`
	CompletedSegmentCount                int     `json:"completedSegmentCount"`
	PartialSegmentCount                  int     `json:"partialSegmentCount"`
}

type ExplorationPreview struct {
	CoverageProfile    routeanalysis.CoverageProfile     `json:"coverageProfile"`
	NormalizedRoute    json.RawMessage                   `json:"normalizedRoute"`
	MatchedFragments   []routeanalysis.MatchedFragment   `json:"matchedFragments"`
	UnmatchedFragments []routeanalysis.UnmatchedFragment `json:"unmatchedFragments"`
	CoverageSegments   []routeanalysis.CoverageSegment   `json:"coverageSegments"`
	Metrics            Metrics                           `json:"metrics"`
}

type Result struct {
	PreviewFingerprint string                         `json:"previewFingerprint"`
	GeoDataVersion     routeanalysis.VersionReference `json:"geoDataVersion"`
	Routing            Routing                        `json:"routing"`
	ExplorationPreview ExplorationPreview             `json:"explorationPreview"`
	Warnings           []string                       `json:"warnings"`
	Materialization    MaterializationProvenance      `json:"-"`
}

// MaterializationProvenance is server-only data required to persist and later
// rebuild a trusted Route. It deliberately is not part of the public API.
type MaterializationProvenance struct {
	RoutingMetadata port.Metadata
	AnalysisVersion string
	Matching        routeanalysis.MatchingParameters
}

type Service struct {
	engine   port.RoutingEngine
	analyzer GeoAnalyzer
	logger   *slog.Logger
}

func NewService(engine port.RoutingEngine, analyzer GeoAnalyzer, logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{engine: engine, analyzer: analyzer, logger: logger}
}

func (service *Service) Create(ctx context.Context, request Request) (Result, error) {
	if err := validate(request); err != nil {
		return Result{}, err
	}
	started := time.Now()
	metadata, version, err := service.compatibleDataset(ctx, request.CityID)
	if err != nil {
		return Result{}, err
	}
	if metadata.Profile != request.Profile {
		return Result{}, ErrDatasetMismatch
	}
	routingStarted := time.Now()
	route, err := service.engine.Route(ctx, port.RouteRequest{Profile: request.Profile, Waypoints: request.Waypoints})
	routingDuration := time.Since(routingStarted)
	if err != nil {
		return Result{}, err
	}
	analysisStarted := time.Now()
	analysis, err := service.analyzer.AnalyzeGeometryForVersion(ctx, version, "preview", route.Geometry, routeanalysis.AnalyzeRequest{
		Matching: routeanalysis.DefaultMatchingParameters(), Coverage: routeanalysis.CoverageProfiles["balanced"],
	})
	analysisDuration := time.Since(analysisStarted)
	if err != nil {
		if errors.Is(err, querying.ErrNotFound) {
			return Result{}, ErrGeoUnavailable
		}
		return Result{}, fmt.Errorf("analyze route preview: %w", err)
	}
	if analysis.GeoDataVersion.ID != version.ID || analysis.GeoDataVersion.SourceChecksum != metadata.SourceChecksum {
		service.logger.ErrorContext(ctx, "route preview geo version changed during analysis", "city_id", request.CityID)
		return Result{}, ErrDatasetMismatch
	}
	metrics := metricsFromAnalysis(analysis)
	warnings := make([]string, 0, len(analysis.Warnings)+1)
	warnings = append(warnings, analysis.Warnings...)
	if metrics.RouteMatchedRatio < lowRouteMatchWarningThreshold {
		warnings = append(warnings, "low_route_match")
	}
	service.logger.InfoContext(ctx, "route preview calculated",
		"engine", metadata.Engine, "city_id", request.CityID, "waypoint_count", len(request.Waypoints),
		"routing_duration_ms", routingDuration.Milliseconds(),
		"analysis_duration_ms", analysisDuration.Milliseconds(),
		"total_duration_ms", time.Since(started).Milliseconds(),
		"route_matched_ratio", metrics.RouteMatchedRatio,
	)
	result := Result{
		GeoDataVersion: analysis.GeoDataVersion,
		Routing: Routing{
			Engine: metadata.Engine, Profile: request.Profile, DistanceMeters: route.DistanceMeters,
			DurationSeconds: route.DurationSeconds, Geometry: route.Geometry, Waypoints: route.Waypoints,
		},
		ExplorationPreview: ExplorationPreview{
			CoverageProfile: analysis.CoverageProfile, NormalizedRoute: analysis.NormalizedRoute,
			MatchedFragments: analysis.MatchedFragments, UnmatchedFragments: analysis.UnmatchedFragments,
			CoverageSegments: potentialCoverageSegments(analysis.CoverageSegments), Metrics: metrics,
		},
		Warnings: warnings,
		Materialization: MaterializationProvenance{
			RoutingMetadata: metadata, AnalysisVersion: routeanalysis.AnalysisVersion,
			Matching: analysis.Matching,
		},
	}
	fingerprint, err := fingerprint(request, result)
	if err != nil {
		return Result{}, fmt.Errorf("fingerprint route preview: %w", err)
	}
	result.PreviewFingerprint = fingerprint
	return result, nil
}

func (service *Service) Ready(ctx context.Context) error {
	metadata, err := service.engine.Metadata(ctx)
	if err != nil {
		return err
	}
	_, _, err = service.compatibleDatasetWithMetadata(ctx, metadata.CityID, metadata)
	return err
}

func (service *Service) compatibleDataset(
	ctx context.Context, cityID string,
) (port.Metadata, querying.Version, error) {
	metadata, err := service.engine.Metadata(ctx)
	if err != nil {
		return port.Metadata{}, querying.Version{}, err
	}
	return service.compatibleDatasetWithMetadata(ctx, cityID, metadata)
}

func (service *Service) compatibleDatasetWithMetadata(
	ctx context.Context, cityID string, metadata port.Metadata,
) (port.Metadata, querying.Version, error) {
	if metadata.CityID != cityID {
		service.logger.ErrorContext(ctx, "route preview routing city mismatch", "routing_city_id", metadata.CityID, "city_id", cityID)
		return port.Metadata{}, querying.Version{}, ErrDatasetMismatch
	}
	version, err := service.analyzer.CurrentVersion(ctx, cityID)
	if err != nil {
		if errors.Is(err, querying.ErrNotFound) {
			return port.Metadata{}, querying.Version{}, ErrGeoUnavailable
		}
		return port.Metadata{}, querying.Version{}, fmt.Errorf("resolve current geo data version: %w", err)
	}
	if metadata.SourceChecksum != version.SourceChecksum {
		service.logger.ErrorContext(ctx, "route preview dataset mismatch",
			"engine", metadata.Engine, "routing_checksum", metadata.SourceChecksum,
			"geo_checksum", version.SourceChecksum, "city_id", cityID,
		)
		return port.Metadata{}, querying.Version{}, ErrDatasetMismatch
	}
	return metadata, version, nil
}

func potentialCoverageSegments(segments []routeanalysis.CoverageSegment) []routeanalysis.CoverageSegment {
	result := make([]routeanalysis.CoverageSegment, 0, len(segments))
	for _, segment := range segments {
		if segment.Status == "COMPLETED" || segment.Status == "PARTIAL" || segment.Status == "CONNECTOR" {
			result = append(result, segment)
		}
	}
	return result
}

func validate(request Request) error {
	if request.CityID == "" {
		return fmt.Errorf("%w: cityId is required", ErrInvalidRequest)
	}
	if request.Profile != "pedestrian" {
		return fmt.Errorf("%w: profile must be pedestrian", ErrInvalidRequest)
	}
	if len(request.Waypoints) < 2 || len(request.Waypoints) > 10 {
		return fmt.Errorf("%w: route preview requires between 2 and 10 waypoints", ErrInvalidRequest)
	}
	for _, point := range request.Waypoints {
		if point.Lat < -90 || point.Lat > 90 || point.Lon < -180 || point.Lon > 180 ||
			math.IsNaN(point.Lat) || math.IsNaN(point.Lon) || math.IsInf(point.Lat, 0) || math.IsInf(point.Lon, 0) {
			return fmt.Errorf("%w: waypoint coordinates must be finite and in range", ErrInvalidRequest)
		}
	}
	return nil
}

func metricsFromAnalysis(analysis routeanalysis.Analysis) Metrics {
	metrics := Metrics{
		GeometricCoveredLengthMeters:  analysis.Metrics.GeometricCoveredLengthMeters,
		CompletedNetworkLengthMeters:  analysis.Metrics.CompletedNetworkLengthMeters,
		ContextExplorableLengthMeters: analysis.Metrics.ContextExplorableLengthMeters,
		CompletedNetworkRatio:         analysis.Metrics.CompletedNetworkRatio,
		RouteMatchedRatio:             analysis.Metrics.RouteMatchedRatio,
		RouteUnmatchedLengthMeters:    analysis.Metrics.RouteUnmatchedLengthMeters,
	}
	for _, fragment := range analysis.MatchedFragments {
		length := math.Max(0, fragment.RouteEndMeters-fragment.RouteStartMeters)
		switch fragment.Classification {
		case "EXPLORE":
			metrics.MatchedExplorableRouteLengthMeters += length
		case "ROUTABLE_ONLY":
			metrics.MatchedRoutableOnlyRouteLengthMeters += length
		}
	}
	for _, segment := range analysis.CoverageSegments {
		switch segment.Status {
		case "COMPLETED":
			metrics.CompletedSegmentCount++
		case "PARTIAL":
			metrics.PartialSegmentCount++
		}
	}
	return metrics
}
