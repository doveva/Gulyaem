#!/usr/bin/env python3
"""Generate reproducible Stage 1.7 API, fixture and coverage validation evidence."""

import argparse
import datetime as dt
import json
import math
import platform
import statistics
import time
import urllib.error
import urllib.parse
import urllib.request
from collections import defaultdict
from pathlib import Path


CITY_ID = "01900000-0000-7000-8000-000000000001"
DEFAULT_OUTPUT = "data/validation/spb-stage1/report.json"
COVERAGE_PROFILES = {
    "strict": {"name": "strict", "radiusMeters": 35, "coverageRatio": 0.8,
               "minRequiredMeters": 20, "maxRequiredMeters": 120},
    "balanced": {"name": "balanced", "radiusMeters": 50, "coverageRatio": 0.6,
                 "minRequiredMeters": 15, "maxRequiredMeters": 80},
    "generous": {"name": "generous", "radiusMeters": 100, "coverageRatio": 0.4,
                 "minRequiredMeters": 10, "maxRequiredMeters": 50},
}
ANALYSIS_CONTEXT_RADIUS_METERS = 225


class HTTPRequestError(RuntimeError):
    def __init__(self, url, status, payload):
        super().__init__("{} returned HTTP {}: {}".format(url, status, payload))
        self.status = status
        self.payload = payload


def request_json(url, method="GET", payload=None):
    body = None if payload is None else json.dumps(payload).encode("utf-8")
    headers = {"Accept": "application/json"}
    if body is not None:
        headers["Content-Type"] = "application/json"
    request = urllib.request.Request(url, data=body, headers=headers, method=method)
    started = time.perf_counter()
    try:
        with urllib.request.urlopen(request, timeout=60) as response:
            raw = response.read()
            status = response.status
    except urllib.error.HTTPError as error:
        detail = error.read().decode("utf-8", errors="replace")
        try:
            decoded = json.loads(detail)
        except json.JSONDecodeError:
            decoded = {"raw": detail}
        raise HTTPRequestError(url, error.code, decoded) from error
    elapsed_ms = (time.perf_counter() - started) * 1000
    if status < 200 or status >= 300:
        raise RuntimeError("{} returned HTTP {}".format(url, status))
    return json.loads(raw), len(raw), elapsed_ms


def percentile(values, fraction):
    ordered = sorted(values)
    if not ordered:
        return 0.0
    position = (len(ordered) - 1) * fraction
    lower = math.floor(position)
    upper = math.ceil(position)
    if lower == upper:
        return ordered[lower]
    return ordered[lower] + (ordered[upper] - ordered[lower]) * (position - lower)


def rounded(value):
    return round(value, 3)


def segment_url(api_url, bbox):
    query = urllib.parse.urlencode({
        "cityId": CITY_ID,
        "bbox": ",".join(str(bbox[key]) for key in ("west", "south", "east", "north")),
        "classification": "EXPLORE,ROUTABLE_ONLY,IGNORE",
    })
    return "{}/api/v1/geo/segments?{}".format(api_url, query)


def district_url(api_url, bbox):
    query = urllib.parse.urlencode({
        "cityId": CITY_ID,
        "bbox": ",".join(str(bbox[key]) for key in ("west", "south", "east", "north")),
    })
    return "{}/api/v1/geo/districts?{}".format(api_url, query)


def reason_summary(features):
    values = defaultdict(lambda: {"segments": 0, "lengthMeters": 0.0})
    for feature in features:
        properties = feature["properties"]
        key = "{}:{}".format(properties["classification"], properties["reasonCode"])
        values[key]["segments"] += 1
        values[key]["lengthMeters"] += properties["lengthMeters"]
    return {
        key: {"segments": value["segments"], "lengthMeters": rounded(value["lengthMeters"])}
        for key, value in sorted(values.items())
    }


def measure_area(api_url, area, warm_requests):
    collection, response_bytes, first_ms = request_json(segment_url(api_url, area["bbox"]))
    warm_latencies = []
    for _ in range(warm_requests):
        _, _, latency = request_json(segment_url(api_url, area["bbox"]))
        warm_latencies.append(latency)
    districts, district_bytes, district_ms = request_json(district_url(api_url, area["bbox"]))
    return {
        "id": area["name"],
        "sourceAreaId": area["sourceAreaId"],
        "description": area["description"],
        "bbox": area["bbox"],
        "segments": {
            "returnedCount": collection["meta"]["returnedCount"],
            "responseBytes": response_bytes,
            "firstMilliseconds": rounded(first_ms),
            "warmRequests": warm_requests,
            "p50Milliseconds": rounded(statistics.median(warm_latencies)),
            "p95Milliseconds": rounded(percentile(warm_latencies, 0.95)),
            "statistics": collection["meta"]["statistics"],
            "reasonDistribution": reason_summary(collection["features"]),
        },
        "districts": {
            "returnedCount": districts["meta"]["returnedCount"],
            "responseBytes": district_bytes,
            "latencyMilliseconds": rounded(district_ms),
        },
    }


