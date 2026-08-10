package routeanalysis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/doveva/Gulyaem/backend/internal/geo/domain"
	"github.com/doveva/Gulyaem/backend/internal/geo/querying"
)

var (
	ErrRouteNotFound     = errors.New("sample route not found")
	ErrInvalidParameters = errors.New("invalid route analysis parameters")
)

type Service struct {
	repository Repository
	fixtures   fixtureSet
}

func NewService(repository Repository, dataRoot string) (*Service, error) {
	fixtures, err := loadFixtureSet(dataRoot)
	if err != nil {
		return nil, err
	}
	return &Service{repository: repository, fixtures: fixtures}, nil
}

func (service *Service) Routes(ctx context.Context, cityID string) (RouteCollection, error) {
	version, err := service.repository.CurrentVersion(ctx, cityID)
	if err != nil {
		return RouteCollection{}, err
	}
	return RouteCollection{
		Routes: service.fixtures.routes, GeoDataVersion: versionReference(version),
		ExpectedSourceChecksum:       service.fixtures.manifest.ExpectedSourceChecksum,
		ExpectedNormalizationVersion: service.fixtures.manifest.ExpectedNormalizationVersion,
		Warnings:                     service.versionWarnings(version.SourceChecksum, version.NormalizationVersion),
	}, nil
}

func (service *Service) Analyze(ctx context.Context, cityID, routeID string, request AnalyzeRequest) (Analysis, error) {
	route, found := service.fixtures.byID[routeID]
	if !found {
		return Analysis{}, ErrRouteNotFound
	}
	return service.analyze(ctx, cityID, route, request)
}

// AnalyzeGeometry runs the Stage 1 matcher and coverage calculation for a
// routing-engine result without making engine edge IDs part of the domain.
func (service *Service) AnalyzeGeometry(
	ctx context.Context, cityID, routeID string, geometry json.RawMessage, request AnalyzeRequest,
) (Analysis, error) {
	var line lineStringGeometry
	if err := json.Unmarshal(geometry, &line); err != nil || line.Type != "LineString" || len(line.Coordinates) < 2 {
		return Analysis{}, fmt.Errorf("%w: route geometry must be a GeoJSON LineString", ErrInvalidParameters)
	}
	points := make([]domain.Point, len(line.Coordinates))
	for index, coordinate := range line.Coordinates {
		if len(coordinate) < 2 || math.IsNaN(coordinate[0]) || math.IsNaN(coordinate[1]) ||
			math.IsInf(coordinate[0], 0) || math.IsInf(coordinate[1], 0) {
			return Analysis{}, fmt.Errorf("%w: route geometry contains an invalid coordinate", ErrInvalidParameters)
		}
		points[index] = domain.Point{Lon: coordinate[0], Lat: coordinate[1]}
	}
	return service.analyze(ctx, cityID, Route{ID: routeID, Geometry: geometry, Points: points}, request)
}

func (service *Service) analyze(ctx context.Context, cityID string, route Route, request AnalyzeRequest) (Analysis, error) {
	if err := validateAnalyzeRequest(request); err != nil {
		return Analysis{}, err
	}
	version, err := service.repository.CurrentVersion(ctx, cityID)
	if err != nil {
		return Analysis{}, err
	}
	candidates, err := service.repository.CandidateSegments(ctx, cityID, route.Geometry, AnalysisContextRadiusMeters)
	if err != nil {
		return Analysis{}, err
	}
	matched, unmatched, normalized, direct, matchedMeters := matchRoute(route.Points, candidates, request.Matching)
	coverageCandidates := []CandidateSegment{}
	if len(normalized) > 0 {
		coverageCandidates, err = service.repository.CoverageSegments(
			ctx, cityID, multiLineGeometryJSON(normalized), request.Coverage.RadiusMeters, AnalysisContextRadiusMeters,
		)
		if err != nil {
			return Analysis{}, err
		}
	}
	coverage, metrics := calculateCoverage(coverageCandidates, direct, request.Coverage)
	totalLength := routeLength(route.Points)
	if totalLength > 0 {
		metrics.RouteMatchedRatio = math.Min(1, matchedMeters/totalLength)
		metrics.RouteUnmatchedLengthMeters = math.Max(0, totalLength-matchedMeters)
	}
	return Analysis{
		RouteID: route.ID, GeoDataVersion: versionReference(version),
		Warnings: service.versionWarnings(version.SourceChecksum, version.NormalizationVersion),
		Matching: request.Matching, CoverageProfile: request.Coverage,
		ContextRadiusMeters: AnalysisContextRadiusMeters,
		SourceRoute:         route.Geometry, NormalizedRoute: multiLineGeometryJSON(normalized),
		MatchedFragments: matched, UnmatchedFragments: unmatched,
		CoverageSegments: coverage, Metrics: metrics,
	}, nil
}

