package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/doveva/Gulyaem/backend/internal/config"
	"github.com/doveva/Gulyaem/backend/internal/exploration"
	"github.com/doveva/Gulyaem/backend/internal/geo/routeanalysis"
	"github.com/doveva/Gulyaem/backend/internal/platform/database"
	explorationdb "github.com/doveva/Gulyaem/backend/internal/platform/database/exploration"
	"github.com/doveva/Gulyaem/backend/internal/platform/database/geoquery"
	routeanalysisdb "github.com/doveva/Gulyaem/backend/internal/platform/database/routeanalysis"
)

const defaultCityID = "01900000-0000-7000-8000-000000000001"

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	actorID := flag.String("actor", cfg.DevelopmentActorID, "actor UUID")
	cityID := flag.String("city", defaultCityID, "city UUID")
	timeout := flag.Duration("timeout", 30*time.Minute, "rebuild timeout")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	db, err := database.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer db.Close()
	geo := geoquery.New(db)
	analyzer := routeanalysis.NewAnalyzer(routeanalysisdb.New(db, geo))
	repository := explorationdb.New(db)
	result, err := exploration.NewRebuilder(analyzer, repository).Rebuild(ctx, *actorID, *cityID)
	if err != nil {
		return err
	}
	fmt.Printf("geo_version=%s walks=%d segments=%d duration=%s\n", result.GeoDataVersionID, result.WalksProcessed, result.SegmentsPublished, result.Duration)
	return nil
}
