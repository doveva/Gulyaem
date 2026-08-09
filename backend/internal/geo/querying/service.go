package querying

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"sort"
	"time"

	"github.com/doveva/Gulyaem/backend/internal/geo/domain"
)

const (
	MaximumBBoxAreaSquareKilometers = 25.0
	MaximumFeatures                 = 10000
)

var (
	ErrNotFound      = errors.New("geo resource not found")
	ErrFeatureLimit  = errors.New("geo feature limit exceeded")
	ErrBBoxAreaLimit = errors.New("bbox area limit exceeded")
)

type BBox struct {
	West  float64 `json:"west"`
	South float64 `json:"south"`
	East  float64 `json:"east"`
	North float64 `json:"north"`
}

func (bbox BBox) Valid() bool {
	return bbox.West >= -180 && bbox.East <= 180 && bbox.South >= -90 && bbox.North <= 90 &&
		bbox.West < bbox.East && bbox.South < bbox.North
}

func (bbox BBox) AreaSquareKilometers() float64 {
	middleLatitude := (bbox.South + bbox.North) / 2
	width := haversineMeters(bbox.West, middleLatitude, bbox.East, middleLatitude)
	height := haversineMeters(bbox.West, bbox.South, bbox.West, bbox.North)
	return width * height / 1_000_000
}

type SegmentFilter struct {
	CityID          string
	BBox            BBox
	Classifications []domain.StreetSegmentClassification
	MinLength       *float64
	MaxLength       *float64
}

type Version struct {
	ID                   string
	CityID               string
	Source               string
	SourceTimestamp      *time.Time
	SourceChecksum       string
	NormalizationVersion string
	Status               domain.GeoDataVersionStatus
	ImportedAt           *time.Time
	ImportReport         domain.ImportReport
}

type Segment struct {
	ID                   string
	CityID               string
	GeoDataVersionID     string
	GeometryJSON         json.RawMessage
	LengthMeters         float64
	Classification       domain.StreetSegmentClassification
	Attributes           domain.StreetSegmentAttributes
	StreetID             *string
	StreetName           *string
	VersionStatus        domain.GeoDataVersionStatus
	NormalizationVersion string
	IsCurrent            bool
}

type Statistics struct {
	SegmentsTotal          int64   `json:"segmentsTotal"`
	ExploreCount           int64   `json:"exploreCount"`
	RoutableOnlyCount      int64   `json:"routableOnlyCount"`
	IgnoreCount            int64   `json:"ignoreCount"`
	TotalLengthMeters      float64 `json:"totalLengthMeters"`
	ExplorableLengthMeters float64 `json:"explorableLengthMeters"`
	MinLengthMeters        float64 `json:"minLengthMeters"`
	MedianLengthMeters     float64 `json:"medianLengthMeters"`
	P95LengthMeters        float64 `json:"p95LengthMeters"`
	MaxLengthMeters        float64 `json:"maxLengthMeters"`
	ShortSegmentCount      int64   `json:"shortSegmentCount"`
	LongSegmentCount       int64   `json:"longSegmentCount"`
}

type SegmentCollection struct {
	Version    Version
	Segments   []Segment
	Statistics Statistics
}

type Repository interface {
	CurrentVersion(context.Context, string) (Version, error)
	Segments(context.Context, SegmentFilter, int) ([]Segment, error)
	Segment(context.Context, string) (Segment, error)
}

type Service struct {
	repository Repository
}

func NewService(repository Repository) *Service {
	return &Service{repository: repository}
}

func (service *Service) CurrentVersion(ctx context.Context, cityID string) (Version, error) {
	return service.repository.CurrentVersion(ctx, cityID)
}

func (service *Service) Segments(ctx context.Context, filter SegmentFilter) (SegmentCollection, error) {
	if filter.BBox.AreaSquareKilometers() > MaximumBBoxAreaSquareKilometers {
		return SegmentCollection{}, ErrBBoxAreaLimit
	}
	version, err := service.repository.CurrentVersion(ctx, filter.CityID)
	if err != nil {
		return SegmentCollection{}, err
	}
	segments, err := service.repository.Segments(ctx, filter, MaximumFeatures+1)
	if err != nil {
		return SegmentCollection{}, err
	}
	if len(segments) > MaximumFeatures {
		return SegmentCollection{}, ErrFeatureLimit
	}
	return SegmentCollection{
		Version:    version,
		Segments:   segments,
		Statistics: calculateStatistics(segments),
	}, nil
}

func (service *Service) Segment(ctx context.Context, segmentID string) (Segment, error) {
	return service.repository.Segment(ctx, segmentID)
}

func calculateStatistics(segments []Segment) Statistics {
	statistics := Statistics{SegmentsTotal: int64(len(segments))}
	lengths := make([]float64, 0, len(segments))
	for _, segment := range segments {
		lengths = append(lengths, segment.LengthMeters)
		statistics.TotalLengthMeters += segment.LengthMeters
		if segment.LengthMeters < 5 {
			statistics.ShortSegmentCount++
		}
		if segment.LengthMeters > 500 {
			statistics.LongSegmentCount++
		}
		switch segment.Classification {
		case domain.StreetSegmentExplore:
			statistics.ExploreCount++
			statistics.ExplorableLengthMeters += segment.LengthMeters
		case domain.StreetSegmentRoutableOnly:
			statistics.RoutableOnlyCount++
		case domain.StreetSegmentIgnore:
			statistics.IgnoreCount++
		}
	}
	if len(lengths) == 0 {
		return statistics
	}
	sort.Float64s(lengths)
	statistics.MinLengthMeters = lengths[0]
	statistics.MaxLengthMeters = lengths[len(lengths)-1]
	middle := len(lengths) / 2
	if len(lengths)%2 == 0 {
		statistics.MedianLengthMeters = (lengths[middle-1] + lengths[middle]) / 2
	} else {
		statistics.MedianLengthMeters = lengths[middle]
	}
	p95Index := int(math.Ceil(float64(len(lengths))*0.95)) - 1
	statistics.P95LengthMeters = lengths[p95Index]
	return statistics
}

func haversineMeters(lon1, lat1, lon2, lat2 float64) float64 {
	const earthRadiusMeters = 6371008.8
	latitude1 := lat1 * math.Pi / 180
	latitude2 := lat2 * math.Pi / 180
	deltaLatitude := (lat2 - lat1) * math.Pi / 180
	deltaLongitude := (lon2 - lon1) * math.Pi / 180
	a := math.Sin(deltaLatitude/2)*math.Sin(deltaLatitude/2) +
		math.Cos(latitude1)*math.Cos(latitude2)*math.Sin(deltaLongitude/2)*math.Sin(deltaLongitude/2)
	return earthRadiusMeters * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}
