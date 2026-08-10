package routeanalysis

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/doveva/Gulyaem/backend/internal/geo/domain"
)

var routeIDPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

type routeManifest struct {
	SchemaVersion                int    `json:"schemaVersion"`
	Name                         string `json:"name"`
	CityCode                     string `json:"cityCode"`
	ExpectedSourceChecksum       string `json:"expectedSourceChecksum"`
	ExpectedNormalizationVersion string `json:"expectedNormalizationVersion"`
	Routes                       struct {
		File   string `json:"file"`
		SHA256 string `json:"sha256"`
	} `json:"routes"`
}

type routeFeatureCollection struct {
	Type     string         `json:"type"`
	Features []routeFeature `json:"features"`
}

type routeFeature struct {
	Type       string `json:"type"`
	ID         string `json:"id"`
	Properties struct {
		ID                   string `json:"id"`
		Name                 string `json:"name"`
		AreaID               string `json:"areaId"`
		Description          string `json:"description"`
		IntentionalUnmatched bool   `json:"intentionalUnmatched"`
	} `json:"properties"`
	Geometry lineStringGeometry `json:"geometry"`
}

type lineStringGeometry struct {
	Type        string      `json:"type"`
	Coordinates [][]float64 `json:"coordinates"`
}

type fixtureSet struct {
	manifest routeManifest
	routes   []Route
	byID     map[string]Route
}

func loadFixtureSet(dataRoot string) (fixtureSet, error) {
	directory := filepath.Join(dataRoot, "sample-routes", "spb-stage1")
	manifestBytes, err := os.ReadFile(filepath.Join(directory, "manifest.json"))
	if err != nil {
		return fixtureSet{}, fmt.Errorf("read sample route manifest: %w", err)
	}
	var manifest routeManifest
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		return fixtureSet{}, fmt.Errorf("decode sample route manifest: %w", err)
	}
	if manifest.SchemaVersion != 1 || manifest.CityCode == "" || manifest.Routes.File == "" {
		return fixtureSet{}, errors.New("sample route manifest is invalid")
	}
	routePath := filepath.Join(directory, manifest.Routes.File)
	relative, err := filepath.Rel(directory, routePath)
	if err != nil || relative == ".." || filepath.IsAbs(relative) {
		return fixtureSet{}, errors.New("sample route path escapes fixture directory")
	}
	routeBytes, err := os.ReadFile(routePath)
	if err != nil {
		return fixtureSet{}, fmt.Errorf("read sample routes: %w", err)
	}
	checksum := fmt.Sprintf("%x", sha256.Sum256(routeBytes))
	if checksum != manifest.Routes.SHA256 {
		return fixtureSet{}, fmt.Errorf("sample route checksum mismatch: got %s, want %s", checksum, manifest.Routes.SHA256)
	}
	var collection routeFeatureCollection
	if err := json.Unmarshal(routeBytes, &collection); err != nil {
		return fixtureSet{}, fmt.Errorf("decode sample routes: %w", err)
	}
	if collection.Type != "FeatureCollection" || len(collection.Features) == 0 {
		return fixtureSet{}, errors.New("sample routes must be a non-empty FeatureCollection")
	}
	set := fixtureSet{manifest: manifest, routes: make([]Route, 0, len(collection.Features)), byID: make(map[string]Route)}
	for _, feature := range collection.Features {
		if feature.Type != "Feature" || feature.ID != feature.Properties.ID || !routeIDPattern.MatchString(feature.ID) {
			return fixtureSet{}, fmt.Errorf("sample route id %q is invalid", feature.ID)
		}
		if feature.Geometry.Type != "LineString" || len(feature.Geometry.Coordinates) < 2 {
			return fixtureSet{}, fmt.Errorf("sample route %q geometry is invalid", feature.ID)
		}
		if _, duplicate := set.byID[feature.ID]; duplicate {
			return fixtureSet{}, fmt.Errorf("sample route id %q is duplicated", feature.ID)
		}
		points := make([]domain.Point, len(feature.Geometry.Coordinates))
		for index, coordinate := range feature.Geometry.Coordinates {
			if len(coordinate) < 2 {
				return fixtureSet{}, fmt.Errorf("sample route %q coordinate is invalid", feature.ID)
			}
			points[index] = domain.Point{Lon: coordinate[0], Lat: coordinate[1]}
		}
		geometry, _ := json.Marshal(feature.Geometry)
		route := Route{
			ID: feature.ID, Name: feature.Properties.Name, AreaID: feature.Properties.AreaID,
			Description: feature.Properties.Description, IntentionalUnmatched: feature.Properties.IntentionalUnmatched,
			Geometry: geometry, Points: points,
		}
		set.routes = append(set.routes, route)
		set.byID[route.ID] = route
	}
	return set, nil
}
