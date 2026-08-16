package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"slices"
	"time"

	"github.com/Asksel-Ecosystem/askcel-go/health"
	"github.com/doveva/Gulyaem/backend/internal/exploration"
	"github.com/doveva/Gulyaem/backend/internal/geo/querying"
	"github.com/doveva/Gulyaem/backend/internal/geo/routeanalysis"
	"github.com/doveva/Gulyaem/backend/internal/routing/preview"
	"github.com/doveva/Gulyaem/backend/internal/walks"
	"github.com/go-chi/chi/v5/middleware"
)

type HealthChecker interface {
	Ping(context.Context) error
}

type ReadinessChecker interface {
	Ready(context.Context) error
}

type ActorResolver interface {
	ActorID(context.Context) (string, error)
}
type StaticActorResolver struct{ ID string }

func (r StaticActorResolver) ActorID(context.Context) (string, error) {
	if r.ID == "" {
		return "", errors.New("development actor is not configured")
	}
	return r.ID, nil
}

type Dependencies struct {
	Database       HealthChecker
	Logger         *slog.Logger
	Environment    string
	AllowedOrigins []string
	Geo            *querying.Service
	RouteAnalysis  *routeanalysis.Service
	RoutePreview   *preview.Service
	Routing        HealthChecker
	RoutingDataset ReadinessChecker
	Actor          ActorResolver
	Walks          *walks.Service
	Exploration    *exploration.Service
}

type Handler struct {
	http.Handler
	health *health.Registry
}

func NewHandler(deps Dependencies) *Handler {
	mux := http.NewServeMux()

	checks := health.New(
		health.WithTimeout(2*time.Second),
		health.WithObserver(func(event health.Event) {
			if event.Err != nil && deps.Logger != nil {
				deps.Logger.Error("health check failed",
					"class", event.Class,
					"check", event.Name,
					"timed_out", event.TimedOut,
					"duration_ms", event.Duration.Milliseconds(),
					"error", event.Err,
				)
			}
		}),
	)
	mustRegisterHealthCheck(checks.Liveness("startup", func(context.Context) error { return nil }))
	if deps.Database != nil {
		mustRegisterHealthCheck(checks.Readiness("database", deps.Database.Ping))
	}
	if deps.Routing != nil {
		mustRegisterHealthCheck(checks.Readiness("routing", deps.Routing.Ping))
	}
	if deps.RoutingDataset != nil {
		mustRegisterHealthCheck(checks.Readiness("routing-dataset", deps.RoutingDataset.Ready))
	}

	mux.Handle("GET /health/live", checks.LiveHandler())
	mux.Handle("GET /health/ready", checks.ReadyHandler())
	registerGeoRoutes(mux, deps)
	registerRouteAnalysisRoutes(mux, deps)
	registerRoutePreviewRoutes(mux, deps)
	registerWalkRoutes(mux, deps)
	registerExplorationRoutes(mux, deps)

	var handler http.Handler = mux
	handler = cors(deps.AllowedOrigins)(handler)
	handler = middleware.Compress(5, "application/json")(handler)
	handler = requestLogger(deps.Logger)(handler)
	handler = middleware.Recoverer(handler)
	handler = middleware.ClientIPFromRemoteAddr(handler)
	handler = middleware.RequestID(handler)
	return &Handler{Handler: handler, health: checks}
}

// Drain removes the API from readiness while allowing in-flight requests to
// finish and leaving liveness untouched.
func (h *Handler) Drain() {
	h.health.Drain()
}

func mustRegisterHealthCheck(err error) {
	if err != nil {
		panic(err)
	}
}

func writeJSON(response http.ResponseWriter, status int, body any) {
	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(status)
	_ = json.NewEncoder(response).Encode(body)
}

func cors(allowedOrigins []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			origin := request.Header.Get("Origin")
			if origin != "" && slices.Contains(allowedOrigins, origin) {
				response.Header().Set("Access-Control-Allow-Origin", origin)
				response.Header().Set("Vary", "Origin")
				response.Header().Set("Access-Control-Allow-Headers", "Content-Type")
				response.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,OPTIONS")
			}
			if request.Method == http.MethodOptions {
				response.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(response, request)
		})
	}
}

type responseRecorder struct {
	http.ResponseWriter
	status int
}

func (recorder *responseRecorder) WriteHeader(status int) {
	recorder.status = status
	recorder.ResponseWriter.WriteHeader(status)
}

func requestLogger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			started := time.Now()
			recorder := &responseRecorder{ResponseWriter: response, status: http.StatusOK}
			next.ServeHTTP(recorder, request)
			logger.InfoContext(request.Context(), "http request",
				"method", request.Method,
				"route", request.Pattern,
				"status", recorder.status,
				"duration_ms", time.Since(started).Milliseconds(),
				"request_id", middleware.GetReqID(request.Context()),
			)
		})
	}
}
