package routingspike

import (
	"math"
	"sort"
)

const earthRadiusMeters = 6371008.8

func geometryLength(line LineString) float64 {
	length := 0.0
	for index := 1; index < len(line.Coordinates); index++ {
		length += pointDistance(line.Coordinates[index-1], line.Coordinates[index])
	}
	return length
}

func corridorMetrics(candidate, reference LineString, corridor, sampleStep float64) CorridorMetrics {
	return CorridorMetrics{
		CandidateInsideReferenceRatio: withinRatio(candidate.Coordinates, reference.Coordinates, corridor, sampleStep),
		ReferenceInsideCandidateRatio: withinRatio(reference.Coordinates, candidate.Coordinates, corridor, sampleStep),
	}
}

func withinRatio(source, target []Point, corridor, sampleStep float64) float64 {
	samples := densifyLine(source, sampleStep)
	if len(samples) == 0 || len(target) < 2 {
		return 0
	}
	inside := 0
	for _, sample := range samples {
		if distanceToLine(sample, target) <= corridor {
			inside++
		}
	}
	return float64(inside) / float64(len(samples))
}

func densifyLine(points []Point, step float64) []Point {
	if len(points) < 2 || step <= 0 {
		return nil
	}
	result := []Point{points[0]}
	for index := 1; index < len(points); index++ {
		start, end := points[index-1], points[index]
		length := pointDistance(start, end)
		count := int(math.Ceil(length / step))
		for part := 1; part <= count; part++ {
			ratio := math.Min(1, float64(part)*step/length)
			if length == 0 {
				ratio = 1
			}
			result = append(result, Point{
				start[0] + (end[0]-start[0])*ratio,
				start[1] + (end[1]-start[1])*ratio,
			})
		}
	}
	return result
}

func distanceToLine(point Point, line []Point) float64 {
	minimum := math.Inf(1)
	for index := 1; index < len(line); index++ {
		minimum = math.Min(minimum, distanceToSegment(point, line[index-1], line[index]))
	}
	return minimum
}

func distanceToSegment(point, start, end Point) float64 {
	latitude := (point[1] + start[1] + end[1]) / 3 * math.Pi / 180
	project := func(value Point) (float64, float64) {
		return value[0] * math.Pi / 180 * earthRadiusMeters * math.Cos(latitude),
			value[1] * math.Pi / 180 * earthRadiusMeters
	}
	px, py := project(point)
	ax, ay := project(start)
	bx, by := project(end)
	dx, dy := bx-ax, by-ay
	denominator := dx*dx + dy*dy
	if denominator == 0 {
		return math.Hypot(px-ax, py-ay)
	}
	ratio := ((px-ax)*dx + (py-ay)*dy) / denominator
	ratio = math.Max(0, math.Min(1, ratio))
	return math.Hypot(px-(ax+ratio*dx), py-(ay+ratio*dy))
}

func pointDistance(a, b Point) float64 {
	lat1, lat2 := a[1]*math.Pi/180, b[1]*math.Pi/180
	dLat := lat2 - lat1
	dLon := (b[0] - a[0]) * math.Pi / 180
	value := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(lat1)*math.Cos(lat2)*math.Sin(dLon/2)*math.Sin(dLon/2)
	return 2 * earthRadiusMeters * math.Atan2(math.Sqrt(value), math.Sqrt(1-value))
}

func percentile(values []float64, quantile float64) float64 {
	if len(values) == 0 {
		return 0
	}
	ordered := append([]float64(nil), values...)
	sort.Float64s(ordered)
	index := int(math.Ceil(quantile*float64(len(ordered)))) - 1
	index = max(0, min(len(ordered)-1, index))
	return ordered[index]
}

func buildSummaries(engines []Engine, cases []CaseResult, matches []MapMatchResult) []EngineSummary {
	result := make([]EngineSummary, 0, len(engines))
	for _, engine := range engines {
		summary := EngineSummary{EngineID: engine.ID, TotalRoutes: len(cases), MapMatchingStatus: "not_run"}
		candidateRatios, referenceRatios, matcherRatios, latencies := []float64{}, []float64{}, []float64{}, []float64{}
		for _, routeCase := range cases {
			for _, routeResult := range routeCase.Results {
				if routeResult.EngineID != engine.ID || routeResult.Status != "ok" {
					continue
				}
				summary.SuccessfulRoutes++
				candidateRatios = append(candidateRatios, routeResult.Corridor.CandidateInsideReferenceRatio)
				referenceRatios = append(referenceRatios, routeResult.Corridor.ReferenceInsideCandidateRatio)
				latencies = append(latencies, routeResult.Latency.P50Milliseconds)
				if routeResult.Matcher != nil {
					matcherRatios = append(matcherRatios, routeResult.Matcher.RouteMatchedRatio)
				}
			}
		}
		summary.MeanCandidateCorridorRatio = mean(candidateRatios)
		summary.MeanReferenceCorridorRatio = mean(referenceRatios)
		summary.MeanStreetSegmentMatchRatio = mean(matcherRatios)
		summary.MedianWarmLatencyMs = percentile(latencies, .5)
		statuses := map[string]bool{}
		for _, match := range matches {
			if match.EngineID == engine.ID {
				statuses[match.Status] = true
			}
		}
		switch {
		case statuses["ok"] && len(statuses) == 1:
			summary.MapMatchingStatus = "ok"
		case statuses["ok"]:
			summary.MapMatchingStatus = "partial"
		case len(statuses) > 0:
			summary.MapMatchingStatus = "unavailable"
		}
		result = append(result, summary)
	}
	return result
}

func mean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	total := 0.0
	for _, value := range values {
		total += value
	}
	return total / float64(len(values))
}
