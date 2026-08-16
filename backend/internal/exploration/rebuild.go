package exploration

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/doveva/Gulyaem/backend/internal/geo/domain"
	"github.com/doveva/Gulyaem/backend/internal/geo/querying"
	"github.com/doveva/Gulyaem/backend/internal/geo/routeanalysis"
)

type RebuildWalk struct {
	ID         string
	FinishedAt time.Time
	Geometry   json.RawMessage
}
type RebuiltProgress struct {
	SegmentID                       string
	FirstExploredAt, LastExploredAt time.Time
	VisitCount                      int
	FirstWalkID, LastWalkID         string
}
type RebuildResult struct {
	GeoDataVersionID  string        `json:"geoDataVersionId"`
	WalksProcessed    int           `json:"walksProcessed"`
	SegmentsPublished int           `json:"segmentsPublished"`
	Duration          time.Duration `json:"duration"`
}

type RebuildAnalyzer interface {
	CurrentVersion(context.Context, string) (querying.Version, error)
	AnalyzeGeometryForVersion(context.Context, querying.Version, string, json.RawMessage, routeanalysis.AnalyzeRequest) (routeanalysis.Analysis, error)
}
type RebuildRepository interface {
	CompletedWalks(context.Context, string, string) ([]RebuildWalk, error)
	PublishRebuild(context.Context, string, string, string, []RebuiltProgress) error
}
type Rebuilder struct {
	analyzer   RebuildAnalyzer
	repository RebuildRepository
}

func NewRebuilder(analyzer RebuildAnalyzer, repository RebuildRepository) *Rebuilder {
	return &Rebuilder{analyzer: analyzer, repository: repository}
}

func (r *Rebuilder) Rebuild(ctx context.Context, actorID, cityID string) (RebuildResult, error) {
	started := time.Now()
	version, err := r.analyzer.CurrentVersion(ctx, cityID)
	if err != nil {
		return RebuildResult{}, err
	}
	walks, err := r.repository.CompletedWalks(ctx, actorID, cityID)
	if err != nil {
		return RebuildResult{}, err
	}
	progress := map[string]RebuiltProgress{}
	for _, walk := range walks {
		analysis, err := r.analyzer.AnalyzeGeometryForVersion(ctx, version, "rebuild-"+walk.ID, walk.Geometry, routeanalysis.AnalyzeRequest{Matching: routeanalysis.DefaultMatchingParameters(), Coverage: routeanalysis.CoverageProfiles["balanced"]})
		if err != nil {
			return RebuildResult{}, fmt.Errorf("analyze completed walk %s: %w", walk.ID, err)
		}
		seen := map[string]bool{}
		for _, segment := range analysis.CoverageSegments {
			if segment.Status != "COMPLETED" || segment.Classification != domain.StreetSegmentExplore || seen[segment.SegmentID] {
				continue
			}
			seen[segment.SegmentID] = true
			current, exists := progress[segment.SegmentID]
			if !exists {
				current = RebuiltProgress{SegmentID: segment.SegmentID, FirstExploredAt: walk.FinishedAt, FirstWalkID: walk.ID}
			}
			current.LastExploredAt = walk.FinishedAt
			current.LastWalkID = walk.ID
			current.VisitCount++
			progress[segment.SegmentID] = current
		}
	}
	items := make([]RebuiltProgress, 0, len(progress))
	for _, item := range progress {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].SegmentID < items[j].SegmentID })
	if err := r.repository.PublishRebuild(ctx, actorID, cityID, version.ID, items); err != nil {
		return RebuildResult{}, err
	}
	return RebuildResult{GeoDataVersionID: version.ID, WalksProcessed: len(walks), SegmentsPublished: len(items), Duration: time.Since(started)}, nil
}
