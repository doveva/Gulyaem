# Stage 3 — Recommended Implementation Plan

# Stage 3.1 — Domain schema + actor context

Deliver:

```text
ActorContext
DEVELOPMENT_ACTOR_ID
routes
route_segment_matches
walks
progress/delta/state tables
migrations
repository boundaries
```

Tests:

- migrations;
- constraints;
- actor ownership;
- lifecycle invalid rows where enforced.

Done when schema and actor context exist without product behavior yet.

---

# Stage 3.2 — Preview fingerprint + route materialization

Extend Stage 2 preview with opaque fingerprint.

Implement:

```text
RouteMaterializer
```

Flow:

```text
waypoints
→ RoutePreview recompute
→ fingerprint compare
→ persist Route + matches
```

Add `POST /walks` creation idempotency via `clientRequestId`.

Done when:

- browser geometry is ignored;
- stale preview is rejected;
- duplicate create retry returns same Walk;
- Route provenance is persisted.

---

# Stage 3.3 — Walk lifecycle

Implement:

```text
GET walk
start
finish
cancel
```

State transitions in application service, not handler.

Done when:

- server timestamps are stable under retry;
- invalid transitions are rejected;
- no exploration mutation occurs.

---

# Stage 3.4 — Route Review and correction

Implement:

```text
PUT /walks/{id}/route
```

Allowed only DRAFT/REVIEW.

Reuse RouteMaterializer and Stage 2 preview.

Correction atomically replaces Route snapshot and increments revision.

Done when:

- ACTIVE correction fails;
- REVIEW correction changes persisted route;
- old matches are replaced;
- COMPLETED route cannot mutate.

---

# Stage 3.5 — Exploration completion core

Implement `ExplorationCompletionService`.

Transaction:

```text
lock Walk
→ validate current geo state
→ read route matches
→ classify NEW/REVISITED
→ progress upsert
→ delta rows
→ district deltas
→ finalize route
→ complete Walk
```

Done when:

- first walk creates progress;
- overlapping second walk only adds truly new segments;
- retry is idempotent;
- transaction rollback leaves no partial progress.

---

# Stage 3.6 — Personal exploration reads

Implement:

```text
GET city exploration
GET explored segments bbox
```

Add district progress calculation with clipped lengths.

Done when `/map` can render persistent explored overlay after reload.

---

# Stage 3.7 — Rebuildability

Implement reusable:

```text
ExplorationRebuilder
```

and executable:

```text
cmd/exploration-rebuild
```

Suggested invocation:

```text
make exploration-rebuild
```

with actor/city configuration.

Done when validation can:

1. record expected current progress;
2. clear materialized current progress;
3. rebuild from completed Walk geometry;
4. compare equivalent segment set/statistics.

---

# Stage 3.8 — Frontend full flow

Extend `/map`:

```text
Start
Active
Finish
Review
Correct
Complete
Summary
Updated map
```

Add durable activeWalkId refresh recovery.

Done when one browser can complete full flow with real backend/Valhalla/PostGIS.

---

# Stage 3.9 — Validation and freeze

Run:

- new/revisited scenarios;
- no-new scenario;
- partial-only scenario;
- ROUTABLE_ONLY scenario;
- correction scenario;
- concurrent retry;
- reload recovery;
- rebuild;
- district progression;
- desktop + physical mobile.

Measure:

```text
materialization p50/p95
lifecycle p50/p95
completion p50/p95
exploration read p50/p95
rebuild duration
```

Resolve proposed ADR statuses.

Create:

```text
docs/stage 3/validation-report.md
```

Only then move to Stage 4.
