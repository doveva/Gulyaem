package districting

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/doveva/Gulyaem/backend/internal/geo/domain"
)

const maximumImportErrorLength = 4000

type Service struct {
	store VersionStore
	now   func() time.Time
}

type ImportRequest struct {
	CityCode             string
	FilePath             string
	ExpectedChecksum     string
	ExpectedFeatureCount int
	Source               string
	SourceURL            string
	SourceTimestamp      *time.Time
	NormalizationVersion string
}

type ImportResult struct {
	Version domain.DistrictDataVersion
	Outcome string
}

func NewService(store VersionStore) *Service {
	return &Service{store: store, now: time.Now}
}

func (service *Service) Import(ctx context.Context, request ImportRequest) (ImportResult, error) {
	if err := validateRequest(request); err != nil {
		return ImportResult{}, err
	}
	startedAt := service.now()
	checksum, size, err := fileChecksum(request.FilePath)
	if err != nil {
		return ImportResult{}, fmt.Errorf("checksum district GeoJSON: %w", err)
	}
	begin, err := service.store.BeginImport(ctx, domain.BeginDistrictImport{
		CityCode: request.CityCode, Source: request.Source, SourceURL: request.SourceURL,
		SourceChecksum: checksum, SourceFileName: filepath.Base(request.FilePath),
		SourceSizeBytes: size, NormalizationVersion: request.NormalizationVersion,
	})
	if err != nil {
		return ImportResult{}, err
	}
	if begin.AlreadyReady {
		return ImportResult{Version: begin.Version, Outcome: "already_ready"}, nil
	}

	report := domain.DistrictImportReport{}
	fail := func(importError error) (ImportResult, error) {
		report.Outcome = "failed"
		report.DurationMillis = service.now().Sub(startedAt).Milliseconds()
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
	districts, err := parseGeoJSON(request.FilePath)
	if err != nil {
		return fail(err)
	}
	report.FeaturesProcessed = int64(len(districts))
	if request.ExpectedFeatureCount > 0 && len(districts) != request.ExpectedFeatureCount {
		return fail(fmt.Errorf("fixture feature count mismatch: got %d, want %d", len(districts), request.ExpectedFeatureCount))
	}
	report.Outcome = "imported"
	report.DistrictsPublished = int64(len(districts))
	report.DurationMillis = service.now().Sub(startedAt).Milliseconds()
	version, err := service.store.CompleteImport(ctx, begin.Version.ID, request.SourceTimestamp, report, districts)
	if err != nil {
		return fail(fmt.Errorf("publish district import: %w", err))
	}
	return ImportResult{Version: version, Outcome: "imported"}, nil
}

func validateRequest(request ImportRequest) error {
	switch {
	case strings.TrimSpace(request.CityCode) == "":
		return errors.New("city code is required")
	case strings.TrimSpace(request.FilePath) == "":
		return errors.New("GeoJSON file path is required")
	case strings.TrimSpace(request.Source) == "":
		return errors.New("source is required")
	case strings.TrimSpace(request.NormalizationVersion) == "":
		return errors.New("normalization version is required")
	default:
		return nil
	}
}

type featureCollection struct {
	Type     string    `json:"type"`
	Features []feature `json:"features"`
}

type feature struct {
	ID         string          `json:"id"`
	Type       string          `json:"type"`
	Properties properties      `json:"properties"`
	Geometry   json.RawMessage `json:"geometry"`
}

type properties struct {
	ExternalID    string `json:"externalId"`
	Name          string `json:"name"`
	Kind          string `json:"kind"`
	AdminLevel    int    `json:"adminLevel"`
	OSMRelationID int64  `json:"osmRelationId"`
	Wikidata      string `json:"wikidata"`
}

func parseGeoJSON(path string) ([]domain.DistrictDraft, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read district GeoJSON: %w", err)
	}
	var collection featureCollection
	if err := json.Unmarshal(contents, &collection); err != nil {
		return nil, fmt.Errorf("decode district GeoJSON: %w", err)
	}
	if collection.Type != "FeatureCollection" || len(collection.Features) == 0 {
		return nil, errors.New("district GeoJSON must be a non-empty FeatureCollection")
	}
	seen := make(map[string]struct{}, len(collection.Features))
	districts := make([]domain.DistrictDraft, 0, len(collection.Features))
	for index, item := range collection.Features {
		externalID := strings.TrimSpace(item.Properties.ExternalID)
		if item.Type != "Feature" || externalID == "" || strings.TrimSpace(item.Properties.Name) == "" ||
			strings.TrimSpace(item.Properties.Kind) == "" || len(item.Geometry) == 0 || string(item.Geometry) == "null" {
			return nil, fmt.Errorf("district feature %d is incomplete", index)
		}
		if _, exists := seen[externalID]; exists {
			return nil, fmt.Errorf("duplicate district externalId %q", externalID)
		}
		seen[externalID] = struct{}{}
		var geometryHeader struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(item.Geometry, &geometryHeader); err != nil ||
			(geometryHeader.Type != "Polygon" && geometryHeader.Type != "MultiPolygon") {
			return nil, fmt.Errorf("district feature %d must contain Polygon or MultiPolygon geometry", index)
		}
		attributes := map[string]any{"adminLevel": item.Properties.AdminLevel, "osmRelationId": item.Properties.OSMRelationID}
		if item.Properties.Wikidata != "" {
			attributes["wikidata"] = item.Properties.Wikidata
		}
		districts = append(districts, domain.DistrictDraft{
			ExternalID: externalID, Name: strings.TrimSpace(item.Properties.Name), Kind: strings.TrimSpace(item.Properties.Kind),
			GeometryJSON: item.Geometry, Attributes: attributes,
		})
	}
	return districts, nil
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
		return "", 0, errors.New("district GeoJSON is empty")
	}
	return fmt.Sprintf("%x", hash.Sum(nil)), size, nil
}

func truncateError(message string) string {
	if len(message) <= maximumImportErrorLength {
		return message
	}
	return message[:maximumImportErrorLength]
}
