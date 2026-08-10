package importing

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"
)

var fixtureNamePattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

type FixtureManifest struct {
	SchemaVersion int           `json:"schemaVersion"`
	Name          string        `json:"name"`
	Description   string        `json:"description"`
	CityCode      string        `json:"cityCode"`
	BBox          BBox          `json:"bbox"`
	Areas         []FixtureArea `json:"areas,omitempty"`
	Source        Source        `json:"source"`
	PBF           PBF           `json:"pbf"`
}

type FixtureArea struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	BBox        BBox   `json:"bbox"`
}

type BBox struct {
	West  float64 `json:"west"`
	South float64 `json:"south"`
	East  float64 `json:"east"`
	North float64 `json:"north"`
}

type Source struct {
	Name        string    `json:"name"`
	URL         string    `json:"url"`
	RetrievedAt time.Time `json:"retrievedAt"`
	Attribution string    `json:"attribution"`
}

type PBF struct {
	File   string `json:"file"`
	SHA256 string `json:"sha256"`
}

type LoadedFixture struct {
	Manifest FixtureManifest
	FilePath string
}

func LoadFixture(dataRoot, name string) (LoadedFixture, error) {
	if !fixtureNamePattern.MatchString(name) {
		return LoadedFixture{}, fmt.Errorf("invalid fixture name %q", name)
	}
	directory := filepath.Join(dataRoot, "test-areas", name)
	manifestPath := filepath.Join(directory, "manifest.json")
	document, err := os.ReadFile(manifestPath)
	if err != nil {
		return LoadedFixture{}, fmt.Errorf("read fixture manifest: %w", err)
	}

	var manifest FixtureManifest
	if err := json.Unmarshal(document, &manifest); err != nil {
		return LoadedFixture{}, fmt.Errorf("decode fixture manifest: %w", err)
	}
	if err := validateManifest(manifest, name); err != nil {
		return LoadedFixture{}, err
	}
	filePath := filepath.Join(directory, manifest.PBF.File)
	relative, err := filepath.Rel(directory, filePath)
	if err != nil || relative == ".." || filepath.IsAbs(relative) || len(relative) >= 3 && relative[:3] == ".."+string(filepath.Separator) {
		return LoadedFixture{}, errors.New("fixture PBF path escapes fixture directory")
	}
	return LoadedFixture{Manifest: manifest, FilePath: filePath}, nil
}

func validateManifest(manifest FixtureManifest, expectedName string) error {
	switch {
	case manifest.SchemaVersion != 1:
		return fmt.Errorf("unsupported fixture manifest schema version %d", manifest.SchemaVersion)
	case manifest.Name != expectedName:
		return fmt.Errorf("fixture manifest name %q does not match directory %q", manifest.Name, expectedName)
	case manifest.CityCode == "":
		return errors.New("fixture cityCode is required")
	case manifest.Source.Name == "":
		return errors.New("fixture source name is required")
	case manifest.Source.RetrievedAt.IsZero():
		return errors.New("fixture source retrievedAt is required")
	case manifest.PBF.File == "":
		return errors.New("fixture PBF file is required")
	case !regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(manifest.PBF.SHA256):
		return errors.New("fixture PBF sha256 must be 64 lowercase hexadecimal characters")
	case !manifest.BBox.Valid():
		return errors.New("fixture bbox is invalid")
	}
	areaNames := make(map[string]struct{}, len(manifest.Areas))
	for _, area := range manifest.Areas {
		if !fixtureNamePattern.MatchString(area.Name) {
			return fmt.Errorf("fixture area name %q is invalid", area.Name)
		}
		if !area.BBox.Valid() {
			return fmt.Errorf("fixture area %q bbox is invalid", area.Name)
		}
		if _, duplicate := areaNames[area.Name]; duplicate {
			return fmt.Errorf("fixture area name %q is duplicated", area.Name)
		}
		areaNames[area.Name] = struct{}{}
	}
	return nil
}

func (bbox BBox) Valid() bool {
	return bbox.West >= -180 && bbox.East <= 180 && bbox.South >= -90 && bbox.North <= 90 &&
		bbox.West < bbox.East && bbox.South < bbox.North
}
