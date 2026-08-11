package routeanalysis

import (
	"strings"

	"github.com/doveva/Gulyaem/backend/internal/geo/domain"
)

// GradeSignature identifies the vertical/indoor context in which radius
// coverage is allowed to propagate.
func GradeSignature(attributes domain.StreetSegmentAttributes) string {
	tags := attributes.SourceTags
	parts := []string{"surface"}
	for _, key := range []string{"bridge", "tunnel", "indoor", "level"} {
		if value := strings.TrimSpace(tags[key]); value != "" && value != "no" && value != "0" {
			parts = append(parts, key+"="+value)
		}
	}
	return strings.Join(parts, ";")
}
