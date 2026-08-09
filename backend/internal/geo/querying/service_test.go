package querying

import (
	"context"
	"errors"
	"testing"

	"github.com/doveva/Gulyaem/backend/internal/geo/domain"
)

type repositoryStub struct {
	version  Version
	segments []Segment
	err      error
	limit    int
}

func (stub *repositoryStub) CurrentVersion(context.Context, string) (Version, error) {
	if stub.err != nil {
		return Version{}, stub.err
	}
	return stub.version, nil
}

func (stub *repositoryStub) Segments(_ context.Context, _ SegmentFilter, limit int) ([]Segment, error) {
	stub.limit = limit
	return stub.segments, stub.err
}

func (stub *repositoryStub) Segment(context.Context, string) (Segment, error) {
	if len(stub.segments) == 0 {
		return Segment{}, ErrNotFound
	}
	return stub.segments[0], nil
}

func TestSegmentsRejectsExcessiveBBoxBeforeRepository(t *testing.T) {
	repository := &repositoryStub{err: errors.New("must not be called")}
	service := NewService(repository)
	_, err := service.Segments(context.Background(), SegmentFilter{
		CityID: "city", BBox: BBox{West: 30, South: 59, East: 31, North: 60},
	})
	if !errors.Is(err, ErrBBoxAreaLimit) {
		t.Fatalf("error = %v", err)
	}
}

func TestSegmentsEnforcesFeatureLimit(t *testing.T) {
	segments := make([]Segment, MaximumFeatures+1)
	repository := &repositoryStub{version: Version{ID: "version"}, segments: segments}
	service := NewService(repository)
	_, err := service.Segments(context.Background(), SegmentFilter{
		CityID: "city", BBox: BBox{West: 30.3, South: 59.93, East: 30.31, North: 59.94},
	})
	if !errors.Is(err, ErrFeatureLimit) || repository.limit != MaximumFeatures+1 {
		t.Fatalf("error = %v limit = %d", err, repository.limit)
	}
}

func TestSegmentsCalculatesFilteredStatistics(t *testing.T) {
	repository := &repositoryStub{
		version: Version{ID: "version"},
		segments: []Segment{
			{LengthMeters: 4, Classification: domain.StreetSegmentExplore},
			{LengthMeters: 10, Classification: domain.StreetSegmentExplore},
			{LengthMeters: 20, Classification: domain.StreetSegmentRoutableOnly},
			{LengthMeters: 600, Classification: domain.StreetSegmentIgnore},
		},
	}
	collection, err := NewService(repository).Segments(context.Background(), SegmentFilter{
		CityID: "city", BBox: BBox{West: 30.3, South: 59.93, East: 30.31, North: 59.94},
	})
	if err != nil {
		t.Fatal(err)
	}
	statistics := collection.Statistics
	if statistics.SegmentsTotal != 4 || statistics.ExploreCount != 2 || statistics.RoutableOnlyCount != 1 || statistics.IgnoreCount != 1 {
		t.Fatalf("counts = %+v", statistics)
	}
	if statistics.TotalLengthMeters != 634 || statistics.ExplorableLengthMeters != 14 || statistics.MedianLengthMeters != 15 || statistics.P95LengthMeters != 600 {
		t.Fatalf("lengths = %+v", statistics)
	}
	if statistics.ShortSegmentCount != 1 || statistics.LongSegmentCount != 1 {
		t.Fatalf("diagnostics = %+v", statistics)
	}
}
