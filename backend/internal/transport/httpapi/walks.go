package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/doveva/Gulyaem/backend/internal/exploration"
	"github.com/doveva/Gulyaem/backend/internal/geo/routeanalysis"
	"github.com/doveva/Gulyaem/backend/internal/routing/port"
	"github.com/doveva/Gulyaem/backend/internal/routing/preview"
	"github.com/doveva/Gulyaem/backend/internal/walks"
)

func registerWalkRoutes(router *http.ServeMux, deps Dependencies) {
	if deps.Walks == nil || deps.Actor == nil {
		return
	}
	router.Handle("POST /api/v1/walks", createWalkHandler(deps))
	router.Handle("GET /api/v1/walks/{walkId}", getWalkHandler(deps))
	router.Handle("POST /api/v1/walks/{walkId}/start", walkActionHandler(deps, "start"))
	router.Handle("POST /api/v1/walks/{walkId}/finish", walkActionHandler(deps, "finish"))
	router.Handle("POST /api/v1/walks/{walkId}/cancel", walkActionHandler(deps, "cancel"))
	router.Handle("PUT /api/v1/walks/{walkId}/route", correctWalkRouteHandler(deps))
	if deps.Exploration != nil {
		router.Handle("POST /api/v1/walks/{walkId}/complete", completeWalkHandler(deps))
	}
}

func createWalkHandler(deps Dependencies) http.HandlerFunc {
	return func(response http.ResponseWriter, request *http.Request) {
		actor, ok := resolveActor(response, request, deps)
		if !ok {
			return
		}
		var body walks.CreateRequest
		if !decodeStrict(response, request, &body) {
			return
		}
		result, created, err := deps.Walks.Create(request.Context(), actor, body)
		if err != nil {
			handleWalkError(response, deps, err)
			return
		}
		status := http.StatusOK
		if created {
			status = http.StatusCreated
		}
		writeJSON(response, status, result)
	}
}
func getWalkHandler(deps Dependencies) http.HandlerFunc {
	return func(response http.ResponseWriter, request *http.Request) {
		actor, ok := resolveActor(response, request, deps)
		if !ok {
			return
		}
		result, err := deps.Walks.Get(request.Context(), actor, request.PathValue("walkId"))
		if err != nil {
			handleWalkError(response, deps, err)
			return
		}
		writeJSON(response, http.StatusOK, result)
	}
}
func walkActionHandler(deps Dependencies, action string) http.HandlerFunc {
	return func(response http.ResponseWriter, request *http.Request) {
		actor, ok := resolveActor(response, request, deps)
		if !ok {
			return
		}
		var result walks.Aggregate
		var err error
		switch action {
		case "start":
			result, err = deps.Walks.Start(request.Context(), actor, request.PathValue("walkId"))
		case "finish":
			result, err = deps.Walks.Finish(request.Context(), actor, request.PathValue("walkId"))
		default:
			result, err = deps.Walks.Cancel(request.Context(), actor, request.PathValue("walkId"))
		}
		if err != nil {
			handleWalkError(response, deps, err)
			return
		}
		writeJSON(response, http.StatusOK, result)
	}
}
func correctWalkRouteHandler(deps Dependencies) http.HandlerFunc {
	return func(response http.ResponseWriter, request *http.Request) {
		actor, ok := resolveActor(response, request, deps)
		if !ok {
			return
		}
		var body walks.CorrectRouteRequest
		if !decodeStrict(response, request, &body) {
			return
		}
		result, err := deps.Walks.CorrectRoute(request.Context(), actor, request.PathValue("walkId"), body)
		if err != nil {
			handleWalkError(response, deps, err)
			return
		}
		writeJSON(response, http.StatusOK, result)
	}
}
func completeWalkHandler(deps Dependencies) http.HandlerFunc {
	return func(response http.ResponseWriter, request *http.Request) {
		actor, ok := resolveActor(response, request, deps)
		if !ok {
			return
		}
		result, err := deps.Exploration.Complete(request.Context(), actor, request.PathValue("walkId"))
		if err != nil {
			handleWalkError(response, deps, err)
			return
		}
		writeJSON(response, http.StatusOK, result)
	}
}

func resolveActor(response http.ResponseWriter, request *http.Request, deps Dependencies) (string, bool) {
	id, err := deps.Actor.ActorID(request.Context())
	if err != nil {
		deps.Logger.Error("resolve actor", "error", err)
		writeRoutePreviewError(response, http.StatusServiceUnavailable, "actor_unavailable", "actor context is unavailable")
		return "", false
	}
	return id, true
}
func decodeStrict(response http.ResponseWriter, request *http.Request, target any) bool {
	request.Body = http.MaxBytesReader(response, request.Body, 1<<20)
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		writeRoutePreviewError(response, http.StatusBadRequest, "invalid_body", "request body must be valid JSON")
		return false
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		writeRoutePreviewError(response, http.StatusBadRequest, "invalid_body", "request body must contain one JSON object")
		return false
	}
	return true
}
func handleWalkError(response http.ResponseWriter, deps Dependencies, err error) {
	switch {
	case errors.Is(err, walks.ErrInvalidRequest):
		writeRoutePreviewError(response, http.StatusUnprocessableEntity, "invalid_request", err.Error())
	case errors.Is(err, walks.ErrNotFound):
		writeRoutePreviewError(response, http.StatusNotFound, "not_found", "walk not found")
	case errors.Is(err, walks.ErrPreviewStale):
		writeRoutePreviewError(response, http.StatusConflict, "route_preview_stale", "route preview changed and must be reviewed again")
	case errors.Is(err, walks.ErrIdempotencyConflict):
		writeRoutePreviewError(response, http.StatusConflict, "client_request_conflict", "clientRequestId was already used with different input")
	case errors.Is(err, walks.ErrRouteNotEditable):
		writeRoutePreviewError(response, http.StatusConflict, "walk_route_not_editable", "walk route cannot be edited in its current state")
	case errors.Is(err, walks.ErrInvalidState), errors.Is(err, walks.ErrConcurrentChange):
		writeRoutePreviewError(response, http.StatusConflict, "walk_invalid_state", "walk is not in the required state")
	case errors.Is(err, exploration.ErrRouteGeoVersionStale):
		writeRoutePreviewError(response, http.StatusConflict, "walk_route_geo_version_stale", "final route must be refreshed against current geo data before completion")
	case errors.Is(err, exploration.ErrRebuildRequired):
		writeRoutePreviewError(response, http.StatusConflict, "exploration_rebuild_required", "personal exploration state must be rebuilt for current geo data")
	case errors.Is(err, preview.ErrInvalidRequest), errors.Is(err, preview.ErrDatasetMismatch), errors.Is(err, preview.ErrGeoUnavailable):
		handleRoutePreviewError(response, deps, err)
	case errors.Is(err, port.ErrRouteNotFound), errors.Is(err, port.ErrTimeout), errors.Is(err, port.ErrUnavailable),
		errors.Is(err, port.ErrInvalidResponse), errors.Is(err, routeanalysis.ErrInvalidParameters):
		handleRoutePreviewError(response, deps, err)
	default:
		deps.Logger.Error("walk API request failed", "error", err)
		writeRoutePreviewError(response, http.StatusInternalServerError, "internal_error", "internal server error")
	}
}
