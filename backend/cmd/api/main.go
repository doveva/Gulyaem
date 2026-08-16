package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

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

	logger := newLogger(cfg.LogLevel)
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
	routingEngine := valhalla.New(cfg.ValhallaURL, cfg.RoutingTimeout, routingMetadata)
	routePreviewService := preview.NewService(routingEngine, routeAnalyzer, logger)
	walkRepository := walksdb.New(db)
	walkService := walks.NewService(routePreviewService, walkRepository)
	explorationRepository := explorationdb.New(db)
	explorationService := exploration.NewService(explorationRepository, logger)

	handler := httpapi.NewHandler(httpapi.Dependencies{
		Database:       db,
		Logger:         logger,
		Environment:    cfg.Environment,
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
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("api listening", "address", cfg.HTTPAddress, "environment", cfg.Environment)
		serverErrors <- server.ListenAndServe()
	}()

	signalCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	select {
	case <-signalCtx.Done():
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

func newLogger(levelName string) *slog.Logger {
	level := slog.LevelInfo
	switch levelName {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}

	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
}
