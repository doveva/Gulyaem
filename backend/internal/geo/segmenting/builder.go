package segmenting

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/doveva/Gulyaem/backend/internal/geo/domain"
)

const earthRadiusMeters = 6371008.8

type graphNode struct {
	key      string
	point    domain.Point
	sourceID *int64
	tags     map[string]string
	boundary bool
}

type graphEdge struct {
	a          string
	b          string
	profile    wayProfile
	sourceWays []int64
	clipped    bool
}

func Build(input Input) (Result, error) {
	if input.BBox != nil && !input.BBox.Valid() {
		return Result{}, errors.New("invalid import bbox")
	}
	if input.MaxSegmentLength < 0 {
		return Result{}, errors.New("max segment length cannot be negative")
	}

	sourceNodes := make(map[int64]Node, len(input.Nodes))
	for _, node := range input.Nodes {
		sourceNodes[node.SourceID] = node
	}

	graphNodes := make(map[string]graphNode)
	edges := make([]graphEdge, 0)
	atomicDuplicates := make(map[string]int)
	result := Result{}

	for _, way := range input.Ways {
		profile := normalizeWay(way.Tags)
		if profile.unsupportedArea {
			result.Report.UnsupportedPedestrianAreas++
		}
		if !profile.candidate {
			continue
		}
		result.Report.CandidateWays++
		if len(way.NodeIDs) < 2 {
			result.Report.SegmentsRejected++
			result.Report.InvalidGeometry++
			continue
		}

		for pairIndex := 0; pairIndex < len(way.NodeIDs)-1; pairIndex++ {
			fromID := way.NodeIDs[pairIndex]
			toID := way.NodeIDs[pairIndex+1]
			from, ok := sourceNodes[fromID]
			if !ok {
				return Result{}, fmt.Errorf("candidate way %d references missing node %d", way.SourceID, fromID)
			}
			to, ok := sourceNodes[toID]
			if !ok {
				return Result{}, fmt.Errorf("candidate way %d references missing node %d", way.SourceID, toID)
			}
			if !validPoint(from.Point) || !validPoint(to.Point) {
				result.Report.SegmentsRejected++
				result.Report.InvalidGeometry++
				continue
			}

			clippedFrom, clippedTo := false, false
			fromPoint, toPoint := from.Point, to.Point
			if input.BBox != nil {
				var visible bool
				fromPoint, toPoint, clippedFrom, clippedTo, visible = clipLine(fromPoint, toPoint, *input.BBox)
				if !visible {
					continue
				}
			}
			if pointsEqual(fromPoint, toPoint) {
				result.Report.SegmentsRejected++
				result.Report.ZeroLengthSegments++
				continue
			}

			fromKey := sourceNodeKey(fromID)
			fromSourceID := fromID
			if clippedFrom {
				fromKey = boundaryNodeKey(way.SourceID, pairIndex, 0, fromPoint)
				fromSourceID = 0
			}
			toKey := sourceNodeKey(toID)
			toSourceID := toID
			if clippedTo {
				toKey = boundaryNodeKey(way.SourceID, pairIndex, 1, toPoint)
				toSourceID = 0
			}

			addGraphNode(graphNodes, fromKey, fromPoint, fromSourceID, from.Tags, clippedFrom)
			addGraphNode(graphNodes, toKey, toPoint, toSourceID, to.Tags, clippedTo)
			edge := graphEdge{
				a:          fromKey,
				b:          toKey,
				profile:    profile,
				sourceWays: []int64{way.SourceID},
				clipped:    clippedFrom || clippedTo,
			}
			key := canonicalAtomicKey(graphNodes[fromKey].point, graphNodes[toKey].point, profile.semanticKey)
			if existingIndex, duplicate := atomicDuplicates[key]; duplicate {
				edges[existingIndex].sourceWays = appendUniqueInt64(edges[existingIndex].sourceWays, way.SourceID)
				edges[existingIndex].clipped = edges[existingIndex].clipped || edge.clipped
				result.Report.SegmentsDeduplicated++
				result.Report.DuplicateGeometry++
				continue
			}
			atomicDuplicates[key] = len(edges)
			edges = append(edges, edge)
		}
	}

	drafts := buildDrafts(graphNodes, edges)
	if input.MaxSegmentLength > 0 {
		drafts = splitDrafts(drafts, input.MaxSegmentLength)
	}
	drafts, duplicateReport := deduplicateDrafts(drafts)
	result.Report.SegmentsDeduplicated += duplicateReport.SegmentsDeduplicated
	result.Report.DuplicateGeometry += duplicateReport.DuplicateGeometry
	result.Report.ConflictingDuplicateGeometry += duplicateReport.ConflictingDuplicateGeometry
	result.Segments = drafts
	finalizeReport(&result.Report, drafts)
	return result, nil
}

