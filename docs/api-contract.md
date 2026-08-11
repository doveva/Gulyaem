# Stage 2 — API Contract

This document fixes the intended Stage 2 HTTP semantics. Exact Go type names may differ.

# 1. Create route preview

```http
POST /api/v1/route-previews
Content-Type: application/json
```

## Request

```json
{
  "cityId": "10000000-0000-0000-0000-000000000001",
  "profile": "pedestrian",
  "waypoints": [
    {
      "lat": 59.935,
      "lon": 30.325
    },
    {
      "lat": 59.931,
      "lon": 30.340
    }
  ]
}
```

## Validation

- `cityId`: UUID;
- `profile`: Stage 2 accepts only `pedestrian`;
- `waypoints`: 2–10;
- finite coordinates;
- latitude `[-90, 90]`;
- longitude `[-180, 180]`;
- unknown fields rejected.

---

# 2. Success response

```http
200 OK
```

Conceptual response:

```json
{
  "geoDataVersion": {
    "id": "uuid",
    "cityId": "uuid",
    "sourceChecksum": "sha256...",
    "normalizationVersion": "stage1-segments-v1",
    "status": "READY"
  },
  "routing": {
    "engine": "valhalla",
    "profile": "pedestrian",
    "distanceMeters": 4200,
    "durationSeconds": 3600,
    "geometry": {
      "type": "LineString",
      "coordinates": []
    },
    "waypoints": [
      {
        "input": { "lat": 59.935, "lon": 30.325 },
        "resolved": { "lat": 59.9351, "lon": 30.3252 }
      }
    ]
  },
  "explorationPreview": {
    "coverageProfile": {
      "name": "balanced",
      "radiusMeters": 50,
      "coverageRatio": 0.6,
      "minRequiredMeters": 15,
      "maxRequiredMeters": 80
    },
    "normalizedRoute": {
      "type": "MultiLineString",
      "coordinates": []
    },
    "matchedFragments": [],
    "unmatchedFragments": [],
    "coverageSegments": [],
    "metrics": {
      "routeMatchedRatio": 0.99,
      "routeUnmatchedLengthMeters": 42,
      "completedNetworkLengthMeters": 1800,
      "contextExplorableLengthMeters": 9600,
      "completedNetworkRatio": 0.1875,
      "matchedExplorableRouteLengthMeters": 4000,
      "matchedRoutableOnlyRouteLengthMeters": 158,
      "completedSegmentCount": 27,
      "partialSegmentCount": 6
    }
  },
  "warnings": []
}
```

`resolved` waypoint is optional if the routing adapter cannot reliably expose a normalized snap without leaking engine internals.

---

# 3. Coverage segment

Minimum contract:

```json
{
  "segmentId": "uuid",
  "classification": "EXPLORE",
  "geometry": {
    "type": "LineString",
    "coordinates": []
  },
  "lengthMeters": 83.2,
  "coveredMeters": 70.1,
  "requiredMeters": 49.9,
  "status": "COMPLETED",
  "provenance": "DIRECT_AND_RADIUS"
}
```

Possible status:

```text
COMPLETED
PARTIAL
NOT_COVERED
CONNECTOR
```

Product client does not need to render all statuses equally.

---

# 4. Warnings

Warnings are successful-result diagnostics.

Potential Stage 2 warnings:

```text
low_route_match
routing_snap_adjusted
```

Do not put infrastructure failure in warnings if the preview itself cannot be trusted.

---

# 5. Errors

## 400 — malformed JSON / unsupported shape

```json
{
  "code": "invalid_body",
  "message": "request body must be valid JSON"
}
```

## 422 — waypoint validation

```json
{
  "code": "invalid_waypoints",
  "message": "route preview requires between 2 and 10 waypoints"
}
```

## 422 — route not found

```json
{
  "code": "route_not_found",
  "message": "pedestrian route could not be built"
}
```

## 409 — incompatible routing graph

```json
{
  "code": "routing_geo_version_mismatch",
  "message": "routing graph and current geo data version use different source datasets"
}
```

## 503 — routing unavailable

```json
{
  "code": "routing_unavailable",
  "message": "routing service is unavailable"
}
```

## 504 — upstream timeout

```json
{
  "code": "routing_timeout",
  "message": "routing service did not respond in time"
}
```

## 503 — no READY geo version

```json
{
  "code": "geo_data_unavailable",
  "message": "geo data is unavailable"
}
```

---

# 6. No persistence contract

This endpoint is a calculation endpoint.

A successful response does NOT imply that the server created:

- Route;
- Walk;
- route draft;
- user-owned state.

Repeated identical requests may recalculate.

---

# 7. Internal/debug API

Stage 1 sample route endpoints may remain for `/debug/geo`.

Product `/map` must not depend on:

```text
/api/v1/geo/sample-routes
```

---

# 8. Version safety

Successful response includes current `GeoDataVersion`.

This prepares Stage 3 to materialize a selected preview while retaining explicit geo-version context.
