package exploration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"regexp"
	"time"

	"github.com/doveva/Gulyaem/backend/internal/geo/querying"
	"github.com/doveva/Gulyaem/backend/internal/walks"
)

var (
	ErrRouteGeoVersionStale = errors.New("walk route geo version stale")
	ErrRebuildRequired      = errors.New("exploration rebuild required")
	ErrInvalidBBox          = errors.New("invalid exploration bbox")
)

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-8][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`)

type State struct {
	Status    string     `json:"status"`
	UpdatedAt *time.Time `json:"updatedAt,omitempty"`
}
type Metric struct {
	ExploredLengthMeters  float64 `json:"exploredLengthMeters"`
	EligibleLengthMeters  float64 `json:"eligibleLengthMeters"`
	Percentage            float64 `json:"percentage"`
	ExploredSegmentsCount int     `json:"exploredSegmentsCount"`
}
type DistrictMetric struct {
	DistrictID           string  `json:"districtId"`
	Name                 string  `json:"name"`
	ExploredLengthMeters float64 `json:"exploredLengthMeters"`
	EligibleLengthMeters float64 `json:"eligibleLengthMeters"`
	Percentage           float64 `json:"percentage"`
}
type CityResult struct {
	GeoDataVersion struct {
		ID string `json:"id"`
	} `json:"geoDataVersion"`
	State     State            `json:"state"`
	City      Metric           `json:"city"`
	Districts []DistrictMetric `json:"districts"`
}

type Repository interface {
	Complete(context.Context, string, string) (walks.CompletionResult, error)
	City(context.Context, string, string) (CityResult, error)
	Segments(context.Context, string, string, [4]float64, int) (json.RawMessage, error)
}

type Service struct {
	repository Repository
	logger     *slog.Logger
}

func NewService(repository Repository, loggers ...*slog.Logger) *Service {
	logger := slog.Default()
	if len(loggers) > 0 && loggers[0] != nil {
		logger = loggers[0]
	}
	return &Service{repository: repository, logger: logger}
}

func (s *Service) Complete(ctx context.Context, actorID, walkID string) (walks.CompletionResult, error) {
	if !uuidPattern.MatchString(actorID) || !uuidPattern.MatchString(walkID) {
		return walks.CompletionResult{}, fmt.Errorf("%w: ids must be UUIDs", walks.ErrInvalidRequest)
	}
	started := time.Now()
	result, err := s.repository.Complete(ctx, actorID, walkID)
	if err == nil {
		s.logger.InfoContext(ctx, "walk exploration completed", "walk_id", walkID, "new_segments", result.Exploration.NewSegmentsCount, "revisited_segments", result.Exploration.RevisitedSegmentsCount, "duration_ms", time.Since(started).Milliseconds())
	}
	return result, err
}
func (s *Service) City(ctx context.Context, actorID, cityID string) (CityResult, error) {
	if !uuidPattern.MatchString(actorID) || !uuidPattern.MatchString(cityID) {
		return CityResult{}, fmt.Errorf("%w: ids must be UUIDs", walks.ErrInvalidRequest)
	}
	started := time.Now()
	result, err := s.repository.City(ctx, actorID, cityID)
	if err == nil {
		s.logger.InfoContext(ctx, "city exploration read", "city_id", cityID, "duration_ms", time.Since(started).Milliseconds())
	}
	return result, err
}
func (s *Service) Segments(ctx context.Context, actorID, cityID string, bbox [4]float64) (json.RawMessage, error) {
	if !uuidPattern.MatchString(actorID) || !uuidPattern.MatchString(cityID) {
		return nil, fmt.Errorf("%w: ids must be UUIDs", walks.ErrInvalidRequest)
	}
	if !validBBox(bbox) {
		return nil, ErrInvalidBBox
	}
	started := time.Now()
	result, err := s.repository.Segments(ctx, actorID, cityID, bbox, querying.MaximumFeatures)
	if err == nil {
		s.logger.InfoContext(ctx, "explored segments read", "city_id", cityID, "duration_ms", time.Since(started).Milliseconds())
	}
	return result, err
}
func validBBox(b [4]float64) bool {
	for _, v := range b {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return false
		}
	}
	bbox := querying.BBox{West: b[0], South: b[1], East: b[2], North: b[3]}
	return bbox.Valid() && bbox.AreaSquareKilometers() <= querying.MaximumBBoxAreaSquareKilometers
}
