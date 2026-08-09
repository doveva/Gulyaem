package importing

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/doveva/Gulyaem/backend/internal/geo/domain"
)

type storeStub struct {
	beginResult domain.BeginImportResult
	beginInput  domain.BeginImport
	completed   bool
	failed      bool
	report      domain.ImportReport
	segments    []domain.StreetSegmentDraft
	completeErr error
}

func (store *storeStub) BeginImport(_ context.Context, input domain.BeginImport) (domain.BeginImportResult, error) {
	store.beginInput = input
	if store.beginResult.Version.ID == "" {
		store.beginResult.Version = domain.GeoDataVersion{ID: "version-1", SourceChecksum: input.SourceChecksum}
	}
	return store.beginResult, nil
}

func (store *storeStub) CompleteImport(_ context.Context, _ string, _ *time.Time, report domain.ImportReport, segments []domain.StreetSegmentDraft) (domain.GeoDataVersion, error) {
	store.completed = true
	store.report = report
	store.segments = segments
	if store.completeErr != nil {
		return domain.GeoDataVersion{}, store.completeErr
	}
	version := store.beginResult.Version
	version.Status = domain.GeoDataVersionReady
	version.SourceChecksum = store.beginInput.SourceChecksum
	version.ImportReport = report
	return version, nil
}

func (store *storeStub) FailImport(_ context.Context, _ string, report domain.ImportReport, _ error) error {
	store.failed = true
	store.report = report
	return nil
}

type scannerStub struct {
	called bool
	err    error
}

func (scanner *scannerStub) Scan(_ context.Context, _ string, visitor SourceVisitor) (SourceMetadata, error) {
	scanner.called = true
	if scanner.err != nil {
		return SourceMetadata{}, scanner.err
	}
	_ = visitor.VisitNode(SourceNode{SourceID: 1, Lon: 30.31, Lat: 59.94})
	_ = visitor.VisitNode(SourceNode{SourceID: 2, Lon: 30.311, Lat: 59.94})
	_ = visitor.VisitWay(SourceWay{SourceID: 3, NodeIDs: []int64{1, 2}, Tags: map[string]string{"highway": "footway"}})
	_ = visitor.VisitRelation(SourceRelation{SourceID: 4})
	timestamp := time.Date(2026, 8, 9, 16, 54, 41, 0, time.UTC)
	return SourceMetadata{Timestamp: &timestamp}, nil
}

func TestServiceImportsAndCountsSourceObjects(t *testing.T) {
	path := writeSourceFile(t, "valid pbf placeholder")
	store := &storeStub{}
	scanner := &scannerStub{}
	service := NewService(store, scanner)
	service.now = advancingClock()

	result, err := service.Import(context.Background(), ImportRequest{
		CityCode:             "spb",
		FilePath:             path,
		Source:               "openstreetmap",
		NormalizationVersion: "stage1-v1",
	})
	if err != nil {
		t.Fatalf("Import() error = %v", err)
	}
	if result.Outcome != "imported" || !store.completed || store.failed {
		t.Fatalf("unexpected import result: result=%+v completed=%v failed=%v", result, store.completed, store.failed)
	}
	if store.report.ObjectsProcessed != 4 || store.report.NodesProcessed != 2 || store.report.WaysProcessed != 1 || store.report.RelationsProcessed != 1 {
		t.Fatalf("report = %+v", store.report)
	}
	if store.report.SegmentsGenerated != 1 || len(store.segments) != 1 || store.segments[0].Classification != domain.StreetSegmentExplore {
		t.Fatalf("segments not generated: report=%+v segments=%+v", store.report, store.segments)
	}
	if len(store.beginInput.SourceChecksum) != 64 || store.beginInput.SourceSizeBytes <= 0 {
		t.Fatalf("source identity not captured: %+v", store.beginInput)
	}
}

func TestServiceReturnsExistingReadyVersionWithoutScanning(t *testing.T) {
	path := writeSourceFile(t, "same fixture")
	store := &storeStub{beginResult: domain.BeginImportResult{
		Version:      domain.GeoDataVersion{ID: "existing", Status: domain.GeoDataVersionReady},
		AlreadyReady: true,
	}}
	scanner := &scannerStub{}
	service := NewService(store, scanner)

	result, err := service.Import(context.Background(), ImportRequest{
		CityCode:             "spb",
		FilePath:             path,
		Source:               "openstreetmap",
		NormalizationVersion: "stage1-v1",
	})
	if err != nil {
		t.Fatalf("Import() error = %v", err)
	}
	if result.Outcome != "already_ready" || result.Version.ID != "existing" {
		t.Fatalf("result = %+v", result)
	}
	if scanner.called || store.completed || store.failed {
		t.Fatal("idempotent import must not scan or mutate lifecycle")
	}
}

func TestServiceMarksChecksumMismatchFailed(t *testing.T) {
	path := writeSourceFile(t, "corrupted fixture")
	store := &storeStub{}
	scanner := &scannerStub{}
	service := NewService(store, scanner)

	result, err := service.Import(context.Background(), ImportRequest{
		CityCode:             "spb",
		FilePath:             path,
		ExpectedChecksum:     strings.Repeat("0", 64),
		Source:               "openstreetmap",
		NormalizationVersion: "stage1-v1",
	})
	if err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("Import() error = %v", err)
	}
	if result.Outcome != "failed" || !store.failed || scanner.called || store.completed {
		t.Fatalf("unexpected failure lifecycle: result=%+v store=%+v scanner=%+v", result, store, scanner)
	}
}

func TestServiceMarksScannerErrorFailed(t *testing.T) {
	path := writeSourceFile(t, "invalid pbf")
	store := &storeStub{}
	scanner := &scannerStub{err: errors.New("invalid header")}
	service := NewService(store, scanner)

	result, err := service.Import(context.Background(), ImportRequest{
		CityCode:             "spb",
		FilePath:             path,
		Source:               "openstreetmap",
		NormalizationVersion: "stage1-v1",
	})
	if err == nil || result.Outcome != "failed" || !store.failed || store.completed {
		t.Fatalf("unexpected scanner failure: result=%+v err=%v store=%+v", result, err, store)
	}
}

func TestServiceMarksPublicationErrorFailedWithSegmentReport(t *testing.T) {
	path := writeSourceFile(t, "valid pbf placeholder")
	store := &storeStub{completeErr: errors.New("transaction rejected")}
	service := NewService(store, &scannerStub{})

	result, err := service.Import(context.Background(), ImportRequest{
		CityCode:             "spb",
		FilePath:             path,
		Source:               "openstreetmap",
		NormalizationVersion: "stage1-segments-v1",
	})
	if err == nil || result.Outcome != "failed" || !store.completed || !store.failed {
		t.Fatalf("unexpected publication failure: result=%+v err=%v store=%+v", result, err, store)
	}
	if store.report.SegmentsGenerated != 1 || store.report.Outcome != "failed" {
		t.Fatalf("failed publication lost segment report: %+v", store.report)
	}
}

func writeSourceFile(t *testing.T, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "source.osm.pbf")
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func advancingClock() func() time.Time {
	current := time.Date(2026, 8, 9, 0, 0, 0, 0, time.UTC)
	return func() time.Time {
		current = current.Add(time.Second)
		return current
	}
}
