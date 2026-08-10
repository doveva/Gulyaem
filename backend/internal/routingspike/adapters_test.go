package routingspike

import (
	"encoding/json"
	"math"
	"testing"
)

func TestParseEngineResponses(t *testing.T) {
	tests := []struct {
		name         string
		parse        func([]byte) (rawRouteResponse, error)
		body         string
		wantDistance float64
	}{
		{
			name: "valhalla geojson", parse: parseValhalla, wantDistance: 1250,
			body: `{"trip":{"summary":{"length":1.25,"time":900},"legs":[{"shape":{"type":"LineString","coordinates":[[30,60],[30.1,60.1]]}}]}}`,
		},
		{
			name: "graphhopper", parse: parseGraphHopper, wantDistance: 1250,
			body: `{"paths":[{"distance":1250,"time":900000,"points":{"type":"LineString","coordinates":[[30,60],[30.1,60.1]]}}]}`,
		},
		{
			name: "osrm", parse: func(data []byte) (rawRouteResponse, error) { return parseOSRM(data, "routes") }, wantDistance: 1250,
			body: `{"code":"Ok","routes":[{"distance":1250,"duration":900,"geometry":{"type":"LineString","coordinates":[[30,60],[30.1,60.1]]}}]}`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := test.parse([]byte(test.body))
			if err != nil {
				t.Fatal(err)
			}
			if result.DistanceMeters != test.wantDistance || len(result.Geometry.Coordinates) != 2 {
				t.Fatalf("unexpected result: %+v", result)
			}
		})
	}
}

func TestDecodePolyline(t *testing.T) {
	shape, _ := json.Marshal("_izlhA~rlgdF_{geC~ywl@_kwzCn`{nI")
	points, err := decodeShape(shape, 6)
	if err != nil {
		t.Fatal(err)
	}
	if len(points) != 3 || math.Abs(points[0][1]-38.5) > 0.00001 || math.Abs(points[0][0]-(-120.2)) > 0.00001 {
		t.Fatalf("unexpected decoded polyline: %#v", points)
	}
}
