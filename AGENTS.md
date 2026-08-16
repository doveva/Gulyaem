# AGENTS.md — ГуляЕм Stage 3

## Mission

Реализовывать и стабилизировать только **Stage 3 — Exploration Core** по документам
`docs/stage 3/`. Stage 1 geo core и Stage 2 routing/preview semantics считаются frozen.

## Hard scope

- server-side Route materialization из повторно вычисленного preview;
- opaque versioned `previewFingerprint`;
- actor-scoped Route, Walk, progress, state и immutable exploration delta;
- lifecycle `DRAFT → ACTIVE → REVIEW → COMPLETED` и cancellation;
- correction только в DRAFT/REVIEW;
- атомарный и идемпотентный completion;
- только `COMPLETED EXPLORE` влияет на progress;
- clipped district progress;
- current exploration reads и bbox GeoJSON;
- rebuild из final geometry COMPLETED Walk;
- `/map` full flow, summary и reload recovery;
- server-side development actor из `DEVELOPMENT_ACTOR_ID`.

## Hard non-goals

- authentication/accounts;
- GPS/GPX capture;
- Places, Visits, Photos;
- Sharing, Recommendations, Social;
- accumulation of PARTIAL coverage;
- completed Walk edit/delete or Walk history UI;
- background job infrastructure;
- user exclusion storage/UI;
- Redis, microservices, Kubernetes.

## Frozen semantics

- Valhalla and Stage 2 routing port;
- StreetSegment IDs are internal identity; Valhalla IDs are diagnostic only;
- topology-first segmentation and WalkabilityProfile v1;
- coverage profiles: Strict 50 m, Balanced 100 m / 0.4 / 15–80 m, Generous 200 m;
- grade-aware local coverage;
- `ROUTABLE_ONLY` never contributes exploration;
- bbox + GeoJSON map delivery.

## Architecture

```text
HTTP + ActorContext
  ↓
Walks ─→ Stage 2 RoutePreview ─→ Valhalla + RouteAnalyzer
  ↓
Exploration completion/read/rebuild
  ↓
PostgreSQL/PostGIS
```

Routing does not own exploration rules. Exploration does not call Valhalla. Client geometry,
segment IDs, progress values and actor IDs are never authoritative.

## Correctness rules

- completion and progress/delta/finalization are one transaction;
- lock persistent state and enforce uniqueness; frontend suppression is not correctness;
- completion requires current Route `GeoDataVersion` and compatible `ExplorationState`;
- PARTIAL never accumulates; ROUTABLE_ONLY/IGNORE never create progress;
- district lengths use clipped intersection geometry;
- historical delta snapshots are immutable;
- rebuild uses final Route geometry and publishes current progress atomically.

## Testing

Maintain Go unit/API tests, real PostGIS integration tests including concurrent completion, frontend
unit/type/lint checks, Playwright first-Walk and reload flows, migrations, docs-as-code and Compose
validation. Manual dense-center, regular-urban, park and physical-mobile validation remains required
before the stage is frozen.
