package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/doveva/Gulyaem/backend/internal/geo/routeanalysis"
	"github.com/doveva/Gulyaem/backend/internal/platform/database"
	"github.com/doveva/Gulyaem/backend/internal/platform/database/geoquery"
	routeanalysisdb "github.com/doveva/Gulyaem/backend/internal/platform/database/routeanalysis"
	"github.com/doveva/Gulyaem/backend/internal/routingspike"
)

const defaultCityID = "01900000-0000-7000-8000-000000000001"

func main() {
	if err := run(); err != nil {
		slog.Error("routing spike failed", "error", err)
		os.Exit(1)
	}
}

func run() error {
	root := flag.String("root", "..", "repository root")
	fixture := flag.String("fixture", "data/routing-spike/spb-stage1/cases.json", "routing fixture path")
	output := flag.String("output", "frontend/public/routing-spike/comparison.json", "comparison report path")
	setup := flag.String("setup-metrics", ".routing/setup-metrics.json", "local setup metrics path")
	cityID := flag.String("city-id", defaultCityID, "city UUID for StreetSegment matching")
	skipMatcher := flag.Bool("skip-street-matcher", false, "run engine comparison without PostgreSQL StreetSegment matching")
	flag.Parse()

	absoluteRoot, err := filepath.Abs(*root)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()

	var analyzer routingspike.Analyzer
	var closeDatabase func()
	if !*skipMatcher {
		databaseURL := os.Getenv("DATABASE_URL")
		if databaseURL == "" {
			return errors.New("DATABASE_URL is required; run through make routing-benchmark or use --skip-street-matcher")
		}
		db, openErr := database.Open(ctx, databaseURL)
		if openErr != nil {
			return openErr
		}
		closeDatabase = db.Close
		geoRepository := geoquery.New(db)
		service, serviceErr := routeanalysis.NewService(
			routeanalysisdb.New(db, geoRepository), filepath.Join(absoluteRoot, "data"),
		)
		if serviceErr != nil {
			db.Close()
			return serviceErr
		}
		analyzer = &streetSegmentAnalyzer{ctx: ctx, cityID: *cityID, service: service}
	}
	if closeDatabase != nil {
		defer closeDatabase()
	}

	report, err := routingspike.Run(ctx, routingspike.Options{
		RepositoryRoot:   absoluteRoot,
		FixturePath:      *fixture,
		OutputPath:       resolve(absoluteRoot, *output),
		SetupMetricsPath: resolve(absoluteRoot, *setup),
		Analyzer:         analyzer,
	})
	if err != nil {
		return err
	}
	fmt.Printf("routing comparison %s: %d cases, %d map-matching results -> %s\n",
		report.Status, len(report.Cases), len(report.MapMatching), resolve(absoluteRoot, *output))
	for _, summary := range report.Summary {
		fmt.Printf("  %-12s routes=%d/%d corridor=%.1f%%/%.1f%% matcher=%.1f%% p50=%.2fms map-matching=%s\n",
			summary.EngineID, summary.SuccessfulRoutes, summary.TotalRoutes,
			summary.MeanCandidateCorridorRatio*100, summary.MeanReferenceCorridorRatio*100,
			summary.MeanStreetSegmentMatchRatio*100, summary.MedianWarmLatencyMs, summary.MapMatchingStatus)
	}
	return nil
}

type streetSegmentAnalyzer struct {
	ctx     context.Context
	cityID  string
	service *routeanalysis.Service
}

func (analyzer *streetSegmentAnalyzer) Analyze(routeID string, geometry json.RawMessage) (*routingspike.MatcherMetrics, error) {
	analysis, err := analyzer.service.AnalyzeGeometry(analyzer.ctx, analyzer.cityID, routeID, geometry, routeanalysis.AnalyzeRequest{
		Matching: routeanalysis.DefaultMatchingParameters(),
		Coverage: routeanalysis.CoverageProfiles["balanced"],
	})
	if err != nil {
		return nil, err
	}
	reasonMeters := make(map[string]float64)
	for _, fragment := range analysis.MatchedFragments {
		reasonMeters[fragment.ReasonCode] += max(0, fragment.RouteEndMeters-fragment.RouteStartMeters)
	}
	return &routingspike.MatcherMetrics{
		RouteMatchedRatio:          analysis.Metrics.RouteMatchedRatio,
		RouteUnmatchedLengthMeters: analysis.Metrics.RouteUnmatchedLengthMeters,
		MatchedReasonMeters:        reasonMeters,
	}, nil
}

func resolve(root, path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(root, path)
}