func addGraphNode(nodes map[string]graphNode, key string, point domain.Point, sourceID int64, tags map[string]string, boundary bool) {
	if _, exists := nodes[key]; exists {
		return
	}
	var sourceIDPointer *int64
	if sourceID != 0 {
		value := sourceID
		sourceIDPointer = &value
	}
	nodes[key] = graphNode{key: key, point: point, sourceID: sourceIDPointer, tags: tags, boundary: boundary}
}

func buildDrafts(nodes map[string]graphNode, edges []graphEdge) []domain.StreetSegmentDraft {
	adjacency := make(map[string][]int)
	neighbors := make(map[string]map[string]struct{})
	semantics := make(map[string]map[string]struct{})
	for index, edge := range edges {
		adjacency[edge.a] = append(adjacency[edge.a], index)
		adjacency[edge.b] = append(adjacency[edge.b], index)
		if neighbors[edge.a] == nil {
			neighbors[edge.a] = make(map[string]struct{})
		}
		if neighbors[edge.b] == nil {
			neighbors[edge.b] = make(map[string]struct{})
		}
		neighbors[edge.a][edge.b] = struct{}{}
		neighbors[edge.b][edge.a] = struct{}{}
		if semantics[edge.a] == nil {
			semantics[edge.a] = make(map[string]struct{})
		}
		if semantics[edge.b] == nil {
			semantics[edge.b] = make(map[string]struct{})
		}
		semantics[edge.a][edge.profile.semanticKey] = struct{}{}
		semantics[edge.b][edge.profile.semanticKey] = struct{}{}
	}

	significant := make(map[string]bool, len(nodes))
	keys := make([]string, 0, len(nodes))
	for key, node := range nodes {
		keys = append(keys, key)
		significant[key] = node.boundary || significantNodeTags(node.tags) || len(neighbors[key]) != 2 || len(semantics[key]) != 1
	}
	sort.Strings(keys)

	used := make([]bool, len(edges))
	drafts := make([]domain.StreetSegmentDraft, 0)
	for _, key := range keys {
		if !significant[key] {
			continue
		}
		for _, edgeIndex := range adjacency[key] {
			if !used[edgeIndex] {
				drafts = append(drafts, walkDraft(key, edgeIndex, nodes, edges, adjacency, significant, used))
			}
		}
	}
	for edgeIndex, edge := range edges {
		if !used[edgeIndex] {
			drafts = append(drafts, walkDraft(edge.a, edgeIndex, nodes, edges, adjacency, significant, used))
		}
	}
	return drafts
}

