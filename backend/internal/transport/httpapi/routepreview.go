package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/doveva/Gulyaem/backend/internal/geo/routeanalysis"
	"github.com/doveva/Gulyaem/backend/internal/routing/port"
	"github.com/doveva/Gulyaem/backend/internal/routing/preview"
)

func registerRoutePreviewRoutes(router *http.ServeMux, deps Dependencies) {
	if deps.RoutePreview == nil {
		return
	}
	router.Handle("POST /api/v1/route-previews", createRoutePreviewHandler(deps))
}

func createRoutePreviewHandler(deps Dependencies) http.HandlerFunc {
	return func(response http.ResponseWriter, request *http.Request) {
		request.Body = http.MaxBytesReader(response, request.Body, 1<<20)
		var body preview.Request
		decoder := json.NewDecoder(request.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&body); err != nil {
			writeRoutePreviewError(response, http.StatusBadRequest, "invalid_body", "request body must be valid JSON")
			return
		}
		var trailing any
		if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
			writeRoutePreviewError(response, http.StatusBadRequest, "invalid_body", "request body must contain one JSON object")
			return
		}
		if !uuidPattern.MatchString(body.CityID) {
			writeRoutePreviewError(response, http.StatusUnprocessableEntity, "invalid_waypoints", "cityId must be a UUID")
			return
		}
		result, err := deps.RoutePreview.Create(request.Context(), body)
		if err != nil {
			handleRoutePreviewError(response, deps, err)
			return
		}
		writeJSON(response, http.StatusOK, result)
	}
}

func handleRoutePreviewError(response http.ResponseWriter, deps Dependencies, err error) {
	switch {
	case errors.Is(err, preview.ErrInvalidRequest), errors.Is(err, routeanalysis.ErrInvalidParameters):
		message := strings.TrimPrefix(err.Error(), preview.ErrInvalidRequest.Error()+": ")
		writeRoutePreviewError(response, http.StatusUnprocessableEntity, "invalid_waypoints", message)
	case errors.Is(err, port.ErrRouteNotFound):
		writeRoutePreviewError(response, http.StatusUnprocessableEntity, "route_not_found", "pedestrian route could not be built")
	case errors.Is(err, preview.ErrDatasetMismatch):
		writeRoutePreviewError(response, http.StatusConflict, "routing_geo_version_mismatch", "routing graph and current geo data version use different source datasets")
	case errors.Is(err, preview.ErrGeoUnavailable):
		writeRoutePreviewError(response, http.StatusServiceUnavailable, "geo_data_unavailable", "geo data is unavailable")
	case errors.Is(err, port.ErrTimeout):
		writeRoutePreviewError(response, http.StatusGatewayTimeout, "routing_timeout", "routing service did not respond in time")
	case errors.Is(err, port.ErrUnavailable), errors.Is(err, port.ErrInvalidResponse):
		writeRoutePreviewError(response, http.StatusServiceUnavailable, "routing_unavailable", "routing service is unavailable")
	default:
		deps.Logger.Error("route preview API request failed", "error", err)
		writeRoutePreviewError(response, http.StatusInternalServerError, "internal_error", "internal server error")
	}
}

func writeRoutePreviewError(response http.ResponseWriter, status int, code, message string) {
	writeJSON(response, status, apiError{Code: code, Message: message})
}
