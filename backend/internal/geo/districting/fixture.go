package districting

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Fixture struct {
	Manifest Manifest
	FilePath string
}

type Manifest struct {
	SchemaVersion int            `json:"schemaVersion"`
	CityCode      string         `json:"cityCode"`
	Source        FixtureSource  `json:"source"`
	GeoJSON       FixtureGeoJSON `json:"geojson"`
}

type FixtureSource struct {
	Name             string    `json:"name"`
	URL              string    `json:"url"`
	License          string    `json:"license"`
	RetrievedAt      time.Time `json:"retrievedAt"`
	OSMBaseTimestamp time.Time `json:"osmBaseTimestamp"`
	RelationIDs      []int64   `json:"relationIds"`
}

type FixtureGeoJSON struct {
	File         string `json:"file"`
	SHA256       string `json:"sha256"`
	SizeBytes    int64  `json:"sizeBytes"`
	FeatureCount int    `json:"featureCount"`
}

func LoadFixture(dataRoot, name string) (Fixture, error) {
	if strings.TrimSpace(name) == "" || filepath.Base(name) != name {
		return Fixture{}, errors.New("district fixture name must be a single path component")
	}
	directory := filepath.Join(dataRoot, "districts", name)
	manifestPath := filepath.Join(directory, "manifest.json")
	contents, err := os.ReadFile(manifestPath)
	if err != nil {
		return Fixture{}, fmt.Errorf("read district fixture manifest: %w", err)
	}
	var manifest Manifest
	if err := json.Unmarshal(contents, &manifest); err != nil {
		return Fixture{}, fmt.Errorf("decode district fixture manifest: %w", err)
	}
	if manifest.SchemaVersion != 1 || manifest.CityCode == "" || manifest.Source.Name == "" ||
		manifest.GeoJSON.File == "" || manifest.GeoJSON.SHA256 == "" || manifest.GeoJSON.FeatureCount <= 0 {
		return Fixture{}, errors.New("district fixture manifest is incomplete or unsupported")
	}
	if filepath.Base(manifest.GeoJSON.File) != manifest.GeoJSON.File {
		return Fixture{}, errors.New("district fixture GeoJSON file must be a single path component")
	}
	return Fixture{Manifest: manifest, FilePath: filepath.Join(directory, manifest.GeoJSON.File)}, nil
}