func walkDraft(
	start string,
	edgeIndex int,
	nodes map[string]graphNode,
	edges []graphEdge,
	adjacency map[string][]int,
	significant map[string]bool,
	used []bool,
) domain.StreetSegmentDraft {
	profile := edges[edgeIndex].profile
	geometry := []domain.Point{nodes[start].point}
	sourceWays := make([]int64, 0)
	warnings := make([]string, 0)
	currentNode := start
	boundaryClip := nodes[start].boundary

	for {
		if used[edgeIndex] {
			break
		}
		edge := edges[edgeIndex]
		used[edgeIndex] = true
		sourceWays = appendUniqueInt64s(sourceWays, edge.sourceWays...)
		warnings = appendUniqueStrings(warnings, edge.profile.warnings...)
		boundaryClip = boundaryClip || edge.clipped
		nextNode := edge.a
		if nextNode == currentNode {
			nextNode = edge.b
		}
		geometry = append(geometry, nodes[nextNode].point)
		boundaryClip = boundaryClip || nodes[nextNode].boundary
		if nextNode == start || significant[nextNode] {
			currentNode = nextNode
			break
		}

		nextEdge := -1
		for _, candidate := range adjacency[nextNode] {
			if !used[candidate] && edges[candidate].profile.semanticKey == profile.semanticKey {
				nextEdge = candidate
				break
			}
		}
		if nextEdge == -1 {
			currentNode = nextNode
			break
		}
		currentNode = nextNode
		edgeIndex = nextEdge
	}

	sort.Slice(sourceWays, func(i, j int) bool { return sourceWays[i] < sourceWays[j] })
	attributes := domain.StreetSegmentAttributes{
		ReasonCode:        profile.reasonCode,
		SourceTags:        cloneTags(profile.relevantTags),
		SourceWayIDs:      sourceWays,
		SourceStartNodeID: cloneInt64Pointer(nodes[start].sourceID),
		SourceEndNodeID:   cloneInt64Pointer(nodes[currentNode].sourceID),
		BoundaryClip:      boundaryClip,
		Warnings:          warnings,
	}
	return domain.StreetSegmentDraft{
		Geometry:       geometry,
		LengthMeters:   lineLength(geometry),
		Classification: profile.classification,
		Attributes:     attributes,
	}
}

func significantNodeTags(tags map[string]string) bool {
	for _, key := range []string{"barrier", "entrance", "access", "foot"} {
		if strings.TrimSpace(tags[key]) != "" {
			return true
		}
	}
	return false
}

func splitDrafts(drafts []domain.StreetSegmentDraft, maximum float64) []domain.StreetSegmentDraft {
	result := make([]domain.StreetSegmentDraft, 0, len(drafts))
	for _, draft := range drafts {
		result = append(result, splitDraft(draft, maximum)...)
	}
	return result
}

func splitDraft(draft domain.StreetSegmentDraft, maximum float64) []domain.StreetSegmentDraft {
	if draft.LengthMeters <= maximum || len(draft.Geometry) < 2 {
		return []domain.StreetSegmentDraft{draft}
	}

	remainingGeometry := append([]domain.Point(nil), draft.Geometry...)
	parts := make([]domain.StreetSegmentDraft, 0, int(math.Ceil(draft.LengthMeters/maximum)))
	current := []domain.Point{remainingGeometry[0]}
	currentLength := 0.0
	for index := 1; index < len(remainingGeometry); {
		from := current[len(current)-1]
		to := remainingGeometry[index]
		edgeLength := pointDistance(from, to)
		if edgeLength == 0 {
			index++
			continue
		}
		available := maximum - currentLength
		if available <= 1e-7 {
			part := draft
			part.Geometry = append([]domain.Point(nil), current...)
			part.LengthMeters = lineLength(part.Geometry)
			part.Attributes.SourceEndNodeID = nil
			parts = append(parts, part)
			current = []domain.Point{from}
			currentLength = 0
			continue
		}
		if edgeLength <= available+1e-7 {
			current = append(current, to)
			currentLength += edgeLength
			index++
			continue
		}
		ratio := available / edgeLength
		splitPoint := domain.Point{Lon: from.Lon + (to.Lon-from.Lon)*ratio, Lat: from.Lat + (to.Lat-from.Lat)*ratio}
		current = append(current, splitPoint)
		part := draft
		part.Geometry = append([]domain.Point(nil), current...)
		part.LengthMeters = lineLength(part.Geometry)
		part.Attributes.SourceEndNodeID = nil
		parts = append(parts, part)
		current = []domain.Point{splitPoint}
		currentLength = 0
	}
	if len(current) > 1 {
		part := draft
		part.Geometry = append([]domain.Point(nil), current...)
		part.LengthMeters = lineLength(part.Geometry)
		part.Attributes.SourceStartNodeID = nil
		parts = append(parts, part)
	}
	for index := range parts {
		if index > 0 {
			parts[index].Attributes.SourceStartNodeID = nil
		}
		if index < len(parts)-1 {
			parts[index].Attributes.SourceEndNodeID = nil
		}
	}
	return parts
}

