# Stage 3 — Acceptance Criteria

## Status legend

- `[x]` — implemented and supported by an automated test or a mechanically verifiable
  schema/repository invariant in the current tree.
- `[ ]` — acceptance evidence is still missing; planned behavior or implementation alone is not
  treated as sufficient where the criterion explicitly requires a regression, performance result,
  manual run or product-comprehension validation.

**Snapshot (2026-08-14):** 161 proven, 19 pending. The pending set is intentionally concentrated
in performance evidence, manual/product validation and final persistence/freeze readiness.

Pending evidence breakdown:

- performance measurements/budgets — 9 (`AC-136`–`AC-140`, `AC-143`–`AC-146`);
- manual/product flow — 8 (`AC-161`–`AC-167`, `AC-176`);
- final integrated persistence/freeze proof — 2 (`AC-178`, `AC-180`).

## Actor context

- [x] AC-001 Application services receive explicit actor context.
- [x] AC-002 HTTP client cannot choose arbitrary actor ID.
- [x] AC-003 development actor is configured server-side.
- [x] AC-004 actor-owned repository reads/writes are scoped by actor.
- [x] AC-005 cross-actor resource isolation is integration-tested.

## Persistence schema

- [x] AC-006 Route persistence exists.
- [x] AC-007 Route stores pinned GeoDataVersion.
- [x] AC-008 Route stores ordered waypoints.
- [x] AC-009 Route stores final LineString geometry.
- [x] AC-010 Route stores normalized geometry.
- [x] AC-011 Route stores routing provenance.
- [x] AC-012 Route stores analysis/coverage provenance.
- [x] AC-013 route coverage snapshot is persisted by StreetSegment.
- [x] AC-014 Walk persistence exists.
- [x] AC-015 progress/delta/state persistence exists.
- [x] AC-016 migrations and down migrations are present.
- [x] AC-017 spatial/repository integration tests use real PostgreSQL/PostGIS.

## Preview fingerprint / materialization

- [x] AC-018 Stage 2 preview returns opaque fingerprint.
- [x] AC-019 fingerprint changes when relevant materialization input changes.
- [x] AC-020 client cannot supply authoritative route geometry to create Walk.
- [x] AC-021 Walk creation recomputes server-side route preview.
- [x] AC-022 fingerprint mismatch returns 409 `route_preview_stale`.
- [x] AC-023 stale materialization persists neither Route nor Walk.
- [x] AC-024 successful materialization persists exactly the server-computed Route.
- [x] AC-025 materialized route matches current compatible routing/geo data.
- [x] AC-026 `clientRequestId` makes Walk creation retry-idempotent.
- [x] AC-027 same clientRequestId with incompatible payload is rejected.

## Walk lifecycle

- [x] AC-028 new Walk starts as DRAFT.
- [x] AC-029 DRAFT can transition to ACTIVE.
- [x] AC-030 ACTIVE can transition to REVIEW.
- [x] AC-031 REVIEW can transition to COMPLETED.
- [x] AC-032 DRAFT/ACTIVE/REVIEW can transition to CANCELLED.
- [x] AC-033 COMPLETED is terminal.
- [x] AC-034 CANCELLED is terminal.
- [x] AC-035 start timestamp is server-owned and stable under retry.
- [x] AC-036 finish timestamp is server-owned and stable under retry.
- [x] AC-037 finish does not update exploration.
- [x] AC-038 cancellation does not update exploration.
- [x] AC-039 invalid transitions return controlled conflict.

## Route correction

- [x] AC-040 route correction is allowed in DRAFT.
- [x] AC-041 route correction is allowed in REVIEW.
- [x] AC-042 route correction is rejected in ACTIVE.
- [x] AC-043 route correction is rejected in COMPLETED/CANCELLED.
- [x] AC-044 correction uses server-side preview recomputation.
- [x] AC-045 correction verifies preview fingerprint.
- [x] AC-046 correction atomically replaces coverage snapshot.
- [x] AC-047 route revision increments.
- [x] AC-048 final Route becomes immutable on completion.

## Completion correctness

