package httpapi

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type healthCheckerStub struct {
	err error
}

func (stub healthCheckerStub) Ping(context.Context) error { return stub.err }

type readinessCheckerStub struct{ err error }

func (stub readinessCheckerStub) Ready(context.Context) error { return stub.err }

func TestReadiness(t *testing.T) {
	tests := []struct {
		name       string
		database   HealthChecker
		wantStatus int
	}{
		{name: "ready", database: healthCheckerStub{}, wantStatus: http.StatusOK},
		{name: "database unavailable", database: healthCheckerStub{err: errors.New("down")}, wantStatus: http.StatusServiceUnavailable},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler := NewHandler(Dependencies{
				Database:    test.database,
				Logger:      slog.New(slog.NewTextHandler(io.Discard, nil)),
				Environment: "test",
			})
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/health/ready", nil))
			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d", response.Code, test.wantStatus)
			}
		})
	}
}

func TestReadinessIncludesRoutingWhenConfigured(t *testing.T) {
	handler := NewHandler(Dependencies{
		Database: healthCheckerStub{}, Routing: healthCheckerStub{err: errors.New("down")},
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), Environment: "test",
	})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/health/ready", nil))
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusServiceUnavailable)
	}
}

func TestReadinessIncludesRoutingDatasetCompatibility(t *testing.T) {
	handler := NewHandler(Dependencies{
		Database: healthCheckerStub{}, Routing: healthCheckerStub{},
		RoutingDataset: readinessCheckerStub{err: errors.New("mismatch")},
		Logger:         slog.New(slog.NewTextHandler(io.Discard, nil)), Environment: "test",
	})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/health/ready", nil))
	if response.Code != http.StatusServiceUnavailable || !strings.Contains(response.Body.String(), "routing dataset incompatible") {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
}

func TestCORSAllowsConfiguredOrigin(t *testing.T) {
	handler := NewHandler(Dependencies{
		Database:       healthCheckerStub{},
		Logger:         slog.New(slog.NewTextHandler(io.Discard, nil)),
		Environment:    "test",
		AllowedOrigins: []string{"http://localhost:5173"},
	})
	request := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	request.Header.Set("Origin", "http://localhost:5173")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if got := response.Header().Get("Access-Control-Allow-Origin"); !strings.EqualFold(got, "http://localhost:5173") {
		t.Fatalf("Access-Control-Allow-Origin = %q", got)
	}
}
