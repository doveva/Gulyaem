import json
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

import coverage_v2
import stage1


class ValidationProfileContractTest(unittest.TestCase):
    def test_stage1_contract_remains_frozen(self):
        self.assertEqual(
            stage1.COVERAGE_PROFILES,
            {
                "strict": {"name": "strict", "radiusMeters": 35, "coverageRatio": 0.8,
                           "minRequiredMeters": 20, "maxRequiredMeters": 120},
                "balanced": {"name": "balanced", "radiusMeters": 50, "coverageRatio": 0.6,
                             "minRequiredMeters": 15, "maxRequiredMeters": 80},
                "generous": {"name": "generous", "radiusMeters": 100, "coverageRatio": 0.4,
                             "minRequiredMeters": 10, "maxRequiredMeters": 50},
            },
        )
        self.assertEqual(stage1.DEFAULT_OUTPUT, "data/validation/spb-stage1/report.json")

    def test_coverage_v2_has_separate_contract_and_output(self):
        self.assertEqual([profile["radiusMeters"] for profile in coverage_v2.COVERAGE_PROFILES.values()],
                         [50, 100, 200])
        self.assertEqual(coverage_v2.COVERAGE_PROFILES["balanced"]["coverageRatio"], 0.4)
        self.assertEqual(coverage_v2.ANALYSIS_VERSION, "route-analysis-v2")
        self.assertNotEqual(coverage_v2.DEFAULT_OUTPUT, stage1.DEFAULT_OUTPUT)

    def test_failed_run_does_not_replace_accepted_report(self):
        with tempfile.TemporaryDirectory() as directory:
            output = Path(directory) / "report.json"
            output.write_text('{"status":"accepted"}\n')
            failed_output = stage1.publish_report({"status": "failed"}, output)

            self.assertEqual(json.loads(output.read_text()), {"status": "accepted"})
            self.assertEqual(failed_output.name, "report.failed.json")
            self.assertEqual(json.loads(failed_output.read_text()), {"status": "failed"})

    def test_route_analysis_runs_every_requested_profile(self):
        analysis = {
            "coverageProfile": stage1.COVERAGE_PROFILES["strict"],
            "contextRadiusMeters": 225,
            "metrics": {
                "routeMatchedRatio": 1,
                "completedNetworkRatio": 0.5,
                "geometricCoveredLengthMeters": 10,
                "completedNetworkLengthMeters": 10,
                "contextExplorableLengthMeters": 20,
                "routeUnmatchedLengthMeters": 0,
            },
        }
        route = {"id": "route", "name": "Route", "areaId": "area", "intentionalUnmatched": False}
        with patch.object(stage1, "request_json", return_value=(analysis, 100, 2)) as request:
            result = stage1.analyze_routes("http://api", [route], ["strict"])

        self.assertEqual([profile["profile"] for profile in result[0]["profiles"]], ["strict"])
        request.assert_called_once()


if __name__ == "__main__":
    unittest.main()
