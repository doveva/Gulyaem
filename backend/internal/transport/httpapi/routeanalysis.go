package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"math"
	"net/http"
	"strings"

	"github.com/doveva/Gulyaem/backend/internal/geo/querying"
	"github.com/doveva/Gulyaem/backend/internal/geo/routeanalysis"
	"github.com/go-chi/chi/v5"
)

type analyzeRouteRequest struct {
	Matching *routeanalysis.MatchingParameters `json:"matching"`
	Coverage struct {
		Profile           string   `json:"profile"`
		RadiusMeters      *float64 `json:"radiusMeters"`
		CoverageRatio     *float64 `json:"coverageRatio"`
		MinRequiredMeters *float64 `json:"minRequiredMeters"`
		MaxRequiredMeters *float64 `json:"maxRequiredMeters"`
	} `json:"coverage"`
}

func registerRouteAnalysisRoutes(router chi.Router, deps Dependencies) {
	if deps.RouteAnalysis == nil {
		return
	}
	router.Get("/api/v1/geo/sample-routes", sampleRoutesHandler(deps))
	router.Post("/api/v1/geo/sample-routes/{routeId}/analyze", analyzeSampleRouteHandler(deps))
}

func sampleRoutesHandler(deps Dependencies) http.HandlerFunc {
	return func(response http.ResponseWriter, request *http.Request) {
		cityID := strings.TrimSpace(request.URL.Query().Get("cityId"))
		if !uuidPattern.MatchString(cityID) {
			writeAPIError(response, http.StatusBadRequest, "invalid_query", "cityId must be a UUID")
			return
		}
		collection, err := deps.RouteAnalysis.Routes(request.Context(), cityID)
		if err != nil {
			handleRouteAnalysisError(response, deps, err)
			return
		}
		writeJSON(response, http.StatusOK, collection)
	}
}

func analyzeSampleRouteHandler(deps Dependencies) http.HandlerFunc {
	return func(response http.ResponseWriter, request *http.Request) {
		cityID := strings.TrimSpace(request.URL.Query().Get("cityId"))
		if !uuidPattern.MatchString(cityID) {
			writeAPIError(response, http.StatusBadRequest, "invalid_query", "cityId must be a UUID")
			return
		}
		routeID := chi.URLParam(request, "routeId")
		if !routeIDPattern.MatchString(routeID) {
			writeAPIError(response, http.StatusBadRequest, "invalid_route_id", "routeId is invalid")
			return
		}
		request.Body = http.MaxBytesReader(response, request.Body, 1<<20)
		var body analyzeRouteRequest
		decoder := json.NewDecoder(request.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&body); err != nil && !errors.Is(err, io.EOF) {
			writeAPIError(response, http.StatusBadRequest, "invalid_body", "request body must be valid JSON")
			return
		}
		parameters, err := resolveAnalyzeRequest(body)
		if err != nil {
			writeAPIError(response, http.StatusBadRequest, "invalid_body", err.Error())
			return
		}
		analysis, err := deps.RouteAnalysis.Analyze(request.Context(), cityID, routeID, parameters)
		if err != nil {
			handleRouteAnalysisError(response, deps, err)
			return
		}
		writeJSON(response, http.StatusOK, analysis)
	}
}

func resolveAnalyzeRequest(body analyzeRouteRequest) (routeanalysis.AnalyzeRequest, error) {
	matching := routeanalysis.DefaultMatchingParameters()
	if body.Matching != nil {
		matching = *body.Matching
	}
	profileName := strings.ToLower(strings.TrimSpace(body.Coverage.Profile))
	if profileName == "" {
		profileName = "balanced"
	}
	profile, preset := routeanalysis.CoverageProfiles[profileName]
	if !preset && profileName != "custom" {
		return routeanalysis.AnalyzeRequest{}, errors.New("coverage.profile must be strict, balanced, generous or custom")
	}
	if profileName == "custom" {
		if body.Coverage.RadiusMeters == nil || body.Coverage.CoverageRatio == nil ||
			body.Coverage.MinRequiredMeters == nil || body.Coverage.MaxRequiredMeters == nil {
			return routeanalysis.AnalyzeRequest{}, errors.New("custom coverage requires radius, ratio, minimum and maximum")
		}
		profile = routeanalysis.CoverageProfile{
			Name: "custom", RadiusMeters: *body.Coverage.RadiusMeters,
			CoverageRatio: *body.Coverage.CoverageRatio, MinRequiredMeters: *body.Coverage.MinRequiredMeters,
			MaxRequiredMeters: *body.Coverage.MaxRequiredMeters,
		}
	} else if body.Coverage.RadiusMeters != nil || body.Coverage.CoverageRatio != nil ||
		body.Coverage.MinRequiredMeters != nil || body.Coverage.MaxRequiredMeters != nil {
		return routeanalysis.AnalyzeRequest{}, errors.New("preset coverage values cannot be overridden; use custom")
	}
	for _, value := range []float64{
		matching.SampleStepMeters, matching.CandidateRadiusMeters, matching.MaxDirectionDegrees,
		matching.EndpointToleranceMeters, profile.RadiusMeters, profile.CoverageRatio,
		profile.MinRequiredMeters, profile.MaxRequiredMeters,
	} {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return routeanalysis.AnalyzeRequest{}, errors.New("analysis parameters must be finite numbers")
		}
	}
	return routeanalysis.AnalyzeRequest{Matching: matching, Coverage: profile}, nil
}

func handleRouteAnalysisError(response http.ResponseWriter, deps Dependencies, err error) {
	switch {
	case errors.Is(err, routeanalysis.ErrRouteNotFound):
		writeAPIError(response, http.StatusNotFound, "route_not_found", "sample route was not found")
	case errors.Is(err, routeanalysis.ErrInvalidParameters):
		writeAPIError(response, http.StatusUnprocessableEntity, "invalid_analysis_parameters", err.Error())
	case errors.Is(err, querying.ErrNotFound):
		writeAPIError(response, http.StatusNotFound, "not_found", "current geo data version was not found")
	default:
		deps.Logger.Error("route analysis API request failed", "error", err)
		writeAPIError(response, http.StatusInternalServerError, "internal_error", "internal server error")
	}
}