func versionReference(version querying.Version) VersionReference {
	return VersionReference{
		ID: version.ID, CityID: version.CityID, SourceChecksum: version.SourceChecksum,
		NormalizationVersion: version.NormalizationVersion, Status: version.Status, ImportedAt: version.ImportedAt,
	}
}

func (service *Service) versionWarnings(checksum, normalization string) []string {
	warnings := make([]string, 0, 2)
	if checksum != service.fixtures.manifest.ExpectedSourceChecksum {
		warnings = append(warnings, "fixture_source_checksum_mismatch")
	}
	if normalization != service.fixtures.manifest.ExpectedNormalizationVersion {
		warnings = append(warnings, "fixture_normalization_version_mismatch")
	}
	return warnings
}

func validateAnalyzeRequest(request AnalyzeRequest) error {
	matching := request.Matching
	coverage := request.Coverage
	if matching.SampleStepMeters <= 0 || matching.SampleStepMeters > 25 ||
		matching.CandidateRadiusMeters <= 0 || matching.CandidateRadiusMeters > 50 ||
		matching.MaxDirectionDegrees <= 0 || matching.MaxDirectionDegrees > 90 ||
		matching.EndpointToleranceMeters <= 0 || matching.EndpointToleranceMeters > 10 {
		return fmt.Errorf("%w: matching parameters are outside supported ranges", ErrInvalidParameters)
	}
	if coverage.RadiusMeters < 5 || coverage.RadiusMeters > 50 || coverage.CoverageRatio <= 0 || coverage.CoverageRatio > 1 ||
		coverage.MinRequiredMeters < 0 || coverage.MaxRequiredMeters <= 0 || coverage.MinRequiredMeters > coverage.MaxRequiredMeters {
		return fmt.Errorf("%w: coverage parameters are outside supported ranges", ErrInvalidParameters)
	}
	return nil
}

type sampleMatch struct {
	segment *CandidateSegment
	nearest nearestPoint
	score   Score
	reason  string
}

func matchRoute(points []domain.Point, candidates []CandidateSegment, parameters MatchingParameters) (
	[]MatchedFragment, []UnmatchedFragment, [][]domain.Point, map[string][][2]float64, float64,
) {
	samples := densify(points, parameters.SampleStepMeters)
	matches := make([]sampleMatch, len(samples))
	var previous *CandidateSegment
	for index, sample := range samples {
		best, second := sampleMatch{}, sampleMatch{}
		best.score.Confidence, second.score.Confidence = -1, -1
		hadDistanceCandidate, hadDirectionCandidate := false, false
		for candidateIndex := range candidates {
			candidate := &candidates[candidateIndex]
			nearest := nearestOnLine(sample.point, candidate.Geometry)
			if nearest.distance > parameters.CandidateRadiusMeters {
				continue
			}
			hadDistanceCandidate = true
			directionDifference := undirectedAngleDifference(sample.heading, nearest.heading)
			if directionDifference > parameters.MaxDirectionDegrees {
				continue
			}
			hadDirectionCandidate = true
			distanceScore := 1 - nearest.distance/parameters.CandidateRadiusMeters
			directionScore := 1 - directionDifference/parameters.MaxDirectionDegrees
			continuityScore := continuity(previous, candidate, parameters)
			confidence := .55*distanceScore + .25*directionScore + .20*continuityScore
			if candidate.Classification == domain.StreetSegmentExplore {
				confidence += .005
			}
			choice := sampleMatch{segment: candidate, nearest: nearest, score: Score{
				DistanceScore: distanceScore, DirectionScore: directionScore,
				ContinuityScore: continuityScore, Confidence: math.Min(1, confidence), Reason: "MATCHED_SEQUENTIAL",
			}}
			if choice.score.Confidence > best.score.Confidence {
				second, best = best, choice
			} else if choice.score.Confidence > second.score.Confidence {
				second = choice
			}
		}
		switch {
		case !hadDistanceCandidate:
			matches[index].reason = "UNMATCHED_NO_CANDIDATE"
		case !hadDirectionCandidate:
			matches[index].reason = "UNMATCHED_DIRECTION"
		case best.segment == nil:
			matches[index].reason = "UNMATCHED_NO_CANDIDATE"
		case second.segment != nil && best.segment.ID != second.segment.ID &&
			best.score.Confidence-second.score.Confidence < .003 && continuity(previous, best.segment, parameters) == 0:
			matches[index].reason = "UNMATCHED_AMBIGUOUS"
		case previous != nil && best.score.ContinuityScore == 0 && best.score.Confidence < .62:
			matches[index].reason = "UNMATCHED_DISCONTINUITY"
		case best.score.Confidence < .32:
			matches[index].reason = "UNMATCHED_DIRECTION"
		default:
			matches[index] = best
			previous = best.segment
		}
	}
	return assembleMatchResult(samples, matches, parameters.SampleStepMeters)
}