def probe_source_area(api_url, area):
    try:
        collection, response_bytes, latency = request_json(segment_url(api_url, area["bbox"]))
        return {
            "id": area["name"],
            "bbox": area["bbox"],
            "status": 200,
            "returnedCount": collection["meta"]["returnedCount"],
            "responseBytes": response_bytes,
            "latencyMilliseconds": rounded(latency),
        }
    except HTTPRequestError as error:
        api_error = error.payload.get("error", {}) if isinstance(error.payload, dict) else {}
        return {
            "id": area["name"],
            "bbox": area["bbox"],
            "status": error.status,
            "errorCode": api_error.get("code"),
            "message": api_error.get("message"),
        }


def analyze_routes(api_url, routes, profile_names):
    results = []
    for route in routes:
        route_profiles = []
        for profile in profile_names:
            url = "{}/api/v1/geo/sample-routes/{}/analyze?{}".format(
                api_url,
                urllib.parse.quote(route["id"]),
                urllib.parse.urlencode({"cityId": CITY_ID}),
            )
            analysis, response_bytes, latency = request_json(
                url, method="POST", payload={"coverage": {"profile": profile}}
            )
            route_profiles.append({
                "profile": profile,
                "responseBytes": response_bytes,
                "latencyMilliseconds": rounded(latency),
                "coverageProfile": analysis["coverageProfile"],
                "contextRadiusMeters": analysis["contextRadiusMeters"],
                "routeMatchedRatio": analysis["metrics"]["routeMatchedRatio"],
                "completedNetworkRatio": analysis["metrics"]["completedNetworkRatio"],
                "geometricCoveredLengthMeters": analysis["metrics"]["geometricCoveredLengthMeters"],
                "completedNetworkLengthMeters": analysis["metrics"]["completedNetworkLengthMeters"],
                "contextExplorableLengthMeters": analysis["metrics"]["contextExplorableLengthMeters"],
                "routeUnmatchedLengthMeters": analysis["metrics"]["routeUnmatchedLengthMeters"],
            })
        results.append({
            "routeId": route["id"],
            "name": route["name"],
            "areaId": route["areaId"],
            "intentionalUnmatched": route["intentionalUnmatched"],
            "profiles": route_profiles,
        })
    return results


def add_check(checks, check_id, passed, evidence):
    checks.append({"id": check_id, "passed": bool(passed), "evidence": evidence})


def publish_report(report, output):
    output.parent.mkdir(parents=True, exist_ok=True)
    if report["status"] == "passed":
        temporary = output.with_suffix(output.suffix + ".tmp")
        temporary.write_text(json.dumps(report, ensure_ascii=False, indent=2) + "\n")
        temporary.replace(output)
        return output
    failed_output = output.with_suffix(".failed" + output.suffix)
    failed_output.write_text(json.dumps(report, ensure_ascii=False, indent=2) + "\n")
    return failed_output


