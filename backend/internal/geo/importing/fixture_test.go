package importing

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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
