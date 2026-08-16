#!/usr/bin/env python3
"""Generate ADR-0014 coverage-v2 validation evidence without mutating Stage 1 evidence."""

import stage1


DEFAULT_OUTPUT = "data/validation/spb-stage3-coverage-v2/report.json"
ANALYSIS_VERSION = "route-analysis-v2"
COVERAGE_PROFILES = {
    "strict": {"name": "strict", "radiusMeters": 50, "coverageRatio": 0.8,
               "minRequiredMeters": 20, "maxRequiredMeters": 120},
    "balanced": {"name": "balanced", "radiusMeters": 100, "coverageRatio": 0.4,
                 "minRequiredMeters": 15, "maxRequiredMeters": 80},
    "generous": {"name": "generous", "radiusMeters": 200, "coverageRatio": 0.4,
                 "minRequiredMeters": 10, "maxRequiredMeters": 50},
}


if __name__ == "__main__":
    stage1.main(
        coverage_profiles=COVERAGE_PROFILES,
        default_output=DEFAULT_OUTPUT,
        analysis_version=ANALYSIS_VERSION,
    )
