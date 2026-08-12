# Stage 3 — API Contract

Exact Go names may differ. Semantics are normative for Stage 3.

# 1. Extension to Stage 2 route preview

Successful:

```http
POST /api/v1/route-previews
```

adds:

```json
{
  "previewFingerprint": "sha256-or-opaque-versioned-value"
}
```

Existing Stage 2 fields remain.

Client must treat fingerprint as opaque.

# 2. Create Walk from reviewed preview

```http
POST /api/v1/walks
Content-Type: application/json
```

Request:

```json
{
  "clientRequestId": "uuid",
  "cityId": "uuid",
  "profile": "pedestrian",
  "expectedPreviewFingerprint": "...",
  "waypoints": [
    {"lat": 59.935, "lon": 30.305},
    {"lat": 59.940, "lon": 30.325}
  ]
}
```

Actor is NOT in body.

Server recomputes route preview and verifies fingerprint.

Response:

```http
201 Created
```

```json
{
  "walk": {
    "id": "uuid",
    "status": "DRAFT",
    "routeId": "uuid",
    "createdAt": "..."
  },
  "route": {
    "id": "uuid",
    "revision": 1,
    "geoDataVersionId": "uuid",
    "distanceMeters": 4200,
    "estimatedDurationSeconds": 3600,
    "geometry": {},
    "waypoints": []
  }
}
```

Retry with same `clientRequestId` returns the same materialized Walk.

# 3. Stale preview

If recomputed preview differs:

```http
409 Conflict
```

```json
{
  "code": "route_preview_stale",
  "message": "route preview changed and must be reviewed again"
}
```

Frontend refreshes Stage 2 preview.

# 4. Get Walk

```http
GET /api/v1/walks/{walkId}
```

Returns actor-owned Walk and current Route.

If COMPLETED, response may include stored exploration summary.

No Walk list endpoint is required.

# 5. Start

```http
POST /api/v1/walks/{walkId}/start
```

Expected:

```text
DRAFT → ACTIVE
```

Response includes server `startedAt`.

Repeated request while ACTIVE returns current state.

Invalid terminal/state transition:

```http
409 walk_invalid_state
```

# 6. Finish

```http
POST /api/v1/walks/{walkId}/finish
```

Expected:

```text
ACTIVE → REVIEW
```

Response includes:

```text
finishedAt
durationSeconds
```

No exploration delta yet.

Repeated request while REVIEW returns current state.

# 7. Correct final route

```http
PUT /api/v1/walks/{walkId}/route
```

Request:

```json
{
  "profile": "pedestrian",
  "expectedPreviewFingerprint": "...",
  "waypoints": []
}
```

Allowed Walk status:

```text
DRAFT
REVIEW
```

Success replaces trusted Route snapshot and increments `revision`.

Errors:

```text
409 route_preview_stale
409 walk_route_not_editable
```

# 8. Complete

```http
POST /api/v1/walks/{walkId}/complete
```

No client-provided segment IDs or progress values.

Success:

```json
{
  "walk": {
    "id": "uuid",
    "status": "COMPLETED",
    "startedAt": "...",
    "finishedAt": "...",
    "completedAt": "...",
    "durationSeconds": 5400,
    "distanceMeters": 7200
  },
  "exploration": {
    "geoDataVersionId": "uuid",
    "newSegmentsCount": 17,
    "revisitedSegmentsCount": 32,
    "newNetworkLengthMeters": 3100,
    "newSegments": {
      "type": "FeatureCollection",
      "features": []
    },
    "districts": [
      {
        "districtId": "uuid",
        "name": "Центральный район",
        "percentageBefore": 0.31,
        "percentageAfter": 0.34,
        "newLengthMeters": 2800
      }
    ]
  }
}
```

Retry after successful completion returns same persisted result.

# 9. Completion conflicts

## Route geo version stale

```http
409
```

```json
{
  "code": "walk_route_geo_version_stale",
  "message": "final route must be refreshed against current geo data before completion"
}
```

## Actor exploration stale

```http
409
```

```json
{
  "code": "exploration_rebuild_required",
  "message": "personal exploration state must be rebuilt for current geo data"
}
```

## Wrong Walk status

```http
409
```

```json
{
  "code": "walk_invalid_state",
  "message": "walk must be in REVIEW before completion"
}
```

# 10. Cancel

```http
POST /api/v1/walks/{walkId}/cancel
```

Allowed:

```text
DRAFT
ACTIVE
REVIEW
```

Success transitions to `CANCELLED`.

Repeated cancel returns current state.

# 11. Current city exploration

```http
GET /api/v1/cities/{cityId}/exploration
```

Conceptual response:

```json
{
  "geoDataVersion": {
    "id": "uuid"
  },
  "state": {
    "status": "READY",
    "updatedAt": "..."
  },
  "city": {
    "exploredLengthMeters": 12340,
    "eligibleLengthMeters": 920000,
    "percentage": 0.0134,
    "exploredSegmentsCount": 183
  },
  "districts": [
    {
      "districtId": "uuid",
      "name": "Центральный район",
      "exploredLengthMeters": 8030,
      "eligibleLengthMeters": 85000,
      "percentage": 0.0945
    }
  ]
}
```

# 12. Explored viewport

```http
GET /api/v1/cities/{cityId}/exploration/segments?bbox=west,south,east,north
```

Response:

```text
GeoJSON FeatureCollection
```

Contains actor-completed current-version `EXPLORE` segments.

Use Stage 1 bbox validation/feature limits.

# 13. Exploration read when stale

If state references old geo version:

```http
409 exploration_rebuild_required
```

Do not return an empty current overlay as if actor had explored nothing.

# 14. Actor/ownership errors

Stage 3 transport has a configured actor.

Resource owned by a different actor should use one consistent policy.

Recommended:

```text
404 not_found
```

to avoid revealing existence.

# 15. Malformed/body limits

Reuse Stage 2 conventions:

- strict JSON;
- unknown fields rejected;
- request size limits;
- UUID validation;
- normalized error body.

# 16. No client-authoritative exploration

The following are never accepted from browser as trusted input:

```text
streetSegmentIds
newSegments
district percentages
route geometry
coverage status
GeoDataVersion override
actorId
```
