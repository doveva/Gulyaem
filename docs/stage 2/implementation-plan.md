# Stage 2 — Recommended Implementation Plan

# Stage 2.1 — Promote Valhalla to normal development dependency

Deliver:

```text
Valhalla compose/runtime
graph build command
routing dataset metadata
health check
configuration
```

Reuse Stage 1 pinned engine/version where possible.

Done when:

- a clean developer environment can build/load graph;
- Valhalla returns pedestrian route;
- graph metadata contains source checksum.

---

# Stage 2.2 — Routing port + Valhalla adapter

Deliver:

```text
RoutingEngine interface
RouteRequest
RouteResult
Valhalla HTTP client
normalized errors
timeouts
contract tests
```

No exploration logic here.

Done when adapter can route fixture waypoints without exposing raw Valhalla response.

---

# Stage 2.3 — Refactor reusable RouteAnalyzer

Current Stage 1 fixture service should not be a production dependency.

Extract:

```text
RouteAnalyzer
```

that accepts arbitrary generated geometry and uses existing matching/coverage.

Stage 1 sample route service should delegate to it.

Done when:

- Stage 1 tests remain green;
- sample-route behavior unchanged;
- arbitrary GeoJSON route can be analyzed without loading fixture manifest.

---

# Stage 2.4 — RoutePreview application service

Implement orchestration:

```text
validate
↓
current GeoDataVersion
↓
routing metadata compatibility
↓
Valhalla route
↓
RouteAnalyzer
↓
compose preview
```

Add derived Stage 2 metrics such as completed/partial counts.

Done when service has no HTTP dependency and no persistence.

---

# Stage 2.5 — RoutePreview HTTP API

Add:

```text
POST /api/v1/route-previews
```

Implement:

- strict JSON decoding;
- body limit;
- normalized errors;
- request/correlation logging;
- no raw Valhalla error leakage.

Done when transport tests cover all expected error classes.

---

# Stage 2.6 — Product `/map` route builder

Build:

```text
Map shell
Waypoint state
Markers
Bottom/side sheet
Route request client
Route layer
Potential coverage layers
Summary
```

Keep `/debug/geo` working.

Done when two-point route works end-to-end.

---

# Stage 2.7 — Route editing

Add:

- intermediate waypoint;
- drag;
- delete;
- reorder;
- clear;
- stale request protection.

Done when rapid edits never restore an older preview.

---

# Stage 2.8 — Performance + compression

Measure:

```text
route generation
route analysis
total preview
GeoJSON payload
compressed payload
mobile render
```

Add HTTP compression for background map data if not already present.

Do not add vector tiles unless evidence crosses ADR reconsideration conditions.

---

# Stage 2.9 — Validation

Run representative manual routes in:

```text
Dense Center
Regular Urban District
Park + Residential
```

Validate:

- route intent;
- waypoint editing;
- visual coverage;
- mobile interaction;
- error states;
- latency.

Produce Stage 2 validation report.

---

# Stage 2.10 — Freeze for Stage 3

Stage 2 should finish with decisions on:

- route-preview API stability;
- route editing UX;
- warning threshold for low matching;
- routing/analysis performance;
- bbox + GeoJSON continuation;
- how Stage 3 materializes a selected preview into persistent Route/Walk.

Do not implement Stage 3 persistence during freeze.