- [x] AC-049 completion requires REVIEW.
- [x] AC-050 completion locks Walk row or equivalent persistent concurrency guard.
- [x] AC-051 completion requires final Route geo version to be current.
- [x] AC-052 stale Route version returns explicit conflict.
- [x] AC-053 completion requires current/compatible exploration state.
- [x] AC-054 rebuild-required state returns explicit conflict.
- [x] AC-055 only COMPLETED EXPLORE coverage affects personal progress.
- [x] AC-056 PARTIAL does not create persistent progress.
- [x] AC-057 multiple PARTIAL Walk do not sum into completion.
- [x] AC-058 ROUTABLE_ONLY never creates progress.
- [x] AC-059 IGNORE never creates progress.
- [x] AC-060 one segment increments visit_count at most once per Walk.
- [x] AC-061 NEW classification is relative to pre-transaction actor state.
- [x] AC-062 REVISITED classification is relative to pre-transaction actor state.
- [x] AC-063 zero-new Walk completes successfully.
- [x] AC-064 completion persists one ExplorationDelta.
- [x] AC-065 completion persists segment-level delta.
- [x] AC-066 completion persists district delta snapshots.
- [x] AC-067 route finalization and Walk completion are in same transaction as progress mutation.
- [x] AC-068 injected failure rolls all completion changes back.
- [x] AC-069 repeated completed request returns same result.
- [x] AC-070 repeated completion does not increment visit_count.
- [x] AC-071 concurrent completion creates one logical mutation.

## District progress

- [x] AC-072 denominator includes only EXPLORE network.
- [x] AC-073 district contribution clips segment geometry to district boundary.
- [x] AC-074 full segment length is not double-counted merely because it intersects a district.
- [x] AC-075 WalkDistrictDelta pins DistrictDataVersion.
- [x] AC-076 percentage_before/after is stored as immutable completion snapshot.
- [x] AC-077 current district progress can be read for actor.

## Exploration state/read model

- [x] AC-078 actor progress persists after process/browser restart.
- [x] AC-079 current exploration state pins represented GeoDataVersion.
- [x] AC-080 version mismatch is detected explicitly.
- [x] AC-081 stale state is not displayed as valid zero/current progress.
- [x] AC-082 city exploration read returns explored/eligible length and percentage.
- [x] AC-083 current district progress list is available.
- [x] AC-084 bbox explored-segment GeoJSON endpoint exists.
- [x] AC-085 bbox explored endpoint reuses viewport limits.
- [x] AC-086 explored overlay contains only current actor/current-version completed EXPLORE segments.
- [x] AC-087 future actor-exclusion denominator seam is documented/not globally baked into progress rows.

## Rebuildability

- [x] AC-088 rebuild service exists.
- [x] AC-089 rebuild CLI/executable exists.
- [x] AC-090 rebuild uses COMPLETED Walk final Route geometry.
- [x] AC-091 rebuild pins one target current GeoDataVersion.
- [x] AC-092 rebuild reuses RouteAnalyzer rather than RouteSegmentMatch as sole truth.
- [x] AC-093 rebuild reconstructs NEW current progress without mutating historical Walk deltas.
- [x] AC-094 rebuild replaces visible progress atomically.
- [x] AC-095 rebuild aborts publication if target geo version changed.
- [x] AC-096 rebuild is idempotent.
- [x] AC-097 rebuild equivalence test compares segment set.
- [x] AC-098 rebuild equivalence test compares visit counts / first-last semantics.
- [x] AC-099 rebuild equivalence test compares city/district current statistics.

## Walk API

- [x] AC-100 `POST /api/v1/walks`.
- [x] AC-101 `GET /api/v1/walks/{id}`.
- [x] AC-102 `POST /api/v1/walks/{id}/start`.
- [x] AC-103 `POST /api/v1/walks/{id}/finish`.
- [x] AC-104 `PUT /api/v1/walks/{id}/route`.
- [x] AC-105 `POST /api/v1/walks/{id}/complete`.
- [x] AC-106 `POST /api/v1/walks/{id}/cancel`.
- [x] AC-107 no product Walk-list/history endpoint is introduced.
- [x] AC-108 unknown fields/body limits follow existing HTTP conventions.
- [x] AC-109 ownership/not-found policy is consistent.

## Frontend flow

