# Stage 3 — Validation Report

**Status:** Stage 3 automated flow passes; targeted acceptance gaps and runtime field validation pending  
**Date:** 2026-08-14

## Implemented evidence

- migrations through `000006_stage3_ownership_integrity` apply to real PostgreSQL/PostGIS;
- opaque `stage3-preview-fingerprint-v1` covers route/materialization provenance;
- route-analysis v2 pins the revised Strict/Balanced/Generous radii (`50/100/200 м`) and the
  Balanced ratio (`0.4`) in preview fingerprints;
- ADR-0014 validation has a separate `coverage_v2.py` runner/output; the frozen Stage 1 runner
  remains on `35/50/100 м` and cannot replace its accepted report with a failed v2 result;
- actor-scoped Route/Walk persistence, correction and lifecycle;
- advisory + row locking and uniqueness protect concurrent completion;
- shared GeoDataVersion locks prevent publisher races during materialization, correction and completion;
- a composite FK enforces Walk/Route actor-city ownership and version-checked match inserts reject
  StreetSegments from another GeoDataVersion;
- real PostGIS regression verifies concurrent retry, NEW→REVISITED and visit-count idempotency;
- a late PostgreSQL trigger failure on the final `REVIEW→COMPLETED` update proves rollback of
  progress, exploration/district deltas, state publication, Walk finalization and Route finalization;
- real PostGIS regressions verify a geo publisher blocks until transaction commit and stale versions persist nothing;
- district before/after uses clipped `ST_Intersection` length;
- exploration city/bbox reads reject stale state;
- rebuild re-analyzes final COMPLETED Walk geometry and atomically publishes progress;
- real PostGIS non-empty rebuild equivalence preserves segment set, visits, first/last semantics,
  city/district metrics and immutable historical deltas across three COMPLETED Walk;
- `/map` implements explicit DRAFT save, active Walk, mandatory review, correction, summary,
  current district progress/refresh, explored overlay and reload recovery.
- Stage 3 Playwright covers first completion, REVIEW correction/save/complete, zero-new repeated Walk,
  explicit DRAFT save, district refresh, ACTIVE reload recovery, builder errors and the mobile builder.

## Automated checks

```text
CGO_ENABLED=0 go test ./...                         PASS
CGO_ENABLED=0 go vet ./...                          PASS
real PostGIS Stage 3 completion + rollback tests   PASS
real PostGIS non-empty rebuild equivalence test    PASS
npm run lint                                       PASS
npm test                                           PASS (27 tests)
npm run build                                      PASS
npm run test:e2e -- e2e/route-builder.spec.ts      PASS (9 tests)
npm run test:e2e                                   PASS (12 tests)
docker compose config --quiet                      PASS
golang-migrate up through migration 000006         PASS
```

## Pending before freeze

- full real Valhalla/PostGIS product flow on dense center, regular urban and park routes;
- p50/p95 materialization, lifecycle, completion, read and rebuild measurements;
- physical mobile validation and copy comprehension;
- arbitrary-route product E2E and explicit process-restart persistence evidence.

Stage 4 readiness is therefore not declared yet.
