package valhalla

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/doveva/Gulyaem/backend/internal/routing/port"
)

const maxResponseBytes = 16 << 20

type Client struct {
	baseURL        string
	httpClient     *http.Client
	metadataSource MetadataSource
}

type MetadataSource interface {
	Load(context.Context) (port.Metadata, error)
}

func New(baseURL string, timeout time.Duration, metadataSource MetadataSource) *Client {
	return &Client{
		baseURL:        strings.TrimRight(baseURL, "/"),
		httpClient:     &http.Client{Timeout: timeout},
		metadataSource: metadataSource,
	}
}

func (client *Client) Metadata(ctx context.Context) (port.Metadata, error) {
	if client.metadataSource == nil {
		return port.Metadata{}, fmt.Errorf("%w: graph metadata source is not configured", port.ErrInvalidResponse)
	}
	metadata, err := client.metadataSource.Load(ctx)
	if err != nil {
		return port.Metadata{}, err
	}
	if metadata.Engine != "valhalla" || metadata.EngineVersion == "" || metadata.CityID == "" ||
		metadata.SourceChecksum == "" || metadata.Profile != "pedestrian" ||
		metadata.GraphArtifact == "" || metadata.GraphChecksum == "" {
		return port.Metadata{}, fmt.Errorf("%w: incomplete graph metadata", port.ErrInvalidResponse)
	}
	return metadata, nil
}

func (client *Client) Ping(ctx context.Context) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, client.baseURL+"/status", nil)
	if err != nil {
		return err
	}
	response, err := client.httpClient.Do(request)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return port.ErrTimeout
		}
		return fmt.Errorf("%w: %v", port.ErrUnavailable, err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("%w: status %d", port.ErrUnavailable, response.StatusCode)
	}
	return nil
}

func (client *Client) Route(ctx context.Context, request port.RouteRequest) (port.RouteResult, error) {
	body := struct {
		Locations   []port.Point `json:"locations"`
		Costing     string       `json:"costing"`
		Units       string       `json:"units"`
		ShapeFormat string       `json:"shape_format"`
	}{Locations: request.Waypoints, Costing: request.Profile, Units: "kilometers", ShapeFormat: "geojson"}
	encoded, err := json.Marshal(body)
	if err != nil {
		return port.RouteResult{}, fmt.Errorf("encode Valhalla route request: %w", err)
	}
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, client.baseURL+"/route", bytes.NewReader(encoded))
	if err != nil {
		return port.RouteResult{}, fmt.Errorf("create Valhalla route request: %w", err)
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	response, err := client.httpClient.Do(httpRequest)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) && ctx.Err() == context.DeadlineExceeded {
			return port.RouteResult{}, port.ErrTimeout
		}
		return port.RouteResult{}, fmt.Errorf("%w: %v", port.ErrUnavailable, err)
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusBadRequest || response.StatusCode == http.StatusNotFound {
		return port.RouteResult{}, port.ErrRouteNotFound
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return port.RouteResult{}, fmt.Errorf("%w: status %d", port.ErrUnavailable, response.StatusCode)
	}
	limited := io.LimitReader(response.Body, maxResponseBytes+1)
	payload, err := io.ReadAll(limited)
	if err != nil {
		return port.RouteResult{}, fmt.Errorf("%w: read response", port.ErrInvalidResponse)
	}
	if len(payload) > maxResponseBytes {
		return port.RouteResult{}, fmt.Errorf("%w: response is too large", port.ErrInvalidResponse)
	}
	return decodeRoute(payload, request.Waypoints)
}

type valhallaResponse struct {
	Trip struct {
		Summary struct {
			Length float64 `json:"length"`
			Time   float64 `json:"time"`
		} `json:"summary"`
		Locations []struct {
			Lat float64 `json:"lat"`
			Lon float64 `json:"lon"`
		} `json:"locations"`
		Legs []struct {
			Shape json.RawMessage `json:"shape"`
		} `json:"legs"`
	} `json:"trip"`
}

