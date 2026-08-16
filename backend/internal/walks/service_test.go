package walks

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/doveva/Gulyaem/backend/internal/geo/domain"
	"github.com/doveva/Gulyaem/backend/internal/geo/routeanalysis"
	"github.com/doveva/Gulyaem/backend/internal/routing/port"
	"github.com/doveva/Gulyaem/backend/internal/routing/preview"
)

const (
	testActor   = "01900000-0000-7000-8000-000000000003"
	testCity    = "01900000-0000-7000-8000-000000000001"
	testRequest = "01900000-0000-7000-8000-000000000010"
)

type fakePreviewer struct {
	result preview.Result
	calls  int
}

func (f *fakePreviewer) Create(context.Context, preview.Request) (preview.Result, error) {
	f.calls++
	return f.result, nil
}

type fakeStore struct {
	aggregate   *Aggregate
	createCalls int
	createErr   error
	replaceErr  error
}

func (s *fakeStore) FindByClientRequest(_ context.Context, actor, request string) (Aggregate, error) {
	if s.aggregate != nil && s.aggregate.Walk.ActorID == actor && s.aggregate.Walk.ClientRequestID == request {
		return *s.aggregate, nil
	}
	return Aggregate{}, ErrNotFound
}
func (s *fakeStore) Create(_ context.Context, m Materialization, w Walk) (Aggregate, error) {
	s.createCalls++
	if s.createErr != nil {
		return Aggregate{}, s.createErr
	}
	now := time.Date(2026, 8, 12, 12, 0, 0, 0, time.UTC)
	w.CreatedAt = now
	w.UpdatedAt = now
	m.Route.CreatedAt = now
	m.Route.UpdatedAt = now
	a := Aggregate{Walk: w, Route: m.Route}
	s.aggregate = &a
	return a, nil
}
func (s *fakeStore) Get(_ context.Context, actor, id string) (Aggregate, error) {
	if s.aggregate == nil || s.aggregate.Walk.ActorID != actor || s.aggregate.Walk.ID != id {
		return Aggregate{}, ErrNotFound
	}
	return *s.aggregate, nil
}
func (s *fakeStore) Transition(_ context.Context, actor, id string, from, to Status, at time.Time) (Aggregate, error) {
	a, err := s.Get(context.Background(), actor, id)
	if err != nil {
		return Aggregate{}, err
	}
	if a.Walk.Status != from {
		return Aggregate{}, ErrConcurrentChange
	}
	a.Walk.Status = to
	a.Walk.UpdatedAt = at
	if to == StatusActive {
		a.Walk.StartedAt = &at
	}
	if to == StatusReview {
		a.Walk.FinishedAt = &at
	}
	s.aggregate = &a
	return a, nil
}
func (s *fakeStore) ReplaceRoute(_ context.Context, actor, id string, allowed []Status, m Materialization) (Aggregate, error) {
	if s.replaceErr != nil {
		return Aggregate{}, s.replaceErr
	}
	a, err := s.Get(context.Background(), actor, id)
	if err != nil {
		return Aggregate{}, err
	}
	ok := false
	for _, status := range allowed {
		ok = ok || status == a.Walk.Status
	}
	if !ok {
		return Aggregate{}, ErrRouteNotEditable
	}
	m.Route.Revision = a.Route.Revision + 1
	a.Route = m.Route
	s.aggregate = &a
	return a, nil
}

func TestCreateRejectsStalePreviewWithoutPersistence(t *testing.T) {
	p := &fakePreviewer{result: previewResult("fresh")}
	store := &fakeStore{}
	service := NewService(p, store)
	_, _, err := service.Create(context.Background(), testActor, createRequest("stale"))
	if !errors.Is(err, ErrPreviewStale) {
		t.Fatalf("error=%v", err)
	}
	if store.createCalls != 0 {
		t.Fatalf("create calls=%d", store.createCalls)
	}
}

func TestCreateMapsVersionChangedDuringPersistenceToStalePreview(t *testing.T) {
	p := &fakePreviewer{result: previewResult("fingerprint")}
	store := &fakeStore{createErr: ErrMaterializationGeoVersionStale}
	service := NewService(p, store)
	_, _, err := service.Create(context.Background(), testActor, createRequest("fingerprint"))
	if !errors.Is(err, ErrPreviewStale) {
		t.Fatalf("error=%v", err)
	}
}

func TestCreateMapsSegmentVersionMismatchToStalePreview(t *testing.T) {
	p := &fakePreviewer{result: previewResult("fingerprint")}
	store := &fakeStore{createErr: ErrSegmentGeoVersionMismatch}
	service := NewService(p, store)
	_, _, err := service.Create(context.Background(), testActor, createRequest("fingerprint"))
	if !errors.Is(err, ErrPreviewStale) {
		t.Fatalf("error=%v", err)
	}
}

func TestCreateRetryReturnsSameWalk(t *testing.T) {
	p := &fakePreviewer{result: previewResult("fingerprint")}
	store := &fakeStore{}
	service := NewService(p, store)
	first, created, err := service.Create(context.Background(), testActor, createRequest("fingerprint"))
	if err != nil || !created {
		t.Fatalf("first created=%v err=%v", created, err)
	}
	second, created, err := service.Create(context.Background(), testActor, createRequest("fingerprint"))
	if err != nil || created {
		t.Fatalf("retry created=%v err=%v", created, err)
	}
	if first.Walk.ID != second.Walk.ID || p.calls != 1 || store.createCalls != 1 {
		t.Fatalf("retry duplicated materialization: %#v %#v calls=%d/%d", first, second, p.calls, store.createCalls)
	}
}

