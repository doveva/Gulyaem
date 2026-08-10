package districting

import (
	"path/filepath"
	"testing"
)

func TestLoadCommittedDistrictFixture(t *testing.T) {
	root := filepath.Join("..", "..", "..", "..", "data")
	fixture, err := LoadFixture(root, "spb-administrative-districts")
	if err != nil {
		t.Fatal(err)
	}
	if fixture.Manifest.CityCode != "spb" || fixture.Manifest.GeoJSON.FeatureCount != 18 || len(fixture.Manifest.Source.RelationIDs) != 18 {
		t.Fatalf("fixture = %+v", fixture.Manifest)
	}
	checksum, size, err := fileChecksum(fixture.FilePath)
	if err != nil {
		t.Fatal(err)
	}
	if checksum != fixture.Manifest.GeoJSON.SHA256 || size != fixture.Manifest.GeoJSON.SizeBytes {
		t.Fatalf("artifact checksum=%s size=%d manifest=%+v", checksum, size, fixture.Manifest.GeoJSON)
	}
	districts, err := parseGeoJSON(fixture.FilePath)
	if err != nil {
		t.Fatal(err)
	}
	if len(districts) != 18 {
		t.Fatalf("district count = %d", len(districts))
	}
}

func TestLoadFixtureRejectsPathComponents(t *testing.T) {
	if _, err := LoadFixture(t.TempDir(), "../unsafe"); err == nil {
		t.Fatal("LoadFixture() unexpectedly accepted a path component")
	}
}
