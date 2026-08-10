package importing

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLoadFixtureRejectsPathEscape(t *testing.T) {
	root := t.TempDir()
	directory := filepath.Join(root, "test-areas", "unsafe")
	if err := os.MkdirAll(directory, 0o700); err != nil {
		t.Fatal(err)
	}
	manifest := `{
		"schemaVersion": 1,
		"name": "unsafe",
		"cityCode": "spb",
		"bbox": {"west": 30, "south": 59, "east": 31, "north": 60},
		"source": {
			"name": "openstreetmap",
			"url": "https://example.test",
			"retrievedAt": "2026-08-09T00:00:00Z",
			"attribution": "OSM"
		},
		"pbf": {"file": "../outside.pbf", "sha256": "` + strings.Repeat("a", 64) + `"}
	}`
	if err := os.WriteFile(filepath.Join(directory, "manifest.json"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadFixture(root, "unsafe"); err == nil || !strings.Contains(err.Error(), "escapes") {
		t.Fatalf("LoadFixture() error = %v", err)
	}
}

func TestLoadCommittedDenseCenterFixture(t *testing.T) {
	root := filepath.Join("..", "..", "..", "..", "data")
	fixture, err := LoadFixture(root, "spb-dense-center")
	if err != nil {
		t.Fatalf("LoadFixture() error = %v", err)
	}
	if fixture.Manifest.CityCode != "spb" || fixture.Manifest.PBF.SHA256 == "" {
		t.Fatalf("fixture = %+v", fixture)
	}
	if _, err := os.Stat(fixture.FilePath); err != nil {
		t.Fatalf("fixture PBF: %v", err)
	}
	checksum, _, err := fileChecksum(fixture.FilePath)
	if err != nil {
		t.Fatal(err)
	}
	if checksum != fixture.Manifest.PBF.SHA256 {
		t.Fatalf("fixture checksum = %s, manifest = %s", checksum, fixture.Manifest.PBF.SHA256)
	}
}

func TestValidateManifestRejectsDuplicateAreaNames(t *testing.T) {
	manifest := FixtureManifest{
		SchemaVersion: 1,
		Name:          "fixture",
		CityCode:      "spb",
		BBox:          BBox{West: 30, South: 59, East: 31, North: 60},
		Areas: []FixtureArea{
			{Name: "center", BBox: BBox{West: 30, South: 59, East: 30.2, North: 59.2}},
			{Name: "center", BBox: BBox{West: 30.3, South: 59.3, East: 30.5, North: 59.5}},
		},
		Source: Source{Name: "openstreetmap", RetrievedAt: mustParseTime(t, "2026-08-09T00:00:00Z")},
		PBF:    PBF{File: "fixture.osm.pbf", SHA256: strings.Repeat("a", 64)},
	}
	if err := validateManifest(manifest, "fixture"); err == nil || !strings.Contains(err.Error(), "duplicated") {
		t.Fatalf("validateManifest() error = %v", err)
	}
}

func mustParseTime(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}
