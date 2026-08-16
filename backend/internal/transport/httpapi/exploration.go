package httpapi

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/doveva/Gulyaem/backend/internal/exploration"
	"github.com/doveva/Gulyaem/backend/internal/geo/querying"
)

func registerExplorationRoutes(router *http.ServeMux, deps Dependencies) {
	if deps.Exploration == nil || deps.Actor == nil {
		return
	}
	router.Handle("GET /api/v1/cities/{cityId}/exploration", cityExplorationHandler(deps))
	router.Handle("GET /api/v1/cities/{cityId}/exploration/segments", exploredSegmentsHandler(deps))
}
func cityExplorationHandler(deps Dependencies) http.HandlerFunc {
	return func(response http.ResponseWriter, request *http.Request) {
		actor, ok := resolveActor(response, request, deps)
		if !ok {
			return
		}
		result, err := deps.Exploration.City(request.Context(), actor, request.PathValue("cityId"))
		if err != nil {
			handleExplorationError(response, deps, err)
			return
		}
		writeJSON(response, http.StatusOK, result)
	}
}
func exploredSegmentsHandler(deps Dependencies) http.HandlerFunc {
	return func(response http.ResponseWriter, request *http.Request) {
		actor, ok := resolveActor(response, request, deps)
		if !ok {
			return
		}
		bbox, err := parseExplorationBBox(request.URL.Query().Get("bbox"))
		if err != nil {
			writeRoutePreviewError(response, http.StatusBadRequest, "invalid_query", "bbox must contain west,south,east,north")
			return
		}
		result, err := deps.Exploration.Segments(request.Context(), actor, request.PathValue("cityId"), bbox)
		if err != nil {
			handleExplorationError(response, deps, err)
			return
		}
		response.Header().Set("Content-Type", "application/geo+json")
		response.WriteHeader(http.StatusOK)
		_, _ = response.Write(result)
	}
}
func parseExplorationBBox(value string) ([4]float64, error) {
	var result [4]float64
	parts := strings.Split(value, ",")
	if len(parts) != 4 {
		return result, errors.New("invalid bbox")
	}
	for i, p := range parts {
		number, err := strconv.ParseFloat(strings.TrimSpace(p), 64)
		if err != nil {
			return result, err
		}
		result[i] = number
	}
	return result, nil
}
func handleExplorationError(response http.ResponseWriter, deps Dependencies, err error) {
	switch {
	case errors.Is(err, exploration.ErrRebuildRequired):
		writeRoutePreviewError(response, http.StatusConflict, "exploration_rebuild_required", "personal exploration state must be rebuilt for current geo data")
	case errors.Is(err, exploration.ErrInvalidBBox):
		writeRoutePreviewError(response, http.StatusBadRequest, "invalid_query", "bbox is outside supported limits")
	case errors.Is(err, querying.ErrFeatureLimit):
		writeRoutePreviewError(response, http.StatusUnprocessableEntity, "feature_limit_exceeded", "viewport contains more than 10000 explored segments; zoom in")
	default:
		handleWalkError(response, deps, err)
	}
}
