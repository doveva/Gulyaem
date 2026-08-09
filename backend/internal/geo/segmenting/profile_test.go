package segmenting

import (
	"testing"

	"github.com/doveva/Gulyaem/backend/internal/geo/domain"
)

func TestWalkabilityProfile(t *testing.T) {
	tests := []struct {
		name           string
		tags           map[string]string
		classification domain.StreetSegmentClassification
		reason         string
		candidate      bool
	}{
		{name: "sidewalk", tags: map[string]string{"highway": "footway", "footway": "sidewalk"}, classification: domain.StreetSegmentExplore, reason: "public_footway", candidate: true},
		{name: "crossing", tags: map[string]string{"highway": "footway", "footway": "crossing"}, classification: domain.StreetSegmentRoutableOnly, reason: "crossing_connector", candidate: true},
		{name: "steps", tags: map[string]string{"highway": "steps"}, classification: domain.StreetSegmentExplore, reason: "public_steps", candidate: true},
		{name: "service track", tags: map[string]string{"highway": "service", "service": "track"}, classification: domain.StreetSegmentExplore, reason: "public_service_track", candidate: true},
		{name: "driveway", tags: map[string]string{"highway": "service", "service": "driveway"}, classification: domain.StreetSegmentRoutableOnly, reason: "service_access", candidate: true},
		{name: "private footway", tags: map[string]string{"highway": "footway", "access": "private"}, classification: domain.StreetSegmentIgnore, reason: "general_access_restricted", candidate: true},
		{name: "foot override", tags: map[string]string{"highway": "footway", "access": "private", "foot": "yes"}, classification: domain.StreetSegmentExplore, reason: "public_footway", candidate: true},
		{name: "separate sidewalk", tags: map[string]string{"highway": "residential", "sidewalk:both": "separate"}, classification: domain.StreetSegmentRoutableOnly, reason: "separate_sidewalk_representation", candidate: true},
		{name: "indoor corridor", tags: map[string]string{"highway": "corridor", "indoor": "yes"}, classification: domain.StreetSegmentIgnore, reason: "indoor_path", candidate: true},
		{name: "pedestrian area", tags: map[string]string{"highway": "pedestrian", "area": "yes"}, candidate: false},
		{name: "not highway", tags: map[string]string{"building": "yes"}, candidate: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			profile := normalizeWay(test.tags)
			if profile.candidate != test.candidate {
				t.Fatalf("candidate = %v, want %v", profile.candidate, test.candidate)
			}
			if !test.candidate {
				return
			}
			if profile.classification != test.classification || profile.reasonCode != test.reason {
				t.Fatalf("profile = %s/%s, want %s/%s", profile.classification, profile.reasonCode, test.classification, test.reason)
			}
		})
	}
}

func TestDirectionalTagsDoNotChangeClassificationButChangeSemantics(t *testing.T) {
	base := normalizeWay(map[string]string{"highway": "footway"})
	directional := normalizeWay(map[string]string{"highway": "footway", "oneway:foot": "yes"})
	if base.classification != directional.classification {
		t.Fatalf("classification changed: %s != %s", base.classification, directional.classification)
	}
	if base.semanticKey == directional.semanticKey {
		t.Fatal("directional metadata must create a semantic boundary")
	}
}
