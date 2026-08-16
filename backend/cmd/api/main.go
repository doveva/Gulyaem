package main

import (
	"context"
	"errors"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Asksel-Ecosystem/askcel-go/observability"
	"github.com/Asksel-Ecosystem/askcel-go/runtime"
	"github.com/doveva/Gulyaem/backend/internal/config"
	"github.com/doveva/Gulyaem/backend/internal/exploration"
	"github.com/doveva/Gulyaem/backend/internal/geo/querying"
	"github.com/doveva/Gulyaem/backend/internal/geo/routeanalysis"
	"github.com/doveva/Gulyaem/backend/internal/platform/database"
	explorationdb "github.com/doveva/Gulyaem/backend/internal/platform/database/exploration"
	"github.com/doveva/Gulyaem/backend/internal/platform/database/geoquery"
	routeanalysisdb "github.com/doveva/Gulyaem/backend/internal/platform/database/routeanalysis"
	walksdb "github.com/doveva/Gulyaem/backend/internal/platform/database/walks"
	"github.com/doveva/Gulyaem/backend/internal/platform/routing/valhalla"
	"github.com/doveva/Gulyaem/backend/internal/routing/preview"
	"github.com/doveva/Gulyaem/backend/internal/transport/httpapi"
	"github.com/doveva/Gulyaem/backend/internal/walks"
)

func main() {
	if err := run(); err != nil {
		slog.Error("api stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	meta, err := runtime.Load()
	if err != nil {
		return err
	}

	telemetry, err := observability.Setup(ctx, observability.Config{Metadata: meta})
	if err != nil {
		return err
	}
	defer func() {
		// Shutdown is bounded by the caller, not by the exporter: a collector
		// that stopped answering must not hold the process open.
		flush, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := telemetry.Shutdown(flush); err != nil {
			log.Printf("telemetry shutdown: %v", err)
		}
	}()

	logger := newLogger(cfg.LogLevel, meta)
	slog.SetDefault(logger)

	startupCtx, cancelStartup := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelStartup()

	db, err := database.Open(startupCtx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer db.Close()
	geoRepository := geoquery.New(db)
	geoService := querying.NewService(geoRepository)
	routeAnalysisRepository := routeanalysisdb.New(db, geoRepository)
	routeAnalyzer := routeanalysis.NewAnalyzer(routeAnalysisRepository)
	routeAnalysisService, err := routeanalysis.NewFixtureService(routeAnalyzer, cfg.GeoDataPath)
	if err != nil {
		return err
	}
	routingMetadata := valhalla.NewFileMetadataSource(cfg.RoutingDatasetMetadataPath)
	routingHTTPClient := telemetry.HTTPClient(&http.Client{Timeout: cfg.RoutingTimeout})
	routingEngine := valhalla.NewWithHTTPClient(cfg.ValhallaURL, routingHTTPClient, routingMetadata)
	routePreviewService := preview.NewService(routingEngine, routeAnalyzer, logger)
	walkRepository := walksdb.New(db)
	walkService := walks.NewService(routePreviewService, walkRepository)
	explorationRepository := explorationdb.New(db)
	explorationService := exploration.NewService(explorationRepository, logger)

	handler := httpapi.NewHandler(httpapi.Dependencies{
		Database:       db,
		Logger:         logger,
		Environment:    meta.Environment,
		AllowedOrigins: cfg.CORSAllowedOrigins,
		Geo:            geoService,
		RouteAnalysis:  routeAnalysisService,
		RoutePreview:   routePreviewService,
		Routing:        routingEngine,
		RoutingDataset: routePreviewService,
		Actor:          httpapi.StaticActorResolver{ID: cfg.DevelopmentActorID},
		Walks:          walkService,
		Exploration:    explorationService,
	})
	server := &http.Server{
		Addr:              cfg.HTTPAddress,
		Handler:           telemetry.HTTPHandler(handler),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("api listening",
			"address", cfg.HTTPAddress,
			"runtime", meta.String(),
			"telemetry_exporting", telemetry.Exporting(),
		)
		serverErrors <- server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		handler.Drain()
		shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancelShutdown()
		if err := server.Shutdown(shutdownCtx); err != nil {
			return errors.New("graceful shutdown failed: " + err.Error())
		}
		logger.Info("api stopped gracefully")
		return nil
	case err := <-serverErrors:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

func newLogger(levelName string, meta runtime.Metadata) *slog.Logger {
	level := slog.LevelInfo
	switch levelName {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}

	return observability.NewLogger(
		slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}),
		observability.WithMetadata(meta),
	)
}
