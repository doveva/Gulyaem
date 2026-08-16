package preview

import (
	"encoding/json"
	"testing"

	"github.com/doveva/Gulyaem/backend/internal/geo/routeanalysis"
	"github.com/doveva/Gulyaem/backend/internal/routing/port"
)

func TestFingerprintIsStableAndCoversMaterializationInputs(t *testing.T) {
	request := Request{CityID: "city", Profile: "pedestrian", Waypoints: []port.Point{{Lat: 59.9, Lon: 30.3}, {Lat: 60, Lon: 30.4}}}
	result := Result{GeoDataVersion: routeanalysis.VersionReference{ID: "version", CityID: "city", SourceChecksum: "source", NormalizationVersion: "normalization"}, Routing: Routing{Profile: "pedestrian", DistanceMeters: 100, DurationSeconds: 80, Geometry: json.RawMessage(`{"type":"LineString","coordinates":[[30.3,59.9],[30.4,60]]}`)}, ExplorationPreview: ExplorationPreview{NormalizedRoute: json.RawMessage(`{"type":"MultiLineString","coordinates":[[[30.3,59.9],[30.4,60]]]}`), CoverageProfile: routeanalysis.CoverageProfiles["balanced"]}, Materialization: MaterializationProvenance{RoutingMetadata: port.Metadata{Engine: "valhalla", EngineVersion: "3.7", GraphChecksum: "graph", SourceChecksum: "source"}, AnalysisVersion: routeanalysis.AnalysisVersion, Matching: routeanalysis.DefaultMatchingParameters()}}
	first, err := fingerprint(request, result)
	if err != nil {
		t.Fatal(err)
	}
	second, _ := fingerprint(request, result)
	if first != second {
		t.Fatalf("unstable fingerprint %q != %q", first, second)
	}
	changed := result
	changed.Materialization.RoutingMetadata.GraphChecksum = "other"
	third, _ := fingerprint(request, changed)
	if first == third {
		t.Fatal("graph checksum did not affect fingerprint")
	}
	changed = result
	changed.Routing.Geometry = json.RawMessage(`{"type":"LineString","coordinates":[[30.3,59.9],[30.5,60]]}`)
	fourth, _ := fingerprint(request, changed)
	if first == fourth {
		t.Fatal("geometry did not affect fingerprint")
	}
	changed = result
	changed.ExplorationPreview.CoverageProfile.RadiusMeters++
	fifth, _ := fingerprint(request, changed)
	if first == fifth {
		t.Fatal("coverage profile did not affect fingerprint")
	}
}
