# AGENTS.md — ГуляЕм Stage 2

## Mission

Реализовать только **Stage 2 — Manual Route & Exploration Preview**.

Stage 1 geo core считается frozen. Не переписывать segmentation, WalkabilityProfile или coverage semantics без отдельного ADR и фактического дефекта.

## Hard scope

Реализуем:

1. production-shaped Valhalla adapter;
2. reproducible Valhalla dev runtime;
3. routing dataset compatibility with current `GeoDataVersion`;
4. stateless route-preview application service;
5. manual waypoint editing;
6. pedestrian route generation;
7. reuse/refactor Stage 1 route analysis for arbitrary generated geometry;
8. route-to-StreetSegment matching;
9. Balanced coverage preview;
10. `/map` route-builder UI;
11. route/distance/duration display;
12. potential exploration visualization;
13. controlled routing/analysis errors;
14. observability and performance measurements;
15. mobile + desktop validation.

## Hard non-goals

DO NOT implement unless requirements are explicitly changed:

- authentication;
- User;
- UserStreetProgress;
- persistence of route previews;
- `Route` repository as a product entity;
- Walk lifecycle;
- Start Walk / Finish Walk;
- GPS capture;
- GPX product import;
- external POI;
- Places / Visits / Photos;
- Sharing;
- Recommendations / route generation by duration;
- route alternatives UI;
- Social;
- vector tiles solely as speculative optimization;
- ML;
- Redis;
- microservices;
- Kubernetes.

## Critical semantic rule

Stage 2 has no user history.

Do not expose product concepts such as:

```text
newMeters
alreadyExplored
newStreetRatio
districtProgress
```

The Stage 2 concept is:

```text
Potential Exploration Coverage
```

Stage 3 will compare this coverage with `UserStreetProgress`.

## Fixed Stage 1 decisions

Treat as frozen inputs:

- Valhalla is the Stage 2 routing engine.
- Routing engine IDs are diagnostic only.
- StreetSegment IDs remain internal domain identity.
- topology-first segmentation;
- no default artificial max-length split;
- WalkabilityProfile v1;
- Balanced coverage:
  - radius 50 m;
  - ratio 0.6;
  - min 15 m;
  - max 80 m;
- grade-aware local coverage;
- `ROUTABLE_ONLY` never contributes exploration;
- bbox + GeoJSON remains initial background-network delivery.

## Architecture rule

Expected high-level flow:

```text
HTTP
 ↓
Route Preview Service
 ├─→ Routing Engine Port
 │    └─→ Valhalla Adapter
 │
 └─→ Geo Route Analyzer
      └─→ PostGIS / StreetSegment
```

Routing code MUST NOT own exploration rules.

Geo matching MUST NOT depend on Valhalla edge/tile IDs.

## Stage 1 route-analysis refactor

Current Stage 1 `routeanalysis.Service` loads sample fixtures.

Stage 2 production route preview MUST NOT depend on sample-route fixture files.

Extract/reuse an analyzer that can operate on arbitrary GeoJSON route geometry:

```text
Analyzer(repository)
    AnalyzeGeometry(...)
```

The Stage 1 fixture/debug service may wrap the same analyzer.

Do not duplicate matching/coverage algorithms.

## Routing dataset compatibility

Valhalla graph and current internal GeoDataVersion should come from the same source dataset.

Implement explicit metadata for the routing graph, including at least:

```text
engine
engineVersion
sourceChecksum
profile
builtAt?
```

Before generating a preview, compare routing dataset checksum with current READY `GeoDataVersion.source_checksum`.

Mismatch must be explicit. Do not silently route against one graph and analyze against another.

## API behavior

Primary endpoint:

```text
POST /api/v1/route-previews
```

Route preview is stateless and MUST NOT create a persistent `Route` or `Walk`.

## Frontend behavior

Product route:

```text
/map
```

Engineering route remains:

```text
/debug/geo
```

Do not turn `/debug/geo` into the product flow.

Avoid firing routing requests continuously during pointer movement. Recalculate on discrete edits such as:

- waypoint added;
- waypoint removed;
- waypoint reordered;
- marker drag end.

Discard stale responses.

## Testing

Required:

- Valhalla adapter contract tests;
- dataset checksum compatibility tests;
- route preview service tests;
- real PostGIS integration tests;
- API tests;
- frontend interaction tests;
- Playwright route-builder flow;
- manual validation on dense center / regular urban / park environment.

## Coding behavior

Prefer reuse of Stage 1 geo algorithms over replacement.

If Stage 2 evidence suggests a Stage 1 frozen decision is wrong, document the defect and propose an ADR instead of silently changing semantics.