func deduplicateDrafts(drafts []domain.StreetSegmentDraft) ([]domain.StreetSegmentDraft, Report) {
	result := make([]domain.StreetSegmentDraft, 0, len(drafts))
	byExact := make(map[string]int)
	geometrySemantics := make(map[string]map[string]struct{})
	report := Report{}
	for _, draft := range drafts {
		geometryKey := canonicalGeometryKey(draft.Geometry)
		semanticKey := draftSemanticKey(draft)
		exactKey := geometryKey + "|" + semanticKey
		if existing, duplicate := byExact[exactKey]; duplicate {
			result[existing].Attributes.SourceWayIDs = appendUniqueInt64s(result[existing].Attributes.SourceWayIDs, draft.Attributes.SourceWayIDs...)
			sort.Slice(result[existing].Attributes.SourceWayIDs, func(i, j int) bool {
				return result[existing].Attributes.SourceWayIDs[i] < result[existing].Attributes.SourceWayIDs[j]
			})
			result[existing].Attributes.Warnings = appendUniqueStrings(result[existing].Attributes.Warnings, draft.Attributes.Warnings...)
			report.SegmentsDeduplicated++
			report.DuplicateGeometry++
			continue
		}
		if geometrySemantics[geometryKey] == nil {
			geometrySemantics[geometryKey] = make(map[string]struct{})
		}
		if len(geometrySemantics[geometryKey]) > 0 {
			report.DuplicateGeometry++
			if _, same := geometrySemantics[geometryKey][semanticKey]; !same {
				report.ConflictingDuplicateGeometry++
			}
		}
		geometrySemantics[geometryKey][semanticKey] = struct{}{}
		byExact[exactKey] = len(result)
		result = append(result, draft)
	}
	return result, report
}

func finalizeReport(report *Report, drafts []domain.StreetSegmentDraft) {
	report.SegmentsGenerated = int64(len(drafts))
	lengths := make([]float64, 0, len(drafts))
	for _, draft := range drafts {
		lengths = append(lengths, draft.LengthMeters)
		report.TotalLengthMeters += draft.LengthMeters
		if draft.Attributes.BoundaryClip {
			report.SegmentsClipped++
		}
		if draft.LengthMeters < 5 {
			report.ShortSegments++
		}
		if draft.LengthMeters > 500 {
			report.LongSegments++
		}
		switch draft.Classification {
		case domain.StreetSegmentExplore:
			report.ExploreSegments++
			report.ExplorableLengthMeters += draft.LengthMeters
		case domain.StreetSegmentRoutableOnly:
			report.RoutableOnlySegments++
		case domain.StreetSegmentIgnore:
			report.IgnoreSegments++
		}
	}
	if len(lengths) == 0 {
		return
	}
	sort.Float64s(lengths)
	report.MinLengthMeters = lengths[0]
	report.MaxLengthMeters = lengths[len(lengths)-1]
	middle := len(lengths) / 2
	if len(lengths)%2 == 0 {
		report.MedianLengthMeters = (lengths[middle-1] + lengths[middle]) / 2
	} else {
		report.MedianLengthMeters = lengths[middle]
	}
	p95Index := int(math.Ceil(float64(len(lengths))*0.95)) - 1
	report.P95LengthMeters = lengths[p95Index]
}

func clipLine(from, to domain.Point, bbox BBox) (domain.Point, domain.Point, bool, bool, bool) {
	dx := to.Lon - from.Lon
	dy := to.Lat - from.Lat
	p := []float64{-dx, dx, -dy, dy}
	q := []float64{from.Lon - bbox.West, bbox.East - from.Lon, from.Lat - bbox.South, bbox.North - from.Lat}
	t0, t1 := 0.0, 1.0
	for index := range p {
		if p[index] == 0 {
			if q[index] < 0 {
				return domain.Point{}, domain.Point{}, false, false, false
			}
			continue
		}
		ratio := q[index] / p[index]
		if p[index] < 0 {
			if ratio > t1 {
				return domain.Point{}, domain.Point{}, false, false, false
			}
			if ratio > t0 {
				t0 = ratio
			}
		} else {
			if ratio < t0 {
				return domain.Point{}, domain.Point{}, false, false, false
			}
			if ratio < t1 {
				t1 = ratio
			}
		}
	}
	clippedFrom := t0 > 1e-12
	clippedTo := t1 < 1-1e-12
	return domain.Point{Lon: from.Lon + t0*dx, Lat: from.Lat + t0*dy},
		domain.Point{Lon: from.Lon + t1*dx, Lat: from.Lat + t1*dy}, clippedFrom, clippedTo, true
}

