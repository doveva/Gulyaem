package httpapi

import "testing"

func TestResolveAnalyzeRequestUsesBalancedDefaults(t *testing.T) {
	request, err := resolveAnalyzeRequest(analyzeRouteRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if request.Matching.SampleStepMeters != 5 || request.Matching.CandidateRadiusMeters != 12 ||
		request.Coverage.Name != "balanced" || request.Coverage.RadiusMeters != 20 {
		t.Fatalf("request = %+v", request)
	}
}

func TestResolveAnalyzeRequestRequiresAllCustomValues(t *testing.T) {
	var body analyzeRouteRequest
	body.Coverage.Profile = "custom"
	if _, err := resolveAnalyzeRequest(body); err == nil {
		t.Fatal("expected custom coverage validation error")
	}
}

func TestResolveAnalyzeRequestRejectsPresetOverrides(t *testing.T) {
	var body analyzeRouteRequest
	body.Coverage.Profile = "strict"
	radius := 15.0
	body.Coverage.RadiusMeters = &radius
	if _, err := resolveAnalyzeRequest(body); err == nil {
		t.Fatal("expected preset override validation error")
	}
}
