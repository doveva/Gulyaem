package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/doveva/Gulyaem/backend/internal/config"
	"github.com/doveva/Gulyaem/backend/internal/geo/districting"
	"github.com/doveva/Gulyaem/backend/internal/platform/database"
	"github.com/doveva/Gulyaem/backend/internal/platform/database/districtversion"
)

type options struct {
	fixture              string
	file                 string
	cityCode             string
	normalizationVersion string
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger, os.Args[1:]); err != nil {
		logger.Error("district import stopped", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger, arguments []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	parsed, err := parseOptions(arguments, cfg.DistrictTestArea)
	if err != nil {
		return err
	}
	request, err := buildRequest(cfg.GeoDataPath, parsed)
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	connectCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	db, err := database.Open(connectCtx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer db.Close()
	result, err := districting.NewService(districtversion.New(db)).Import(ctx, request)
	fields := []any{
		"outcome", result.Outcome, "version_id", result.Version.ID,
		"city_code", request.CityCode, "source_checksum", result.Version.SourceChecksum,
		"normalization_version", request.NormalizationVersion,
		"features_processed", result.Version.ImportReport.FeaturesProcessed,
		"districts_published", result.Version.ImportReport.DistrictsPublished,
		"duration_ms", result.Version.ImportReport.DurationMillis,
	}
	if err != nil {
		logger.Error("district import failed", append(fields, "error", err)...)
		return err
	}
	logger.Info("district import completed", fields...)
	return nil
}

func parseOptions(arguments []string, defaultFixture string) (options, error) {
	flags := flag.NewFlagSet("district-import", flag.ContinueOnError)
	var parsed options
	flags.StringVar(&parsed.fixture, "fixture", defaultFixture, "fixture name below GEO_DATA_PATH/districts")
	flags.StringVar(&parsed.file, "file", "", "explicit GeoJSON path (takes precedence over --fixture)")
	flags.StringVar(&parsed.cityCode, "city-code", "spb", "target city code for --file")
	flags.StringVar(&parsed.normalizationVersion, "normalization-version", envOrDefault("DISTRICT_NORMALIZATION_VERSION", "stage1-districts-v1"), "district normalization rules version")
	if err := flags.Parse(arguments); err != nil {
		return options{}, err
	}
	if flags.NArg() != 0 {
		return options{}, errors.New("unexpected positional arguments")
	}
	return parsed, nil
}

func buildRequest(dataRoot string, parsed options) (districting.ImportRequest, error) {
	if strings.TrimSpace(parsed.file) != "" {
		return districting.ImportRequest{CityCode: parsed.cityCode, FilePath: parsed.file, Source: "openstreetmap", NormalizationVersion: parsed.normalizationVersion}, nil
	}
	if strings.TrimSpace(parsed.fixture) == "" {
		return districting.ImportRequest{}, errors.New("either --fixture or --file is required")
	}
	fixture, err := districting.LoadFixture(dataRoot, parsed.fixture)
	if err != nil {
		return districting.ImportRequest{}, err
	}
	timestamp := fixture.Manifest.Source.OSMBaseTimestamp
	return districting.ImportRequest{
		CityCode: fixture.Manifest.CityCode, FilePath: fixture.FilePath,
		ExpectedChecksum: fixture.Manifest.GeoJSON.SHA256, ExpectedFeatureCount: fixture.Manifest.GeoJSON.FeatureCount,
		Source: fixture.Manifest.Source.Name, SourceURL: fixture.Manifest.Source.URL,
		SourceTimestamp: &timestamp, NormalizationVersion: parsed.normalizationVersion,
	}, nil
}

func envOrDefault(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}
