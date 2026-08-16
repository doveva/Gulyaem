package exploration

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/doveva/Gulyaem/backend/internal/geo/domain"
	"github.com/doveva/Gulyaem/backend/internal/geo/querying"
	"github.com/doveva/Gulyaem/backend/internal/geo/routeanalysis"
)

type rebuildAnalyzerFake struct {
	analyses map[string]routeanalysis.Analysis
}

func (f rebuildAnalyzerFake) CurrentVersion(context.Context, string) (querying.Version, error) {
	return querying.Version{ID: "version", CityID: "city"}, nil
}
func (f rebuildAnalyzerFake) AnalyzeGeometryForVersion(_ context.Context, _ querying.Version, id string, _ json.RawMessage, _ routeanalysis.AnalyzeRequest) (routeanalysis.Analysis, error) {
	return f.analyses[id], nil
}

type rebuildRepositoryFake struct {
	walks     []RebuildWalk
	published []RebuiltProgress
	version   string
}

func (f *rebuildRepositoryFake) CompletedWalks(context.Context, string, string) ([]RebuildWalk, error) {
	return f.walks, nil
}
func (f *rebuildRepositoryFake) PublishRebuild(_ context.Context, _, _, version string, p []RebuiltProgress) error {
	f.version = version
	f.published = p
	return nil
}
func TestRebuildDeduplicatesLoopsAndRestoresVisits(t *testing.T) {
	first := time.Date(2026, 8, 10, 10, 0, 0, 0, time.UTC)
	second := first.Add(24 * time.Hour)
	repo := &rebuildRepositoryFake{walks: []RebuildWalk{{ID: "walk-1", FinishedAt: first}, {ID: "walk-2", FinishedAt: second}}}
	segment := func(id string) routeanalysis.CoverageSegment {
		return routeanalysis.CoverageSegment{SegmentID: id, Classification: domain.StreetSegmentExplore, Status: "COMPLETED"}
	}
	analyzer := rebuildAnalyzerFake{analyses: map[string]routeanalysis.Analysis{"rebuild-walk-1": {CoverageSegments: []routeanalysis.CoverageSegment{segment("a"), segment("a"), {SegmentID: "partial", Classification: domain.StreetSegmentExplore, Status: "PARTIAL"}}}, "rebuild-walk-2": {CoverageSegments: []routeanalysis.CoverageSegment{segment("a"), segment("b")}}}}
	result, err := NewRebuilder(analyzer, repo).Rebuild(context.Background(), "actor", "city")
	if err != nil {
		t.Fatal(err)
	}
	if result.SegmentsPublished != 2 || len(repo.published) != 2 {
		t.Fatalf("result=%+v published=%+v", result, repo.published)
	}
	if repo.published[0].SegmentID != "a" || repo.published[0].VisitCount != 2 || !repo.published[0].FirstExploredAt.Equal(first) || !repo.published[0].LastExploredAt.Equal(second) {
		t.Fatalf("segment a=%+v", repo.published[0])
	}
}