func continuity(previous, candidate *CandidateSegment, parameters MatchingParameters) float64 {
	if previous == nil {
		return .65
	}
	if previous.ID == candidate.ID {
		return 1
	}
	minimum := math.Inf(1)
	for _, a := range []domain.Point{previous.Geometry[0], previous.Geometry[len(previous.Geometry)-1]} {
		for _, b := range []domain.Point{candidate.Geometry[0], candidate.Geometry[len(candidate.Geometry)-1]} {
			minimum = math.Min(minimum, distanceMeters(a, b))
		}
	}
	if minimum <= parameters.EndpointToleranceMeters {
		return .85
	}
	if minimum <= parameters.CandidateRadiusMeters {
		return .35
	}
	return 0
}

func assembleMatchResult(samples []routeSample, matches []sampleMatch, sampleStep float64) (
	[]MatchedFragment, []UnmatchedFragment, [][]domain.Point, map[string][][2]float64, float64,
) {
	matchedFragments := make([]MatchedFragment, 0)
	unmatchedFragments := make([]UnmatchedFragment, 0)
	normalized := make([][]domain.Point, 0)
	direct := make(map[string][][2]float64)
	matchedMeters := 0.0
	for index := 1; index < len(samples); index++ {
		if matches[index-1].segment != nil && matches[index].segment != nil {
			matchedMeters += samples[index].measure - samples[index-1].measure
		}
	}
	for start := 0; start < len(samples); {
		if matches[start].segment == nil {
			end := start + 1
			for end < len(samples) && matches[end].segment == nil && matches[end].reason == matches[start].reason {
				end++
			}
			line := samplePoints(samples[start:end])
			if len(line) == 1 {
				line = append(line, line[0])
			}
			unmatchedFragments = append(unmatchedFragments, UnmatchedFragment{
				Reason: matches[start].reason, Geometry: lineGeometryJSON(line),
				StartMeters: samples[start].measure, EndMeters: samples[end-1].measure,
			})
			start = end
			continue
		}
		end := start + 1
		for end < len(samples) && matches[end].segment != nil && matches[end].segment.ID == matches[start].segment.ID {
			end++
		}
		projected := make([]domain.Point, 0, end-start)
		distanceScore, directionScore, continuityScore, confidence := 0.0, 0.0, 0.0, 0.0
		minimumMeasure, maximumMeasure := math.Inf(1), 0.0
		for index := start; index < end; index++ {
			projected = append(projected, matches[index].nearest.point)
			minimumMeasure = math.Min(minimumMeasure, matches[index].nearest.measure)
			maximumMeasure = math.Max(maximumMeasure, matches[index].nearest.measure)
			distanceScore += matches[index].score.DistanceScore
			directionScore += matches[index].score.DirectionScore
			continuityScore += matches[index].score.ContinuityScore
			confidence += matches[index].score.Confidence
		}
		if len(projected) == 1 {
			projected = append(projected, projected[0])
		}
		count := float64(end - start)
		matchedFragments = append(matchedFragments, MatchedFragment{
			SegmentID: matches[start].segment.ID, Classification: matches[start].segment.Classification,
			ReasonCode: matches[start].segment.ReasonCode,
			Geometry:   lineGeometryJSON(projected), RouteStartMeters: samples[start].measure, RouteEndMeters: samples[end-1].measure,
			Score: Score{DistanceScore: distanceScore / count, DirectionScore: directionScore / count,
				ContinuityScore: continuityScore / count, Confidence: confidence / count, Reason: "MATCHED_SEQUENTIAL"},
		})
		padding := sampleStep / 2
		direct[matches[start].segment.ID] = append(direct[matches[start].segment.ID], [2]float64{
			math.Max(0, minimumMeasure-padding), math.Min(matches[start].segment.LengthMeters, maximumMeasure+padding),
		})
		start = end
	}
	for start := 0; start < len(samples); {
		for start < len(samples) && matches[start].segment == nil {
			start++
		}
		if start == len(samples) {
			break
		}
		end := start + 1
		for end < len(samples) && matches[end].segment != nil {
			end++
		}
		line := make([]domain.Point, 0, end-start)
		for index := start; index < end; index++ {
			line = append(line, matches[index].nearest.point)
		}
		if len(line) >= 2 {
			normalized = append(normalized, line)
		}
		start = end
	}
	return matchedFragments, unmatchedFragments, normalized, direct, matchedMeters
}

