package walks

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"regexp"
	"time"

	"github.com/doveva/Gulyaem/backend/internal/routing/port"
	"github.com/doveva/Gulyaem/backend/internal/routing/preview"
)

var (
	ErrInvalidRequest      = errors.New("invalid walk request")
	ErrNotFound            = errors.New("walk not found")
	ErrPreviewStale        = errors.New("route preview stale")
	ErrInvalidState        = errors.New("walk invalid state")
	ErrRouteNotEditable    = errors.New("walk route not editable")
	ErrIdempotencyConflict = errors.New("client request id reused with different input")
	ErrConcurrentChange    = errors.New("walk changed concurrently")
	// ErrMaterializationGeoVersionStale is returned by persistent stores when
	// the preview-pinned GeoDataVersion is no longer READY at write time.
	ErrMaterializationGeoVersionStale = errors.New("materialization geo version stale")
	// ErrSegmentGeoVersionMismatch is returned when a materialized match points
	// at a StreetSegment outside the Route's pinned GeoDataVersion.
	ErrSegmentGeoVersionMismatch = errors.New("route segment geo version mismatch")
)

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-8][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`)

type Previewer interface {
	Create(context.Context, preview.Request) (preview.Result, error)
}

type Store interface {
	FindByClientRequest(context.Context, string, string) (Aggregate, error)
	Create(context.Context, Materialization, Walk) (Aggregate, error)
	Get(context.Context, string, string) (Aggregate, error)
	Transition(context.Context, string, string, Status, Status, time.Time) (Aggregate, error)
	ReplaceRoute(context.Context, string, string, []Status, Materialization) (Aggregate, error)
}

type Service struct {
	previewer Previewer
	store     Store
	now       func() time.Time
}

func NewService(previewer Previewer, store Store) *Service {
	return &Service{previewer: previewer, store: store, now: func() time.Time { return time.Now().UTC() }}
}

func (s *Service) Create(ctx context.Context, actorID string, request CreateRequest) (Aggregate, bool, error) {
	if err := validateActor(actorID); err != nil {
		return Aggregate{}, false, err
	}
	if err := validateCreate(request); err != nil {
		return Aggregate{}, false, err
	}
	requestFingerprint, err := hashRequest(request)
	if err != nil {
		return Aggregate{}, false, err
	}
	if existing, err := s.store.FindByClientRequest(ctx, actorID, request.ClientRequestID); err == nil {
		if existing.Walk.RequestFingerprint != requestFingerprint {
			return Aggregate{}, false, ErrIdempotencyConflict
		}
		return existing, false, nil
	} else if !errors.Is(err, ErrNotFound) {
		return Aggregate{}, false, err
	}

	result, err := s.previewer.Create(ctx, preview.Request{CityID: request.CityID, Profile: request.Profile, Waypoints: request.Waypoints})
	if err != nil {
		return Aggregate{}, false, err
	}
	if result.PreviewFingerprint != request.ExpectedPreviewFingerprint {
		return Aggregate{}, false, ErrPreviewStale
	}
	materialization, err := materialize(actorID, "", result, request.Waypoints)
	if err != nil {
		return Aggregate{}, false, err
	}
	walkID, err := newUUID()
	if err != nil {
		return Aggregate{}, false, err
	}
	walk := Walk{ID: walkID, ActorID: actorID, CityID: request.CityID, RouteID: materialization.Route.ID,
		ClientRequestID: request.ClientRequestID, RequestFingerprint: requestFingerprint, Status: StatusDraft}
	created, err := s.store.Create(ctx, materialization, walk)
	if isStaleMaterialization(err) {
		return Aggregate{}, false, ErrPreviewStale
	}
	if errors.Is(err, ErrIdempotencyConflict) {
		existing, loadErr := s.store.FindByClientRequest(ctx, actorID, request.ClientRequestID)
		if loadErr != nil {
			return Aggregate{}, false, err
		}
		if existing.Walk.RequestFingerprint != requestFingerprint {
			return Aggregate{}, false, ErrIdempotencyConflict
		}
		return existing, false, nil
	}
	return created, true, err
}

func (s *Service) Get(ctx context.Context, actorID, walkID string) (Aggregate, error) {
	if err := validateIDs(actorID, walkID); err != nil {
		return Aggregate{}, err
	}
	return s.store.Get(ctx, actorID, walkID)
}

func (s *Service) Start(ctx context.Context, actorID, walkID string) (Aggregate, error) {
	return s.transition(ctx, actorID, walkID, StatusDraft, StatusActive, StatusActive)
}

func (s *Service) Finish(ctx context.Context, actorID, walkID string) (Aggregate, error) {
	return s.transition(ctx, actorID, walkID, StatusActive, StatusReview, StatusReview)
}

func (s *Service) Cancel(ctx context.Context, actorID, walkID string) (Aggregate, error) {
	if err := validateIDs(actorID, walkID); err != nil {
		return Aggregate{}, err
	}
	aggregate, err := s.store.Get(ctx, actorID, walkID)
	if err != nil {
		return Aggregate{}, err
	}
	if aggregate.Walk.Status == StatusCancelled {
		return aggregate, nil
	}
	if aggregate.Walk.Status == StatusCompleted {
		return Aggregate{}, ErrInvalidState
	}
	switch aggregate.Walk.Status {
	case StatusDraft, StatusActive, StatusReview:
		return s.store.Transition(ctx, actorID, walkID, aggregate.Walk.Status, StatusCancelled, s.now())
	default:
		return Aggregate{}, ErrInvalidState
	}
}

func (s *Service) CorrectRoute(ctx context.Context, actorID, walkID string, request CorrectRouteRequest) (Aggregate, error) {
	if err := validateIDs(actorID, walkID); err != nil {
		return Aggregate{}, err
	}
	if err := validateRouteInput(request.Profile, request.ExpectedPreviewFingerprint, request.Waypoints); err != nil {
		return Aggregate{}, err
	}
	aggregate, err := s.store.Get(ctx, actorID, walkID)
	if err != nil {
		return Aggregate{}, err
	}
	if aggregate.Walk.Status != StatusDraft && aggregate.Walk.Status != StatusReview {
		return Aggregate{}, ErrRouteNotEditable
	}
	result, err := s.previewer.Create(ctx, preview.Request{CityID: aggregate.Walk.CityID, Profile: request.Profile, Waypoints: request.Waypoints})
	if err != nil {
		return Aggregate{}, err
	}
	if result.PreviewFingerprint != request.ExpectedPreviewFingerprint {
		return Aggregate{}, ErrPreviewStale
	}
	materialization, err := materialize(actorID, aggregate.Route.ID, result, request.Waypoints)
	if err != nil {
		return Aggregate{}, err
	}
	corrected, err := s.store.ReplaceRoute(ctx, actorID, walkID, []Status{StatusDraft, StatusReview}, materialization)
	if isStaleMaterialization(err) {
		return Aggregate{}, ErrPreviewStale
	}
	return corrected, err
}

func isStaleMaterialization(err error) bool {
	return errors.Is(err, ErrMaterializationGeoVersionStale) || errors.Is(err, ErrSegmentGeoVersionMismatch)
}

func (s *Service) transition(ctx context.Context, actorID, walkID string, from, to, idempotent Status) (Aggregate, error) {
	if err := validateIDs(actorID, walkID); err != nil {
		return Aggregate{}, err
	}
	aggregate, err := s.store.Get(ctx, actorID, walkID)
	if err != nil {
		return Aggregate{}, err
	}
	if aggregate.Walk.Status == idempotent {
		return aggregate, nil
	}
	if aggregate.Walk.Status != from {
		return Aggregate{}, ErrInvalidState
	}
	result, err := s.store.Transition(ctx, actorID, walkID, from, to, s.now())
	if errors.Is(err, ErrConcurrentChange) {
		current, loadErr := s.store.Get(ctx, actorID, walkID)
		if loadErr == nil && current.Walk.Status == to {
			return current, nil
		}
	}
	return result, err
}

func materialize(actorID, routeID string, result preview.Result, waypoints []port.Point) (Materialization, error) {
	if routeID == "" {
		var err error
		routeID, err = newUUID()
		if err != nil {
			return Materialization{}, err
		}
	}
	routingProvenance, err := json.Marshal(result.Materialization.RoutingMetadata)
	if err != nil {
		return Materialization{}, err
	}
	analysisProvenanceJSON, err := json.Marshal(analysisProvenance{Version: result.Materialization.AnalysisVersion,
		Matching: result.Materialization.Matching, CoverageProfile: result.ExplorationPreview.CoverageProfile,
		Normalization: result.GeoDataVersion.NormalizationVersion})
	if err != nil {
		return Materialization{}, err
	}
	matchedBySegment := map[string]float64{}
	confidenceBySegment := map[string]float64{}
	for _, fragment := range result.ExplorationPreview.MatchedFragments {
		matchedBySegment[fragment.SegmentID] += math.Max(0, fragment.RouteEndMeters-fragment.RouteStartMeters)
		if fragment.Score.Confidence > confidenceBySegment[fragment.SegmentID] {
			confidenceBySegment[fragment.SegmentID] = fragment.Score.Confidence
		}
	}
	matches := make([]SegmentMatch, 0, len(result.ExplorationPreview.CoverageSegments))
	for _, segment := range result.ExplorationPreview.CoverageSegments {
		confidence := confidenceBySegment[segment.SegmentID]
		matches = append(matches, SegmentMatch{SegmentID: segment.SegmentID, Classification: string(segment.Classification),
			MatchedMeters: matchedBySegment[segment.SegmentID], CoveredMeters: segment.CoveredMeters,
			DirectMeters: segment.DirectMeters, RequiredMeters: segment.RequiredMeters, Status: segment.Status,
			Provenance: segment.Provenance, Confidence: &confidence})
	}
	return Materialization{Route: Route{ID: routeID, ActorID: actorID, CityID: result.GeoDataVersion.CityID,
		GeoDataVersionID: result.GeoDataVersion.ID, Profile: result.Routing.Profile, Waypoints: waypoints,
		Geometry: result.Routing.Geometry, NormalizedGeometry: result.ExplorationPreview.NormalizedRoute,
		DistanceMeters: result.Routing.DistanceMeters, EstimatedDurationSeconds: int(math.Ceil(result.Routing.DurationSeconds)),
		RoutingProvenance: routingProvenance, AnalysisProvenance: analysisProvenanceJSON,
		MaterializationFingerprint: result.PreviewFingerprint, Revision: 1}, Matches: matches}, nil
}

func validateActor(actorID string) error {
	if !uuidPattern.MatchString(actorID) {
		return fmt.Errorf("%w: actor id must be a UUID", ErrInvalidRequest)
	}
	return nil
}
func validateIDs(actorID, walkID string) error {
	if err := validateActor(actorID); err != nil {
		return err
	}
	if !uuidPattern.MatchString(walkID) {
		return fmt.Errorf("%w: walk id must be a UUID", ErrInvalidRequest)
	}
	return nil
}
func validateCreate(r CreateRequest) error {
	if !uuidPattern.MatchString(r.ClientRequestID) || !uuidPattern.MatchString(r.CityID) {
		return fmt.Errorf("%w: ids must be UUIDs", ErrInvalidRequest)
	}
	return validateRouteInput(r.Profile, r.ExpectedPreviewFingerprint, r.Waypoints)
}
func validateRouteInput(profile, expected string, waypoints []port.Point) error {
	if profile != "pedestrian" || expected == "" || len(waypoints) < 2 || len(waypoints) > 10 {
		return fmt.Errorf("%w: invalid route materialization input", ErrInvalidRequest)
	}
	for _, p := range waypoints {
		if math.IsNaN(p.Lat) || math.IsNaN(p.Lon) || math.IsInf(p.Lat, 0) || math.IsInf(p.Lon, 0) || p.Lat < -90 || p.Lat > 90 || p.Lon < -180 || p.Lon > 180 {
			return fmt.Errorf("%w: invalid waypoint", ErrInvalidRequest)
		}
	}
	return nil
}

func hashRequest(request CreateRequest) (string, error) {
	b, err := json.Marshal(request)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}
func newUUID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}
