package importing

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/doveva/Gulyaem/backend/internal/geo/domain"
	"github.com/doveva/Gulyaem/backend/internal/geo/segmenting"
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
	BBox                 *BBox
	NormalizationVersion string
	MaxSegmentLength     float64
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

	visitor := &collectingVisitor{}
	segmentReport := segmenting.Report{}
	fail := func(importError error) (ImportResult, error) {
		report := visitor.report("failed", segmentReport, service.now().Sub(startedAt))
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
	bbox := request.BBox
	if bbox == nil {
		bbox = metadata.BBox
	}
	segmentInput := visitor.segmentInput(request.MaxSegmentLength)
	if bbox != nil {
		segmentInput.BBox = &segmenting.BBox{
			West: bbox.West, South: bbox.South, East: bbox.East, North: bbox.North,
		}
	}
	segmentResult, err := segmenting.Build(segmentInput)
	if err != nil {
		return fail(fmt.Errorf("generate street segments: %w", err))
	}
	segmentReport = segmentResult.Report
	if bbox == nil {
		segmentReport.Warnings = append(segmentReport.Warnings, "missing_source_bbox")
	}

	report := visitor.report("imported", segmentReport, service.now().Sub(startedAt))
	version, err := service.store.CompleteImport(ctx, begin.Version.ID, sourceTimestamp, report, segmentResult.Segments)
	if err != nil {
		return fail(fmt.Errorf("publish geo import: %w", err))
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
	case request.MaxSegmentLength < 0 || math.IsNaN(request.MaxSegmentLength) || math.IsInf(request.MaxSegmentLength, 0):
		return errors.New("max segment length must be finite and non-negative")
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

type collectingVisitor struct {
	nodes           int64
	ways            int64
	relations       int64
	segmentingNodes []segmenting.Node
	segmentingWays  []segmenting.Way
}

func (visitor *collectingVisitor) VisitNode(node SourceNode) error {
	visitor.nodes++
	visitor.segmentingNodes = append(visitor.segmentingNodes, segmenting.Node{
		SourceID: node.SourceID,
		Point:    domain.Point{Lon: node.Lon, Lat: node.Lat},
		Tags:     node.Tags,
	})
	return nil
}

func (visitor *collectingVisitor) VisitWay(way SourceWay) error {
	visitor.ways++
	visitor.segmentingWays = append(visitor.segmentingWays, segmenting.Way{
		SourceID: way.SourceID,
		NodeIDs:  way.NodeIDs,
		Tags:     way.Tags,
	})
	return nil
}

func (visitor *collectingVisitor) VisitRelation(SourceRelation) error {
	visitor.relations++
	return nil
}

func (visitor *collectingVisitor) segmentInput(maximumLength float64) segmenting.Input {
	return segmenting.Input{
		Nodes:            visitor.segmentingNodes,
		Ways:             visitor.segmentingWays,
		MaxSegmentLength: maximumLength,
	}
}

func (visitor *collectingVisitor) report(outcome string, segments segmenting.Report, duration time.Duration) domain.ImportReport {
	return domain.ImportReport{
		Outcome:                      outcome,
		NodesProcessed:               visitor.nodes,
		WaysProcessed:                visitor.ways,
		RelationsProcessed:           visitor.relations,
		ObjectsProcessed:             visitor.nodes + visitor.ways + visitor.relations,
		CandidateWays:                segments.CandidateWays,
		UnsupportedPedestrianAreas:   segments.UnsupportedPedestrianAreas,
		SegmentsGenerated:            segments.SegmentsGenerated,
		SegmentsRejected:             segments.SegmentsRejected,
		SegmentsClipped:              segments.SegmentsClipped,
		SegmentsDeduplicated:         segments.SegmentsDeduplicated,
		DuplicateGeometry:            segments.DuplicateGeometry,
		ConflictingDuplicateGeometry: segments.ConflictingDuplicateGeometry,
		InvalidGeometries:            segments.InvalidGeometry,
		ZeroLengthSegments:           segments.ZeroLengthSegments,
		ShortSegments:                segments.ShortSegments,
		LongSegments:                 segments.LongSegments,
		ExploreSegments:              segments.ExploreSegments,
		RoutableOnlySegments:         segments.RoutableOnlySegments,
		IgnoreSegments:               segments.IgnoreSegments,
		TotalLengthMeters:            segments.TotalLengthMeters,
		ExplorableLengthMeters:       segments.ExplorableLengthMeters,
		MinSegmentLengthMeters:       segments.MinLengthMeters,
		MedianSegmentLengthMeters:    segments.MedianLengthMeters,
		P95SegmentLengthMeters:       segments.P95LengthMeters,
		MaxSegmentLengthMeters:       segments.MaxLengthMeters,
		Warnings:                     segments.Warnings,
		DurationMillis:               duration.Milliseconds(),
	}
}
