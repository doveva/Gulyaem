package port

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

var (
	ErrRouteNotFound   = errors.New("route not found")
	ErrUnavailable     = errors.New("routing service unavailable")
	ErrTimeout         = errors.New("routing service timeout")
	ErrInvalidResponse = errors.New("routing service returned an invalid response")
)

type Point struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

type Waypoint struct {
	Input    Point  `json:"input"`
	Resolved *Point `json:"resolved,omitempty"`
}

type Metadata struct {
	Engine         string     `json:"engine"`
	EngineVersion  string     `json:"engineVersion"`
	CityID         string     `json:"cityId"`
	SourceChecksum string     `json:"sourceChecksum"`
	Profile        string     `json:"profile"`
	GraphArtifact  string     `json:"graphArtifact"`
	GraphChecksum  string     `json:"graphChecksum"`
	BuiltAt        *time.Time `json:"builtAt,omitempty"`
}

type RouteRequest struct {
	Profile   string
	Waypoints []Point
}

type RouteResult struct {
	DistanceMeters  float64
	DurationSeconds float64
	Geometry        json.RawMessage
	Waypoints       []Waypoint
}

type RoutingEngine interface {
	Metadata(context.Context) (Metadata, error)
	Route(context.Context, RouteRequest) (RouteResult, error)
}
