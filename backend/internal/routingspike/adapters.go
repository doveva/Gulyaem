package routingspike

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type engineClient interface {
	Metadata() Engine
	Route(context.Context, []Point) (rawRouteResponse, error)
	MapMatch(context.Context, []Point) (rawRouteResponse, error)
}

func clientsFromEnvironment() []engineClient {
	return []engineClient{
		&valhallaClient{baseClient: newBaseClient(engineURL("VALHALLA_URL", "http://localhost:8002"))},
		&graphHopperClient{baseClient: newBaseClient(engineURL("GRAPHHOPPER_URL", "http://localhost:8989"))},
		&osrmClient{baseClient: newBaseClient(engineURL("OSRM_URL", "http://localhost:5001"))},
	}
}

func engineURL(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return strings.TrimRight(value, "/")
	}
	return fallback
}

type baseClient struct {
	baseURL string
	http    *http.Client
}

func newBaseClient(baseURL string) baseClient {
	return baseClient{baseURL: baseURL, http: &http.Client{Timeout: 45 * time.Second}}
}

func (client baseClient) request(ctx context.Context, method, path, contentType string, body []byte) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, method, client.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	if contentType != "" {
		request.Header.Set("Content-Type", contentType)
	}
	response, err := client.http.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, 16<<20))
	if err != nil {
		return nil, err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		message := strings.TrimSpace(string(data))
		if len(message) > 500 {
			message = message[:500]
		}
		return nil, fmt.Errorf("HTTP %d: %s", response.StatusCode, message)
	}
	return data, nil
}

type valhallaClient struct{ baseClient }

func (client *valhallaClient) Metadata() Engine {
	return Engine{ID: "valhalla", Name: "Valhalla", Version: "3.7.0", Profile: "pedestrian",
		BaseURL: client.baseURL, License: "MIT", MapMatchingSurface: "trace_route HTTP API"}
}

func (client *valhallaClient) Route(ctx context.Context, points []Point) (rawRouteResponse, error) {
	locations := make([]map[string]float64, len(points))
	for index, point := range points {
		locations[index] = map[string]float64{"lon": point[0], "lat": point[1]}
	}
	payload, _ := json.Marshal(map[string]any{
		"locations": locations, "costing": "pedestrian", "units": "kilometers", "shape_format": "geojson",
	})
	data, err := client.request(ctx, http.MethodPost, "/route", "application/json", payload)
	if err != nil {
		return rawRouteResponse{}, err
	}
	return parseValhalla(data)
}

func (client *valhallaClient) MapMatch(ctx context.Context, points []Point) (rawRouteResponse, error) {
	shape := make([]map[string]float64, len(points))
	for index, point := range points {
		shape[index] = map[string]float64{"lon": point[0], "lat": point[1]}
	}
	payload, _ := json.Marshal(map[string]any{
		"shape": shape, "costing": "pedestrian", "shape_match": "map_snap", "shape_format": "geojson",
	})
	data, err := client.request(ctx, http.MethodPost, "/trace_route", "application/json", payload)
	if err != nil {
		return rawRouteResponse{}, err
	}
	return parseValhalla(data)
}

func parseValhalla(data []byte) (rawRouteResponse, error) {
	var response struct {
		Trip struct {
			Summary struct {
				Length float64 `json:"length"`
				Time   float64 `json:"time"`
			} `json:"summary"`
			Legs []struct {
				Shape json.RawMessage `json:"shape"`
			} `json:"legs"`
			StatusMessage string `json:"status_message"`
		} `json:"trip"`
		Error string `json:"error"`
	}
	if err := json.Unmarshal(data, &response); err != nil {
		return rawRouteResponse{}, fmt.Errorf("decode Valhalla response: %w", err)
	}
	if response.Error != "" {
		return rawRouteResponse{}, errors.New(response.Error)
	}
	coordinates := make([]Point, 0)
	for _, leg := range response.Trip.Legs {
		points, err := decodeShape(leg.Shape, 6)
		if err != nil {
			return rawRouteResponse{}, fmt.Errorf("decode Valhalla shape: %w", err)
		}
		coordinates = appendLine(coordinates, points)
	}
	if len(coordinates) < 2 {
		return rawRouteResponse{}, errors.New("Valhalla returned no route geometry")
	}
	return rawRouteResponse{Geometry: LineString{Type: "LineString", Coordinates: coordinates},
		DistanceMeters: response.Trip.Summary.Length * 1000, DurationSeconds: response.Trip.Summary.Time,
		ResponseBytes: len(data)}, nil
}

type graphHopperClient struct{ baseClient }

func (client *graphHopperClient) Metadata() Engine {
	return Engine{ID: "graphhopper", Name: "GraphHopper", Version: "11.0", Profile: "foot",
		BaseURL: client.baseURL, License: "Apache-2.0", MapMatchingSurface: "Java module; probed at /match"}
}

func (client *graphHopperClient) Route(ctx context.Context, points []Point) (rawRouteResponse, error) {
	payload, _ := json.Marshal(map[string]any{
		"points": points, "profile": "foot", "points_encoded": false, "instructions": false, "elevation": false,
	})
	data, err := client.request(ctx, http.MethodPost, "/route", "application/json", payload)
	if err != nil {
		return rawRouteResponse{}, err
	}
	return parseGraphHopper(data)
}

