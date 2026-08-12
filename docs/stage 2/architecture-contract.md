# Stage 2 — Architecture Contract

## Fixed decisions

### Geo core

Stage 1 geo semantics are frozen:

```text
StreetSegment
GeoDataVersion
EXPLORE / ROUTABLE_ONLY / IGNORE
topology-first segmentation
grade-aware matching
Balanced coverage
```

Do not rewrite them as part of route-builder implementation.

### Routing

Routing engine:

```text
Valhalla
```

Routing engine owns:

- pedestrian path generation;
- route geometry;
- distance/duration.

Routing engine does NOT own:

- StreetSegment identity;
- exploration classification;
- coverage state.

### Route preview

`RoutePreview` is stateless in Stage 2.

Do not create persistent:

```text
Route
Walk
Waypoint
RouteSegmentMatch
```

tables merely to support preview.

### Frontend

Product interaction lives at:

```text
/map
```

Engineering tooling remains:

```text
/debug/geo
```

### Personal exploration

There is no `UserStreetProgress` in Stage 2.

Potential coverage != new user progress.

---

## Module boundaries

Recommended logical structure:

```text
internal/
  routing/
    preview/
    port/
  geo/
    routeanalysis/
  platform/
    routing/
      valhalla/
```

Exact directories may differ, but responsibilities should not.

### `routing/preview`

Application orchestration:

```text
validate
geo-version compatibility
route
analyze
compose response
```

### `routing/port`

Engine-neutral contracts.

### `platform/routing/valhalla`

Valhalla HTTP-specific request/response handling.

### `geo/routeanalysis`

Owns route ↔ StreetSegment matching and coverage.

No Valhalla-specific code.

---

## Allowed dependency direction

```text
HTTP transport
     ↓
RoutePreview application service
     ├────────→ RoutingEngine port
     │              ↑
     │        Valhalla adapter
     │
     └────────→ Geo RouteAnalyzer
                    ↓
                 PostGIS
```

Forbidden:

```text
geo routeanalysis → Valhalla
StreetSegment → routing edge ID
React → Valhalla directly
```

Frontend always calls Go backend.

---

## GeoDataVersion compatibility invariant

For every successful preview:

```text
route_source_checksum
==
analysis_geo_source_checksum

resolved_geo_data_version_id
==
candidate_and_coverage_query_geo_data_version_id
```

Resolve the current READY version once before routing, then pass that concrete version ID through
the analyzer and every spatial repository query. Queries must filter by
`street_segments.geo_data_version_id`, not by whichever version has `status='READY'` at query time.
If the pinned version becomes `SUPERSEDED` during the request, analysis continues against its
retained segments. If compatibility cannot be established, do not return a normal successful
preview.

---

## Stage 1 analyzer reuse

Refactor fixture loading away from analysis core if required.

Preferred concept:

```text
Analyzer
  AnalyzeGeometryForVersion(version, ...)

FixtureService
  Routes(...)
  AnalyzeFixture(...)
```

Both debug fixtures and Stage 2 generated routes use the same Analyzer.

---

## API contract rule

The API presents application concepts:

```text
routing
explorationPreview
warnings
```

It does not return raw Valhalla JSON.

---

## Change policy

If Stage 2 discovers a bug in frozen Stage 1 semantics:

1. reproduce it;
2. add a regression test;
3. document it;
4. propose/update ADR;
5. then modify semantics.

Do not tune segmentation/coverage opportunistically inside route-preview code.
