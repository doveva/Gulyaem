package preview

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

const FingerprintVersion = "stage3-preview-fingerprint-v1"

// fingerprint covers every server input or derived value that can change a
// materialized Route. Callers must treat the result as opaque, not as a token.
func fingerprint(request Request, result Result) (string, error) {
	payload := struct {
		CityID             string  `json:"cityId"`
		Profile            string  `json:"profile"`
		Waypoints          any     `json:"waypoints"`
		GeoDataVersion     any     `json:"geoDataVersion"`
		RoutingMetadata    any     `json:"routingMetadata"`
		RouteGeometry      any     `json:"routeGeometry"`
		NormalizedGeometry any     `json:"normalizedGeometry"`
		DistanceMeters     float64 `json:"distanceMeters"`
		DurationSeconds    float64 `json:"durationSeconds"`
		AnalysisVersion    string  `json:"analysisVersion"`
		Matching           any     `json:"matching"`
		CoverageProfile    any     `json:"coverageProfile"`
	}{
		CityID: request.CityID, Profile: request.Profile, Waypoints: request.Waypoints,
		GeoDataVersion: result.GeoDataVersion, RoutingMetadata: result.Materialization.RoutingMetadata,
		RouteGeometry: result.Routing.Geometry, NormalizedGeometry: result.ExplorationPreview.NormalizedRoute,
		DistanceMeters: result.Routing.DistanceMeters, DurationSeconds: result.Routing.DurationSeconds,
		AnalysisVersion: result.Materialization.AnalysisVersion, Matching: result.Materialization.Matching,
		CoverageProfile: result.ExplorationPreview.CoverageProfile,
	}
	canonical, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	hash := sha256.New()
	_, _ = hash.Write([]byte(FingerprintVersion))
	_, _ = hash.Write([]byte{0})
	_, _ = hash.Write(canonical)
	return FingerprintVersion + ":sha256:" + hex.EncodeToString(hash.Sum(nil)), nil
}