func decodeRoute(payload []byte, inputs []port.Point) (port.RouteResult, error) {
	var response valhallaResponse
	if err := json.Unmarshal(payload, &response); err != nil {
		return port.RouteResult{}, fmt.Errorf("%w: malformed JSON", port.ErrInvalidResponse)
	}
	if !finiteNonNegative(response.Trip.Summary.Length) || !finiteNonNegative(response.Trip.Summary.Time) {
		return port.RouteResult{}, fmt.Errorf("%w: invalid summary", port.ErrInvalidResponse)
	}
	coordinates := make([][2]float64, 0)
	for _, leg := range response.Trip.Legs {
		legCoordinates, err := decodeShape(leg.Shape)
		if err != nil {
			return port.RouteResult{}, err
		}
		if len(coordinates) > 0 && len(legCoordinates) > 0 && coordinates[len(coordinates)-1] == legCoordinates[0] {
			legCoordinates = legCoordinates[1:]
		}
		coordinates = append(coordinates, legCoordinates...)
	}
	if len(coordinates) < 2 {
		return port.RouteResult{}, fmt.Errorf("%w: route geometry is missing", port.ErrInvalidResponse)
	}
	geometry, _ := json.Marshal(struct {
		Type        string       `json:"type"`
		Coordinates [][2]float64 `json:"coordinates"`
	}{Type: "LineString", Coordinates: coordinates})
	waypoints := make([]port.Waypoint, len(inputs))
	for index, input := range inputs {
		waypoints[index].Input = input
		if index < len(response.Trip.Locations) {
			resolved := port.Point{Lat: response.Trip.Locations[index].Lat, Lon: response.Trip.Locations[index].Lon}
			if validPoint(resolved) {
				waypoints[index].Resolved = &resolved
			}
		}
	}
	return port.RouteResult{
		DistanceMeters:  response.Trip.Summary.Length * 1000,
		DurationSeconds: response.Trip.Summary.Time,
		Geometry:        geometry,
		Waypoints:       waypoints,
	}, nil
}

func decodeShape(raw json.RawMessage) ([][2]float64, error) {
	trimmed := bytes.TrimSpace(raw)
	var geoJSON struct {
		Type        string       `json:"type"`
		Coordinates [][2]float64 `json:"coordinates"`
	}
	if err := json.Unmarshal(trimmed, &geoJSON); err == nil && geoJSON.Type == "LineString" {
		if validCoordinates(geoJSON.Coordinates) {
			return geoJSON.Coordinates, nil
		}
	}
	var coordinates [][2]float64
	if err := json.Unmarshal(trimmed, &coordinates); err == nil && validCoordinates(coordinates) {
		return coordinates, nil
	}
	var encoded string
	if err := json.Unmarshal(trimmed, &encoded); err == nil {
		coordinates, err := decodePolyline(encoded, 6)
		if err == nil && validCoordinates(coordinates) {
			return coordinates, nil
		}
	}
	return nil, fmt.Errorf("%w: unsupported Valhalla shape", port.ErrInvalidResponse)
}

func decodePolyline(encoded string, precision int) ([][2]float64, error) {
	factor := 1.0
	for range precision {
		factor *= 10
	}
	latitude, longitude := int64(0), int64(0)
	result := make([][2]float64, 0)
	for index := 0; index < len(encoded); {
		values := [2]int64{}
		for valueIndex := range values {
			shift, value := uint(0), int64(0)
			for {
				if index >= len(encoded) {
					return nil, errors.New("truncated encoded polyline")
				}
				part := int64(encoded[index]) - 63
				index++
				value |= (part & 0x1f) << shift
				shift += 5
				if part < 0x20 {
					break
				}
			}
			if value&1 != 0 {
				values[valueIndex] = ^(value >> 1)
			} else {
				values[valueIndex] = value >> 1
			}
		}
		latitude += values[0]
		longitude += values[1]
		result = append(result, [2]float64{float64(longitude) / factor, float64(latitude) / factor})
	}
	return result, nil
}

func validCoordinates(coordinates [][2]float64) bool {
	if len(coordinates) < 2 {
		return false
	}
	for _, coordinate := range coordinates {
		if !validPoint(port.Point{Lon: coordinate[0], Lat: coordinate[1]}) {
			return false
		}
	}
	return true
}

func validPoint(point port.Point) bool {
	return point.Lat >= -90 && point.Lat <= 90 && point.Lon >= -180 && point.Lon <= 180 &&
		!math.IsNaN(point.Lat) && !math.IsNaN(point.Lon) && !math.IsInf(point.Lat, 0) && !math.IsInf(point.Lon, 0)
}

func finiteNonNegative(value float64) bool {
	return value >= 0 && !math.IsNaN(value) && !math.IsInf(value, 0)
}

var _ port.RoutingEngine = (*Client)(nil)