- [x] AC-110 Stage 2 preview exposes Start Walk CTA.
- [x] AC-111 Start materializes DRAFT then starts ACTIVE.
- [x] AC-112 partial create/start failure can be retried without duplicate Walk.
- [x] AC-113 ACTIVE screen shows server-based elapsed time.
- [x] AC-114 ACTIVE screen does not fake GPS/current position.
- [x] AC-115 Finish transitions to REVIEW.
- [x] AC-116 REVIEW shows final route.
- [x] AC-117 REVIEW can open manual correction.
- [x] AC-118 corrected route can be saved back to Walk.
- [x] AC-119 Complete waits for backend result without optimistic progress.
- [x] AC-120 Walk Summary highlights newly explored segments.
- [x] AC-121 Walk Summary shows new network length.
- [x] AC-122 Walk Summary shows district before→after where changed.
- [x] AC-123 zero-new Walk has valid summary UX.
- [x] AC-124 returning to map refreshes persistent exploration overlay.
- [x] AC-125 page reload after completion shows same exploration from backend.
- [x] AC-126 activeWalkId is persisted locally for recovery.
- [x] AC-127 ACTIVE reload restores backend Walk.
- [x] AC-128 REVIEW reload restores backend Walk.
- [x] AC-129 terminal/not-found Walk clears stale local pointer.
- [x] AC-130 Cancel clears active local pointer without progress change.

## Error UX

- [x] AC-131 route_preview_stale is recoverable.
- [x] AC-132 walk_route_geo_version_stale is explicit.
- [x] AC-133 exploration_rebuild_required is explicit.
- [x] AC-134 invalid lifecycle transition is explicit.
- [x] AC-135 errors do not discard recoverable REVIEW waypoint state.

## Observability/performance

- [ ] AC-136 Walk materialization latency is measured.
- [ ] AC-137 lifecycle transition latency is measured.
- [ ] AC-138 completion latency is measured.
- [ ] AC-139 exploration read latency is measured.
- [ ] AC-140 rebuild duration/walk count/segment count are measured.
- [x] AC-141 exact route/waypoint geometry is not logged by default.
- [x] AC-142 completion logs new/revisited counts without sensitive geometry.
- [ ] AC-143 materialization representative p95 is within 1–2s or disposition documented.
- [ ] AC-144 lifecycle p95 <500ms or disposition documented.
- [ ] AC-145 completion p95 <2s or bottleneck/disposition documented.
- [ ] AC-146 exploration bbox p95 <500ms or disposition documented.

## Automated validation

- [x] AC-147 Go unit suite passes.
- [x] AC-148 Go vet/static checks pass.
- [x] AC-149 real PostGIS Stage 3 integration suite passes.
- [x] AC-150 transaction rollback regression passes.
- [x] AC-151 concurrent completion regression passes.
- [x] AC-152 version-stale regression passes.
- [x] AC-153 rebuild equivalence regression passes.
- [x] AC-154 frontend lint/typecheck/unit tests pass.
- [x] AC-155 Playwright full first-Walk E2E passes.
- [x] AC-156 Playwright correction E2E passes.
- [x] AC-157 Playwright reload-recovery E2E passes.
- [x] AC-158 Playwright no-new second Walk E2E passes.
- [x] AC-159 docs-as-code index/check passes.
- [x] AC-160 docker compose config/check passes.

## Manual/product validation

- [ ] AC-161 dense-center full flow manually validated.
- [ ] AC-162 regular-urban full flow manually validated.
- [ ] AC-163 park full flow manually validated.
- [ ] AC-164 physical mobile full flow manually validated.
- [ ] AC-165 Walk Summary is understandable without debug knowledge.
- [ ] AC-166 distinction between route distance and new-network length is understandable.
- [ ] AC-167 user understands exploration changes only after final confirmation.

## Scope discipline

- [x] AC-168 no authentication/account system.
- [x] AC-169 no GPS/GPX capture.
- [x] AC-170 no Places/Visits/Photos.
- [x] AC-171 no Sharing/Recommendations/Social.
- [x] AC-172 no cumulative PARTIAL accumulation.
- [x] AC-173 no completed Walk edit/delete product flow.
- [x] AC-174 no background job infrastructure solely for rebuild.
- [x] AC-175 no user exclusion UI/storage unless scope is explicitly revised.

## Final DoD

- [ ] AC-176 complete end-to-end exploration loop works from arbitrary manual route.
- [x] AC-177 completion retry/concurrency cannot duplicate progress.
- [ ] AC-178 persistent exploration survives reload/process restart.
- [x] AC-179 rebuild reconstructs equivalent current materialized progress from COMPLETED Walk.
- [ ] AC-180 Stage 3 validation report is complete and Stage 4 readiness is explicit.
