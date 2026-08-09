package segmenting

import (
	"sort"
	"strings"

	"github.com/doveva/Gulyaem/backend/internal/geo/domain"
)

var relevantTagKeys = []string{
	"access", "access:conditional", "area", "bridge", "foot", "foot:backward",
	"foot:conditional", "foot:forward", "footway", "highway", "indoor", "level", "name",
	"oneway:foot", "service", "sidewalk", "sidewalk:both", "sidewalk:left", "sidewalk:right",
	"surface", "tunnel",
}

type wayProfile struct {
	candidate       bool
	unsupportedArea bool
	classification  domain.StreetSegmentClassification
	reasonCode      string
	relevantTags    map[string]string
	warnings        []string
	semanticKey     string
}

func normalizeWay(tags map[string]string) wayProfile {
	highway := normalizedTag(tags, "highway")
	if highway == "" {
		return wayProfile{}
	}

	profile := wayProfile{candidate: true, relevantTags: selectRelevantTags(tags)}
	if normalizedTag(tags, "area") == "yes" {
		profile.candidate = false
		profile.unsupportedArea = true
		return profile
	}

	foot := normalizedTag(tags, "foot")
	explicitFootAllowed := oneOf(foot, "yes", "designated", "permissive")
	if oneOf(foot, "no", "private", "use_sidepath") {
		return profile.complete(domain.StreetSegmentIgnore, "foot_access_prohibited")
	}
	if foot == "discouraged" || normalizedTag(tags, "foot:conditional") != "" || normalizedTag(tags, "access:conditional") != "" {
		return profile.complete(domain.StreetSegmentRoutableOnly, "conditional_or_discouraged_access")
	}

	access := normalizedTag(tags, "access")
	if !explicitFootAllowed && oneOf(access, "no", "private", "customers", "agricultural", "forestry", "military", "permit", "delivery") {
		return profile.complete(domain.StreetSegmentIgnore, "general_access_restricted")
	}
	if !explicitFootAllowed && access == "destination" {
		return profile.complete(domain.StreetSegmentRoutableOnly, "destination_access")
	}

	if normalizedTag(tags, "indoor") == "yes" || highway == "corridor" {
		return profile.complete(domain.StreetSegmentIgnore, "indoor_path")
	}

	switch highway {
	case "motorway", "trunk", "raceway", "construction", "proposed", "abandoned":
		return profile.complete(domain.StreetSegmentIgnore, "unsupported_highway_class")
	case "motorway_link", "trunk_link":
		return profile.complete(domain.StreetSegmentIgnore, "unsupported_highway_link")
	case "primary_link", "secondary_link", "tertiary_link":
		return profile.complete(domain.StreetSegmentRoutableOnly, "road_link_connector")
	case "cycleway", "bridleway":
		if explicitFootAllowed {
			return profile.complete(domain.StreetSegmentExplore, "designated_shared_path")
		}
		return profile.complete(domain.StreetSegmentIgnore, "pedestrian_access_not_designated")
	case "footway":
		switch normalizedTag(tags, "footway") {
		case "crossing":
			return profile.complete(domain.StreetSegmentRoutableOnly, "crossing_connector")
		case "traffic_island":
			return profile.complete(domain.StreetSegmentRoutableOnly, "traffic_island_connector")
		case "link":
			return profile.complete(domain.StreetSegmentRoutableOnly, "footway_link_connector")
		default:
			return profile.complete(domain.StreetSegmentExplore, "public_footway")
		}
	case "path":
		return profile.complete(domain.StreetSegmentExplore, "public_path")
	case "track":
		return profile.complete(domain.StreetSegmentExplore, "public_track")
	case "steps":
		return profile.complete(domain.StreetSegmentExplore, "public_steps")
	case "pedestrian":
		return profile.complete(domain.StreetSegmentExplore, "pedestrian_street")
	case "living_street":
		return profile.complete(domain.StreetSegmentExplore, "living_street")
	case "residential", "unclassified":
		if hasSeparateSidewalk(tags) {
			return profile.complete(domain.StreetSegmentRoutableOnly, "separate_sidewalk_representation")
		}
		return profile.complete(domain.StreetSegmentExplore, "public_street")
	case "primary", "secondary", "tertiary":
		if hasSeparateSidewalk(tags) {
			return profile.complete(domain.StreetSegmentRoutableOnly, "separate_sidewalk_representation")
		}
		if normalizedTag(tags, "sidewalk") == "no" {
			return profile.complete(domain.StreetSegmentRoutableOnly, "major_road_without_sidewalk")
		}
		return profile.complete(domain.StreetSegmentExplore, "public_major_street")
	case "service":
		service := normalizedTag(tags, "service")
		if service == "alley" || service == "track" {
			return profile.complete(domain.StreetSegmentExplore, "public_service_"+service)
		}
		if foot == "designated" {
			return profile.complete(domain.StreetSegmentExplore, "designated_service_path")
		}
		if service != "" && !oneOf(service, "driveway", "parking_aisle", "emergency_access", "drive-through", "alley", "track") {
			profile.warnings = append(profile.warnings, "unknown_service_subtype")
		}
		return profile.complete(domain.StreetSegmentRoutableOnly, "service_access")
	default:
		profile.warnings = append(profile.warnings, "unknown_highway_class")
		return profile.complete(domain.StreetSegmentIgnore, "unsupported_highway_class")
	}
}

func (profile wayProfile) complete(classification domain.StreetSegmentClassification, reason string) wayProfile {
	profile.classification = classification
	profile.reasonCode = reason
	profile.semanticKey = buildSemanticKey(profile)
	return profile
}

func buildSemanticKey(profile wayProfile) string {
	parts := []string{string(profile.classification), profile.reasonCode}
	keys := make([]string, 0, len(profile.relevantTags))
	for key := range profile.relevantTags {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		parts = append(parts, key+"="+profile.relevantTags[key])
	}
	return strings.Join(parts, "|")
}

func selectRelevantTags(tags map[string]string) map[string]string {
	selected := make(map[string]string)
	for _, key := range relevantTagKeys {
		if value := strings.TrimSpace(tags[key]); value != "" {
			selected[key] = value
		}
	}
	return selected
}

func normalizedTag(tags map[string]string, key string) string {
	return strings.ToLower(strings.TrimSpace(tags[key]))
}

func hasSeparateSidewalk(tags map[string]string) bool {
	for _, key := range []string{"sidewalk", "sidewalk:both", "sidewalk:left", "sidewalk:right"} {
		if normalizedTag(tags, key) == "separate" {
			return true
		}
	}
	return false
}

func oneOf(value string, candidates ...string) bool {
	for _, candidate := range candidates {
		if value == candidate {
			return true
		}
	}
	return false
}