func samplePoints(samples []routeSample) []domain.Point {
	result := make([]domain.Point, len(samples))
	for index, sample := range samples {
		result[index] = sample.point
	}
	return result
}

func calculateCoverage(candidates []CandidateSegment, directIntervals map[string][][2]float64, profile CoverageProfile) ([]CoverageSegment, Metrics) {
	result := make([]CoverageSegment, 0, len(candidates))
	metrics := Metrics{}
	directGrades := make(map[string]bool)
	for _, candidate := range candidates {
		if len(directIntervals[candidate.ID]) > 0 {
			directGrades[gradeSignature(candidate.Attributes)] = true
		}
	}
	for _, candidate := range candidates {
		directMeters := mergedIntervalLength(directIntervals[candidate.ID])
		coveredMeters := candidate.RadiusCoveredMeters
		if directMeters == 0 && len(directGrades) > 0 && !directGrades[gradeSignature(candidate.Attributes)] {
			coveredMeters = 0
		}
		coveredMeters = math.Min(candidate.LengthMeters, math.Max(directMeters, coveredMeters))
		if candidate.Classification == domain.StreetSegmentRoutableOnly {
			if directMeters > 0 {
				result = append(result, CoverageSegment{
					SegmentID: candidate.ID, Classification: candidate.Classification,
					Geometry: candidate.GeometryJSON, LengthMeters: candidate.LengthMeters,
					DirectMeters: directMeters, Status: "CONNECTOR", Provenance: "DIRECT",
				})
			}
			continue
		}
		metrics.ContextExplorableLengthMeters += candidate.LengthMeters
		required := math.Min(candidate.LengthMeters, math.Max(profile.MinRequiredMeters, math.Min(profile.MaxRequiredMeters, candidate.LengthMeters*profile.CoverageRatio)))
		status := "NOT_COVERED"
		if coveredMeters >= required {
			status = "COMPLETED"
			metrics.CompletedNetworkLengthMeters += candidate.LengthMeters
		} else if coveredMeters > 0 {
			status = "PARTIAL"
		}
		provenance := ""
		switch {
		case directMeters > 0 && coveredMeters > directMeters+.5:
			provenance = "DIRECT_AND_RADIUS"
		case directMeters > 0:
			provenance = "DIRECT"
		case coveredMeters > 0:
			provenance = "RADIUS"
		}
		metrics.GeometricCoveredLengthMeters += coveredMeters
		result = append(result, CoverageSegment{
			SegmentID: candidate.ID, Classification: candidate.Classification,
			Geometry: candidate.GeometryJSON, LengthMeters: candidate.LengthMeters,
			CoveredMeters: coveredMeters, DirectMeters: directMeters, RequiredMeters: required,
			Status: status, Provenance: provenance,
		})
	}
	if metrics.ContextExplorableLengthMeters > 0 {
		metrics.CompletedNetworkRatio = metrics.CompletedNetworkLengthMeters / metrics.ContextExplorableLengthMeters
	}
	sort.Slice(result, func(i, j int) bool { return result[i].SegmentID < result[j].SegmentID })
	return result, metrics
}

func mergedIntervalLength(intervals [][2]float64) float64 {
	if len(intervals) == 0 {
		return 0
	}
	sort.Slice(intervals, func(i, j int) bool { return intervals[i][0] < intervals[j][0] })
	start, end, total := intervals[0][0], intervals[0][1], 0.0
	for _, interval := range intervals[1:] {
		if interval[0] <= end {
			end = math.Max(end, interval[1])
			continue
		}
		total += end - start
		start, end = interval[0], interval[1]
	}
	return total + end - start
}

func gradeSignature(attributes domain.StreetSegmentAttributes) string {
	tags := attributes.SourceTags
	parts := []string{"surface"}
	for _, key := range []string{"bridge", "tunnel", "indoor", "level"} {
		if value := strings.TrimSpace(tags[key]); value != "" && value != "no" && value != "0" {
			parts = append(parts, key+"="+value)
		}
	}
	return strings.Join(parts, ";")
}
