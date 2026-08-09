package importing

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/doveva/Gulyaem/backend/internal/geo/domain"
)

const maximumErrorLength = 4000

type Service struct {
	store   VersionStore
	scanner SourceScanner
	now     func() time.Time
}

type ImportRequest struct {
	CityCode             string
	FilePath             string
	ExpectedChecksum     string
	Source               string
	SourceURL            string
	SourceTimestamp      *time.Time
	NormalizationVersion string
}

type ImportResult struct {
	Version domain.GeoDataVersion
	Outcome string
}

func NewService(store VersionStore, scanner SourceScanner) *Service {
	return &Service{store: store, scanner: scanner, now: time.Now}
}

func (service *Service) Import(ctx context.Context, request ImportRequest) (ImportResult, error) {
	if err := validateRequest(request); err != nil {
		return ImportResult{}, err
	}
	startedAt := service.now()

	checksum, size, err := fileChecksum(request.FilePath)
	if err != nil {
		return ImportResult{}, fmt.Errorf("checksum source PBF: %w", err)
	}

	begin, err := service.store.BeginImport(ctx, domain.BeginImport{
		CityCode:             request.CityCode,
		Source:               request.Source,
		SourceURL:            request.SourceURL,
		SourceChecksum:       checksum,
		SourceFileName:       filepath.Base(request.FilePath),
		SourceSizeBytes:      size,
		NormalizationVersion: request.NormalizationVersion,
	})
	if err != nil {
		return ImportResult{}, err
	}
	if begin.AlreadyReady {
		return ImportResult{Version: begin.Version, Outcome: "already_ready"}, nil
	}

	visitor := &countingVisitor{}
	fail := func(importError error) (ImportResult, error) {
		report := visitor.report("failed", service.now().Sub(startedAt))
		if failError := service.store.FailImport(ctx, begin.Version.ID, report, importError); failError != nil {
			return ImportResult{}, errors.Join(importError, failError)
		}
		begin.Version.Status = domain.GeoDataVersionFailed
		begin.Version.ImportReport = report
		begin.Version.ImportError = truncateError(importError.Error())
		return ImportResult{Version: begin.Version, Outcome: "failed"}, importError
	}

	if request.ExpectedChecksum != "" && !strings.EqualFold(request.ExpectedChecksum, checksum) {
		return fail(fmt.Errorf("fixture checksum mismatch: got %s, want %s", checksum, request.ExpectedChecksum))
	}

	metadata, err := service.scanner.Scan(ctx, request.FilePath, visitor)
	if err != nil {
		return fail(fmt.Errorf("scan source PBF: %w", err))
	}
	sourceTimestamp := metadata.Timestamp
	if sourceTimestamp == nil {
		sourceTimestamp = request.SourceTimestamp
	}

	report := visitor.report("imported", service.now().Sub(startedAt))
	version, err := service.store.CompleteImport(ctx, begin.Version.ID, sourceTimestamp, report)
	if err != nil {
		return ImportResult{}, err
	}
	return ImportResult{Version: version, Outcome: "imported"}, nil
}

func validateRequest(request ImportRequest) error {
	switch {
	case strings.TrimSpace(request.CityCode) == "":
		return errors.New("city code is required")
	case strings.TrimSpace(request.FilePath) == "":
		return errors.New("PBF file path is required")
	case strings.TrimSpace(request.Source) == "":
		return errors.New("source is required")
	case strings.TrimSpace(request.NormalizationVersion) == "":
		return errors.New("normalization version is required")
	default:
		return nil
	}
}

func fileChecksum(path string) (string, int64, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer file.Close()

	hash := sha256.New()
	size, err := io.Copy(hash, file)
	if err != nil {
		return "", 0, err
	}
	if size == 0 {
		return "", 0, errors.New("source PBF is empty")
	}
	return fmt.Sprintf("%x", hash.Sum(nil)), size, nil
}

func truncateError(message string) string {
	if len(message) <= maximumErrorLength {
		return message
	}
	return message[:maximumErrorLength]
}

type countingVisitor struct {
	nodes     int64
	ways      int64
	relations int64
}

func (visitor *countingVisitor) VisitNode(SourceNode) error {
	visitor.nodes++
	return nil
}

func (visitor *countingVisitor) VisitWay(SourceWay) error {
	visitor.ways++
	return nil
}

func (visitor *countingVisitor) VisitRelation(SourceRelation) error {
	visitor.relations++
	return nil
}

func (visitor *countingVisitor) report(outcome string, duration time.Duration) domain.ImportReport {
	return domain.ImportReport{
		Outcome:            outcome,
		NodesProcessed:     visitor.nodes,
		WaysProcessed:      visitor.ways,
		RelationsProcessed: visitor.relations,
		ObjectsProcessed:   visitor.nodes + visitor.ways + visitor.relations,
		DurationMillis:     duration.Milliseconds(),
	}
}
