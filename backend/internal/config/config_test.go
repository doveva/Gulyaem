package config

import "testing"

func TestLoadRequiresDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	if _, err := Load(); err == nil {
		t.Fatal("Load() expected an error without DATABASE_URL")
	}
}

func TestLoadAppliesDefaults(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://example")
	t.Setenv("HTTP_ADDRESS", "")
	t.Setenv("CORS_ALLOWED_ORIGINS", "http://localhost:5173, http://localhost:3000")
	t.Setenv("ROUTING_DATASET_METADATA_PATH", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.HTTPAddress != ":8080" {
		t.Fatalf("HTTPAddress = %q, want :8080", cfg.HTTPAddress)
	}
	if len(cfg.CORSAllowedOrigins) != 2 {
		t.Fatalf("CORSAllowedOrigins = %v", cfg.CORSAllowedOrigins)
	}
	if cfg.RoutingDatasetMetadataPath != "../.routing/valhalla/routing-dataset.json" {
		t.Fatalf("RoutingDatasetMetadataPath = %q", cfg.RoutingDatasetMetadataPath)
	}
}