def main(coverage_profiles=None, default_output=DEFAULT_OUTPUT,
         analysis_version=None):
    coverage_profiles = coverage_profiles or COVERAGE_PROFILES
    profiles = tuple(coverage_profiles)
    parser = argparse.ArgumentParser()
    parser.add_argument("--root", default=".")
    parser.add_argument("--api-url", default="http://localhost:8080")
    parser.add_argument("--warm-requests", type=int, default=30)
    parser.add_argument("--output", default=default_output)
    args = parser.parse_args()
    if args.warm_requests < 1:
        parser.error("--warm-requests must be positive")

    root = Path(args.root).resolve()
    manifest = json.loads((root / "data/test-areas/spb-stage1-validation/manifest.json").read_text())
    validation_cases = json.loads((root / "data/validation/spb-stage1/cases.json").read_text())
    route_fixture = json.loads((root / "data/sample-routes/spb-stage1/routes.geojson").read_text())
    api_url = args.api_url.rstrip("/")

    version, _, version_ms = request_json(
        "{}/api/v1/cities/{}/geo-version".format(api_url, CITY_ID)
    )
    routes_response, _, _ = request_json(
        "{}/api/v1/geo/sample-routes?{}".format(
            api_url, urllib.parse.urlencode({"cityId": CITY_ID})
        )
    )
    route_names = {
        feature["properties"]["id"]: feature["properties"]["name"]
        for feature in route_fixture["features"]
    }
    routes = []
    for route in routes_response["routes"]:
        routes.append({
            "id": route["id"],
            "name": route_names.get(route["id"], route["name"]),
            "areaId": route["areaId"],
            "intentionalUnmatched": route["intentionalUnmatched"],
        })

    viewports = []
    for viewport in validation_cases["viewports"]:
        viewports.append(measure_area(api_url, {
            "name": viewport["id"],
            "sourceAreaId": viewport["sourceAreaId"],
            "description": viewport["description"],
            "bbox": viewport["bbox"],
        }, args.warm_requests))
    source_area_probes = [probe_source_area(api_url, area) for area in manifest["areas"]]
    route_results = analyze_routes(api_url, routes, profiles)
    checks = []
    expected_checksum = manifest["pbf"]["sha256"]
    add_check(checks, "geo-version-ready", version["status"] == "READY", version["status"])
    add_check(checks, "fixture-checksum", version["sourceChecksum"] == expected_checksum, version["sourceChecksum"])
    add_check(
        checks, "normalization-version", version["normalizationVersion"] == "stage1-segments-v1",
        version["normalizationVersion"],
    )
    import_report = version.get("importReport") or {}
    add_check(checks, "no-invalid-geometries", import_report.get("invalidGeometries") == 0, import_report.get("invalidGeometries"))
    add_check(checks, "no-zero-length-segments", import_report.get("zeroLengthSegments") == 0, import_report.get("zeroLengthSegments"))
    for area in viewports:
        stats = area["segments"]["statistics"]
        add_check(checks, "{}-segments".format(area["id"]), stats["segmentsTotal"] > 0, stats["segmentsTotal"])
        add_check(checks, "{}-explore".format(area["id"]), stats["exploreCount"] > 0, stats["exploreCount"])
        add_check(
            checks, "{}-bbox-p95".format(area["id"]), area["segments"]["p95Milliseconds"] < 500,
            area["segments"]["p95Milliseconds"],
        )
        add_check(
            checks, "{}-feature-limit".format(area["id"]), area["segments"]["returnedCount"] < 10000,
            area["segments"]["returnedCount"],
        )
    for probe in source_area_probes:
        protected = probe["status"] == 200 or (
            probe["status"] == 422 and probe.get("errorCode") in
            ("feature_limit_exceeded", "bbox_area_exceeded")
        )
        add_check(
            checks, "{}-source-area-protection".format(probe["id"]), protected,
            {"status": probe["status"], "errorCode": probe.get("errorCode")},
        )
    add_check(checks, "sample-routes", len(route_results) == 5, len(route_results))
    add_check(
        checks, "coverage-profiles",
        all(len(route["profiles"]) == len(profiles) for route in route_results),
        sum(len(route["profiles"]) for route in route_results),
    )
    add_check(
        checks, "coverage-parameters",
        all(
            profile["coverageProfile"] == coverage_profiles[profile["profile"]]
            and profile["contextRadiusMeters"] == ANALYSIS_CONTEXT_RADIUS_METERS
            for route in route_results for profile in route["profiles"]
        ),
        {"profiles": coverage_profiles, "analysisContextRadiusMeters": ANALYSIS_CONTEXT_RADIUS_METERS},
    )

    contract = {
        "fixture": manifest["name"],
        "fixtureChecksum": expected_checksum,
        "normalizationVersion": "stage1-segments-v1",
        "warmRequests": args.warm_requests,
        "bboxP95TargetMilliseconds": 500,
        "featureLimit": 10000,
        "coverageProfiles": coverage_profiles,
        "customCoverageRadiusMeters": {"minimum": 5, "maximum": 200, "default": 50},
        "analysisContextRadiusMeters": ANALYSIS_CONTEXT_RADIUS_METERS,
    }
    if analysis_version is not None:
        contract["analysisVersion"] = analysis_version
    report = {
        "schemaVersion": 1,
        "status": "passed" if all(check["passed"] for check in checks) else "failed",
        "generatedAt": dt.datetime.now(dt.timezone.utc).isoformat().replace("+00:00", "Z"),
        "environment": {
            "os": platform.system(),
            "release": platform.release(),
            "architecture": platform.machine(),
            "apiUrl": api_url,
        },
        "contract": contract,
        "geoVersion": version,
        "geoVersionLatencyMilliseconds": rounded(version_ms),
        "viewports": viewports,
        "sourceAreaProbes": source_area_probes,
        "routes": route_results,
        "checks": checks,
    }
    output = root / args.output
    published_output = publish_report(report, output)
    print(json.dumps({
        "status": report["status"],
        "output": str(published_output),
        "viewports": [{
            "id": area["id"],
            "segments": area["segments"]["returnedCount"],
            "p95Milliseconds": area["segments"]["p95Milliseconds"],
            "responseBytes": area["segments"]["responseBytes"],
        } for area in viewports],
    }, ensure_ascii=False, indent=2))
    if report["status"] != "passed":
        raise SystemExit(1)


if __name__ == "__main__":
    main()
