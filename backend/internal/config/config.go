package config

import (
	"errors"
	"os"
	"strings"
	"time"
)

type Config struct {
	DatabaseURL        string
	HTTPAddress        string
	Environment        string
	GeoDataPath        string
	GeoTestArea        string
	DistrictTestArea   string
	LogLevel           string
	CORSAllowedOrigins []string
	ShutdownTimeout    time.Duration
}

func Load() (Config, error) {
	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if databaseURL == "" {
		return Config{}, errors.New("DATABASE_URL is required")
	}

	return Config{
		DatabaseURL:        databaseURL,
		HTTPAddress:        envOrDefault("HTTP_ADDRESS", ":8080"),
		Environment:        envOrDefault("ENVIRONMENT", "development"),
		GeoDataPath:        envOrDefault("GEO_DATA_PATH", "./data"),
		GeoTestArea:        strings.TrimSpace(os.Getenv("GEO_TEST_AREA")),
		DistrictTestArea:   strings.TrimSpace(os.Getenv("DISTRICT_TEST_AREA")),
		LogLevel:           strings.ToLower(envOrDefault("LOG_LEVEL", "info")),
		CORSAllowedOrigins: splitCSV(envOrDefault("CORS_ALLOWED_ORIGINS", "http://localhost:5173,http://localhost:3000")),
		ShutdownTimeout:    10 * time.Second,
	}, nil
}

func envOrDefault(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if value := strings.TrimSpace(part); value != "" {
			result = append(result, value)
		}
	}
	return result
}