func (client *graphHopperClient) MapMatch(ctx context.Context, points []Point) (rawRouteResponse, error) {
	var builder strings.Builder
	builder.WriteString(`<?xml version="1.0" encoding="UTF-8"?><gpx version="1.1" creator="gulyaem"><trk><trkseg>`)
	for _, point := range points {
		fmt.Fprintf(&builder, `<trkpt lat="%.7f" lon="%.7f"></trkpt>`, point[1], point[0])
	}
	builder.WriteString(`</trkseg></trk></gpx>`)
	data, err := client.request(ctx, http.MethodPost, "/match?profile=foot&type=json", "application/gpx+xml", []byte(builder.String()))
	if err != nil {
		return rawRouteResponse{}, err
	}
	return parseGraphHopper(data)
}

func parseGraphHopper(data []byte) (rawRouteResponse, error) {
	var response struct {
		Message string `json:"message"`
		Paths   []struct {
			Distance float64         `json:"distance"`
			Time     float64         `json:"time"`
			Points   json.RawMessage `json:"points"`
		} `json:"paths"`
	}
	if err := json.Unmarshal(data, &response); err != nil {
		return rawRouteResponse{}, fmt.Errorf("decode GraphHopper response: %w", err)
	}
	if len(response.Paths) == 0 {
		return rawRouteResponse{}, fmt.Errorf("GraphHopper returned no path: %s", response.Message)
	}
	path := response.Paths[0]
	coordinates, err := decodeShape(path.Points, 5)
	if err != nil {
		return rawRouteResponse{}, fmt.Errorf("decode GraphHopper shape: %w", err)
	}
	if len(coordinates) < 2 {
		return rawRouteResponse{}, errors.New("GraphHopper returned no route geometry")
	}
	return rawRouteResponse{Geometry: LineString{Type: "LineString", Coordinates: coordinates},
		DistanceMeters: path.Distance, DurationSeconds: path.Time / 1000, ResponseBytes: len(data)}, nil
}

type osrmClient struct{ baseClient }

func (client *osrmClient) Metadata() Engine {
	return Engine{ID: "osrm", Name: "OSRM", Version: "26.7.3", Profile: "foot.lua",
		BaseURL: client.baseURL, License: "BSD-2-Clause", MapMatchingSurface: "match HTTP API"}
}

func (client *osrmClient) Route(ctx context.Context, points []Point) (rawRouteResponse, error) {
	path := "/route/v1/foot/" + osrmCoordinates(points) + "?overview=full&geometries=geojson&steps=false"
	data, err := client.request(ctx, http.MethodGet, path, "", nil)
	if err != nil {
		return rawRouteResponse{}, err
	}
	return parseOSRM(data, "routes")
}

func (client *osrmClient) MapMatch(ctx context.Context, points []Point) (rawRouteResponse, error) {
	path := "/match/v1/foot/" + osrmCoordinates(points) + "?overview=full&geometries=geojson&steps=false&tidy=true"
	data, err := client.request(ctx, http.MethodGet, path, "", nil)
	if err != nil {
		return rawRouteResponse{}, err
	}
	return parseOSRM(data, "matchings")
}

func osrmCoordinates(points []Point) string {
	values := make([]string, len(points))
	for index, point := range points {
		values[index] = fmt.Sprintf("%.7f,%.7f", point[0], point[1])
	}
	return strings.Join(values, ";")
}

func parseOSRM(data []byte, field string) (rawRouteResponse, error) {
	var response struct {
		Code      string     `json:"code"`
		Message   string     `json:"message"`
		Routes    []osrmPath `json:"routes"`
		Matchings []osrmPath `json:"matchings"`
	}
	if err := json.Unmarshal(data, &response); err != nil {
		return rawRouteResponse{}, fmt.Errorf("decode OSRM response: %w", err)
	}
	if response.Code != "Ok" {
		return rawRouteResponse{}, fmt.Errorf("OSRM %s: %s", response.Code, response.Message)
	}
	paths := response.Routes
	if field == "matchings" {
		paths = response.Matchings
	}
	coordinates := make([]Point, 0)
	distance, duration := 0.0, 0.0
	for _, path := range paths {
		coordinates = appendLine(coordinates, path.Geometry.Coordinates)
		distance += path.Distance
		duration += path.Duration
	}
	if len(coordinates) < 2 {
		return rawRouteResponse{}, errors.New("OSRM returned no route geometry")
	}
	return rawRouteResponse{Geometry: LineString{Type: "LineString", Coordinates: coordinates},
		DistanceMeters: distance, DurationSeconds: duration, ResponseBytes: len(data)}, nil
}

type osrmPath struct {
	Distance float64    `json:"distance"`
	Duration float64    `json:"duration"`
	Geometry LineString `json:"geometry"`
}

func decodeShape(raw json.RawMessage, precision int) ([]Point, error) {
	var geometry LineString
	if len(raw) > 0 && raw[0] == '{' {
		if err := json.Unmarshal(raw, &geometry); err != nil {
			return nil, err
		}
		return geometry.Coordinates, nil
	}
	var encoded string
	if err := json.Unmarshal(raw, &encoded); err != nil {
		return nil, err
	}
	return decodePolyline(encoded, precision)
}

func decodePolyline(encoded string, precision int) ([]Point, error) {
	factor := 1.0
	for range precision {
		factor *= 10
	}
	latitude, longitude := int64(0), int64(0)
	result := make([]Point, 0)
	for index := 0; index < len(encoded); {
		values := make([]int64, 2)
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
		result = append(result, Point{float64(longitude) / factor, float64(latitude) / factor})
	}
	return result, nil
}

func appendLine(target, addition []Point) []Point {
	if len(target) > 0 && len(addition) > 0 && target[len(target)-1] == addition[0] {
		addition = addition[1:]
	}
	return append(target, addition...)
}