func lineLength(points []domain.Point) float64 {
	length := 0.0
	for index := 1; index < len(points); index++ {
		length += pointDistance(points[index-1], points[index])
	}
	return length
}

func pointDistance(from, to domain.Point) float64 {
	lat1 := from.Lat * math.Pi / 180
	lat2 := to.Lat * math.Pi / 180
	deltaLat := (to.Lat - from.Lat) * math.Pi / 180
	deltaLon := (to.Lon - from.Lon) * math.Pi / 180
	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) + math.Cos(lat1)*math.Cos(lat2)*math.Sin(deltaLon/2)*math.Sin(deltaLon/2)
	return earthRadiusMeters * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}

func validPoint(point domain.Point) bool {
	return !math.IsNaN(point.Lon) && !math.IsNaN(point.Lat) && !math.IsInf(point.Lon, 0) && !math.IsInf(point.Lat, 0) &&
		point.Lon >= -180 && point.Lon <= 180 && point.Lat >= -90 && point.Lat <= 90
}

func pointsEqual(first, second domain.Point) bool {
	return math.Abs(first.Lon-second.Lon) < 1e-12 && math.Abs(first.Lat-second.Lat) < 1e-12
}

func sourceNodeKey(sourceID int64) string {
	return "n:" + strconv.FormatInt(sourceID, 10)
}

func boundaryNodeKey(wayID int64, pairIndex, side int, point domain.Point) string {
	return fmt.Sprintf("b:%d:%d:%d:%.12f:%.12f", wayID, pairIndex, side, point.Lon, point.Lat)
}

func canonicalAtomicKey(first, second domain.Point, semantic string) string {
	forward := pointKey(first) + ";" + pointKey(second)
	reverse := pointKey(second) + ";" + pointKey(first)
	if reverse < forward {
		forward = reverse
	}
	return forward + "|" + semantic
}

func canonicalGeometryKey(points []domain.Point) string {
	forwardParts := make([]string, len(points))
	reverseParts := make([]string, len(points))
	for index, point := range points {
		forwardParts[index] = pointKey(point)
		reverseParts[len(points)-1-index] = pointKey(point)
	}
	forward := strings.Join(forwardParts, ";")
	reverse := strings.Join(reverseParts, ";")
	if reverse < forward {
		return reverse
	}
	return forward
}

func pointKey(point domain.Point) string {
	return fmt.Sprintf("%.12f,%.12f", point.Lon, point.Lat)
}

func draftSemanticKey(draft domain.StreetSegmentDraft) string {
	profile := wayProfile{
		classification: draft.Classification,
		reasonCode:     draft.Attributes.ReasonCode,
		relevantTags:   draft.Attributes.SourceTags,
	}
	return buildSemanticKey(profile)
}

func appendUniqueInt64(values []int64, value int64) []int64 {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func appendUniqueInt64s(values []int64, additions ...int64) []int64 {
	for _, addition := range additions {
		values = appendUniqueInt64(values, addition)
	}
	return values
}

func appendUniqueStrings(values []string, additions ...string) []string {
	for _, addition := range additions {
		found := false
		for _, existing := range values {
			if existing == addition {
				found = true
				break
			}
		}
		if !found {
			values = append(values, addition)
		}
	}
	return values
}

func cloneTags(tags map[string]string) map[string]string {
	clone := make(map[string]string, len(tags))
	for key, value := range tags {
		clone[key] = value
	}
	return clone
}

func cloneInt64Pointer(value *int64) *int64 {
	if value == nil {
		return nil
	}
	clone := *value
	return &clone
}