func TestLifecycleRetriesKeepServerTimestampsStable(t *testing.T) {
	p := &fakePreviewer{result: previewResult("fingerprint")}
	store := &fakeStore{}
	service := NewService(p, store)
	created, _, _ := service.Create(context.Background(), testActor, createRequest("fingerprint"))
	clock := time.Date(2026, 8, 12, 13, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return clock }
	active, err := service.Start(context.Background(), testActor, created.Walk.ID)
	if err != nil {
		t.Fatal(err)
	}
	clock = clock.Add(time.Hour)
	retry, err := service.Start(context.Background(), testActor, created.Walk.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !active.Walk.StartedAt.Equal(*retry.Walk.StartedAt) {
		t.Fatalf("start timestamp changed")
	}
	review, err := service.Finish(context.Background(), testActor, created.Walk.ID)
	if err != nil || review.Walk.Status != StatusReview {
		t.Fatalf("finish=%s err=%v", review.Walk.Status, err)
	}
}

func TestRouteCorrectionRejectedWhileActive(t *testing.T) {
	p := &fakePreviewer{result: previewResult("fingerprint")}
	store := &fakeStore{}
	service := NewService(p, store)
	created, _, _ := service.Create(context.Background(), testActor, createRequest("fingerprint"))
	_, _ = service.Start(context.Background(), testActor, created.Walk.ID)
	_, err := service.CorrectRoute(context.Background(), testActor, created.Walk.ID, CorrectRouteRequest{Profile: "pedestrian", ExpectedPreviewFingerprint: "fingerprint", Waypoints: createRequest("fingerprint").Waypoints})
	if !errors.Is(err, ErrRouteNotEditable) {
		t.Fatalf("error=%v", err)
	}
}

func TestCorrectionMapsVersionChangedDuringPersistenceToStalePreview(t *testing.T) {
	p := &fakePreviewer{result: previewResult("fingerprint")}
	store := &fakeStore{}
	service := NewService(p, store)
	created, _, err := service.Create(context.Background(), testActor, createRequest("fingerprint"))
	if err != nil {
		t.Fatal(err)
	}
	store.replaceErr = ErrMaterializationGeoVersionStale
	_, err = service.CorrectRoute(context.Background(), testActor, created.Walk.ID, CorrectRouteRequest{Profile: "pedestrian", ExpectedPreviewFingerprint: "fingerprint", Waypoints: createRequest("fingerprint").Waypoints})
	if !errors.Is(err, ErrPreviewStale) {
		t.Fatalf("error=%v", err)
	}
}

func TestCorrectionMapsSegmentVersionMismatchToStalePreview(t *testing.T) {
	p := &fakePreviewer{result: previewResult("fingerprint")}
	store := &fakeStore{}
	service := NewService(p, store)
	created, _, err := service.Create(context.Background(), testActor, createRequest("fingerprint"))
	if err != nil {
		t.Fatal(err)
	}
	store.replaceErr = ErrSegmentGeoVersionMismatch
	_, err = service.CorrectRoute(context.Background(), testActor, created.Walk.ID, CorrectRouteRequest{Profile: "pedestrian", ExpectedPreviewFingerprint: "fingerprint", Waypoints: createRequest("fingerprint").Waypoints})
	if !errors.Is(err, ErrPreviewStale) {
		t.Fatalf("error=%v", err)
	}
}

func createRequest(fingerprint string) CreateRequest {
	return CreateRequest{ClientRequestID: testRequest, CityID: testCity, Profile: "pedestrian", ExpectedPreviewFingerprint: fingerprint, Waypoints: []port.Point{{Lat: 59.93, Lon: 30.30}, {Lat: 59.94, Lon: 30.31}}}
}
func previewResult(fingerprint string) preview.Result {
	return preview.Result{PreviewFingerprint: fingerprint, GeoDataVersion: routeanalysis.VersionReference{ID: "01900000-0000-7000-8000-000000000020", CityID: testCity, NormalizationVersion: "stage1-segments-v1"}, Routing: preview.Routing{Engine: "valhalla", Profile: "pedestrian", DistanceMeters: 1000, DurationSeconds: 800, Geometry: json.RawMessage(`{"type":"LineString","coordinates":[[30.3,59.93],[30.31,59.94]]}`)}, ExplorationPreview: preview.ExplorationPreview{CoverageProfile: routeanalysis.CoverageProfiles["balanced"], NormalizedRoute: json.RawMessage(`{"type":"MultiLineString","coordinates":[[[30.3,59.93],[30.31,59.94]]]}`), MatchedFragments: []routeanalysis.MatchedFragment{{SegmentID: "01900000-0000-7000-8000-000000000030", RouteStartMeters: 0, RouteEndMeters: 100, Score: routeanalysis.Score{Confidence: .9}}}, CoverageSegments: []routeanalysis.CoverageSegment{{SegmentID: "01900000-0000-7000-8000-000000000030", Classification: domain.StreetSegmentExplore, LengthMeters: 100, CoveredMeters: 100, DirectMeters: 100, RequiredMeters: 40, Status: "COMPLETED"}}}, Materialization: preview.MaterializationProvenance{RoutingMetadata: port.Metadata{Engine: "valhalla", EngineVersion: "3.7.0", GraphChecksum: "abc"}, AnalysisVersion: routeanalysis.AnalysisVersion, Matching: routeanalysis.DefaultMatchingParameters()}}
}
