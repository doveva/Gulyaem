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
	"github.com/doveva/Gulyaem/backend/internal/geo/importing"
	"github.com/doveva/Gulyaem/backend/internal/platform/database"
	"github.com/doveva/Gulyaem/backend/internal/platform/database/geoversion"
	"github.com/doveva/Gulyaem/backend/internal/platform/osm/pbf"
)

type options struct {
	fixture              string
	file                 string
	cityCode             string
	normalizationVersion string
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)
	if err := run(logger, os.Args[1:]); err != nil {
		logger.Error("geo import stopped", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger, arguments []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	parsed, err := parseOptions(arguments, cfg.GeoTestArea)
	if err != nil {
		return err
	}
	request, err := buildRequest(cfg.GeoDataPath, parsed)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	connectCtx, cancelConnect := context.WithTimeout(ctx, 10*time.Second)
	defer cancelConnect()
	db, err := database.Open(connectCtx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer db.Close()

	service := importing.NewService(geoversion.New(db), pbf.NewScanner())
	result, err := service.Import(ctx, request)
	fields := []any{
		"outcome", result.Outcome,
		"version_id", result.Version.ID,
		"city_code", request.CityCode,
		"source_checksum", result.Version.SourceChecksum,
		"normalization_version", request.NormalizationVersion,
		"objects_processed", result.Version.ImportReport.ObjectsProcessed,
		"nodes_processed", result.Version.ImportReport.NodesProcessed,
		"ways_processed", result.Version.ImportReport.WaysProcessed,
		"relations_processed", result.Version.ImportReport.RelationsProcessed,
		"duration_ms", result.Version.ImportReport.DurationMillis,
	}
	if err != nil {
		logger.Error("geo import failed", append(fields, "error", err)...)
		return err
	}
	logger.Info("geo import completed", fields...)
	return nil
}

func parseOptions(arguments []string, defaultFixture string) (options, error) {
	flags := flag.NewFlagSet("geo-import", flag.ContinueOnError)
	var parsed options
	flags.StringVar(&parsed.fixture, "fixture", defaultFixture, "fixture name below GEO_DATA_PATH/test-areas")
	flags.StringVar(&parsed.file, "file", "", "explicit .osm.pbf path (takes precedence over --fixture)")
	flags.StringVar(&parsed.cityCode, "city-code", "spb", "target city code for --file")
	flags.StringVar(&parsed.normalizationVersion, "normalization-version", envOrDefault("NORMALIZATION_VERSION", "stage1-v1"), "normalization rules version")
	if err := flags.Parse(arguments); err != nil {
		return options{}, err
	}
	if flags.NArg() != 0 {
		return options{}, errors.New("unexpected positional arguments")
	}
	return parsed, nil
}

func buildRequest(dataRoot string, parsed options) (importing.ImportRequest, error) {
	if strings.TrimSpace(parsed.file) != "" {
		return importing.ImportRequest{
			CityCode:             parsed.cityCode,
			FilePath:             parsed.file,
			Source:               "openstreetmap",
			NormalizationVersion: parsed.normalizationVersion,
		}, nil
	}
	if strings.TrimSpace(parsed.fixture) == "" {
		return importing.ImportRequest{}, errors.New("either --fixture or --file is required")
	}
	fixture, err := importing.LoadFixture(dataRoot, parsed.fixture)
	if err != nil {
		return importing.ImportRequest{}, err
	}
	timestamp := fixture.Manifest.Source.RetrievedAt
	return importing.ImportRequest{
		CityCode:             fixture.Manifest.CityCode,
		FilePath:             fixture.FilePath,
		ExpectedChecksum:     fixture.Manifest.PBF.SHA256,
		Source:               fixture.Manifest.Source.Name,
		SourceURL:            fixture.Manifest.Source.URL,
		SourceTimestamp:      &timestamp,
		NormalizationVersion: parsed.normalizationVersion,
	}, nil
}

func envOrDefault(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}
