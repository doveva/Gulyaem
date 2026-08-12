# Stage 3 — Acceptance Criteria

## Actor context

- [ ] AC-001 Application services receive explicit actor context.
- [ ] AC-002 HTTP client cannot choose arbitrary actor ID.
- [ ] AC-003 development actor is configured server-side.
- [ ] AC-004 actor-owned repository reads/writes are scoped by actor.
- [ ] AC-005 cross-actor resource isolation is integration-tested.

## Persistence schema

- [ ] AC-006 Route persistence exists.
- [ ] AC-007 Route stores pinned GeoDataVersion.
- [ ] AC-008 Route stores ordered waypoints.
- [ ] AC-009 Route stores final LineString geometry.
- [ ] AC-010 Route stores normalized geometry.
- [ ] AC-011 Route stores routing provenance.
- [ ] AC-012 Route stores analysis/coverage provenance.
- [ ] AC-013 route coverage snapshot is persisted by StreetSegment.
- [ ] AC-014 Walk persistence exists.
- [ ] AC-015 progress/delta/state persistence exists.
- [ ] AC-016 migrations and down migrations are present.
- [ ] AC-017 spatial/repository integration tests use real PostgreSQL/PostGIS.

## Preview fingerprint / materialization

- [ ] AC-018 Stage 2 preview returns opaque fingerprint.
- [ ] AC-019 fingerprint changes when relevant materialization input changes.
- [ ] AC-020 client cannot supply authoritative route geometry to create Walk.
- [ ] AC-021 Walk creation recomputes server-side route preview.
- [ ] AC-022 fingerprint mismatch returns 409 `route_preview_stale`.
- [ ] AC-023 stale materialization persists neither Route nor Walk.
- [ ] AC-024 successful materialization persists exactly the server-computed Route.
- [ ] AC-025 materialized route matches current compatible routing/geo data.
- [ ] AC-026 `clientRequestId` makes Walk creation retry-idempotent.
- [ ] AC-027 same clientRequestId with incompatible payload is rejected.

## Walk lifecycle

- [ ] AC-028 new Walk starts as DRAFT.
- [ ] AC-029 DRAFT can transition to ACTIVE.
- [ ] AC-030 ACTIVE can transition to REVIEW.
- [ ] AC-031 REVIEW can transition to COMPLETED.
- [ ] AC-032 DRAFT/ACTIVE/REVIEW can transition to CANCELLED.
- [ ] AC-033 COMPLETED is terminal.
- [ ] AC-034 CANCELLED is terminal.
- [ ] AC-035 start timestamp is server-owned and stable under retry.
- [ ] AC-036 finish timestamp is server-owned and stable under retry.
- [ ] AC-037 finish does not update exploration.
- [ ] AC-038 cancellation does not update exploration.
- [ ] AC-039 invalid transitions return controlled conflict.

## Route correction

- [ ] AC-040 route correction is allowed in DRAFT.
- [ ] AC-041 route correction is allowed in REVIEW.
- [ ] AC-042 route correction is rejected in ACTIVE.
- [ ] AC-043 route correction is rejected in COMPLETED/CANCELLED.
- [ ] AC-044 correction uses server-side preview recomputation.
- [ ] AC-045 correction verifies preview fingerprint.
- [ ] AC-046 correction atomically replaces coverage snapshot.
- [ ] AC-047 route revision increments.
- [ ] AC-048 final Route becomes immutable on completion.

## Completion correctness

- [ ] AC-049 completion requires REVIEW.
- [ ] AC-050 completion locks Walk row or equivalent persistent concurrency guard.
- [ ] AC-051 completion requires final Route geo version to be current.
- [ ] AC-052 stale Route version returns explicit conflict.
- [ ] AC-053 completion requires current/compatible exploration state.
- [ ] AC-054 rebuild-required state returns explicit conflict.
- [ ] AC-055 only COMPLETED EXPLORE coverage affects personal progress.
- [ ] AC-056 PARTIAL does not create persistent progress.
- [ ] AC-057 multiple PARTIAL Walk do not sum into completion.
- [ ] AC-058 ROUTABLE_ONLY never creates progress.
- [ ] AC-059 IGNORE never creates progress.
- [ ] AC-060 one segment increments visit_count at most once per Walk.
- [ ] AC-061 NEW classification is relative to pre-transaction actor state.
- [ ] AC-062 REVISITED classification is relative to pre-transaction actor state.
- [ ] AC-063 zero-new Walk completes successfully.
- [ ] AC-064 completion persists one ExplorationDelta.
- [ ] AC-065 completion persists segment-level delta.
- [ ] AC-066 completion persists district delta snapshots.
- [ ] AC-067 route finalization and Walk completion are in same transaction as progress mutation.
- [ ] AC-068 injected failure rolls all completion changes back.
- [ ] AC-069 repeated completed request returns same result.
- [ ] AC-070 repeated completion does not increment visit_count.
- [ ] AC-071 concurrent completion creates one logical mutation.

## District progress

- [ ] AC-072 denominator includes only EXPLORE network.
- [ ] AC-073 district contribution clips segment geometry to district boundary.
- [ ] AC-074 full segment length is not double-counted merely because it intersects a district.
- [ ] AC-075 WalkDistrictDelta pins DistrictDataVersion.
- [ ] AC-076 percentage_before/after is stored as immutable completion snapshot.
- [ ] AC-077 current district progress can be read for actor.

## Exploration state/read model

- [ ] AC-078 actor progress persists after process/browser restart.
- [ ] AC-079 current exploration state pins represented GeoDataVersion.
- [ ] AC-080 version mismatch is detected explicitly.
- [ ] AC-081 stale state is not displayed as valid zero/current progress.
- [ ] AC-082 city exploration read returns explored/eligible length and percentage.
- [ ] AC-083 current district progress list is available.
- [ ] AC-084 bbox explored-segment GeoJSON endpoint exists.
- [ ] AC-085 bbox explored endpoint reuses viewport limits.
- [ ] AC-086 explored overlay contains only current actor/current-version completed EXPLORE segments.
- [ ] AC-087 future actor-exclusion denominator seam is documented/not globally baked into progress rows.

## Rebuildability

- [ ] AC-088 rebuild service exists.
- [ ] AC-089 rebuild CLI/executable exists.
- [ ] AC-090 rebuild uses COMPLETED Walk final Route geometry.
- [ ] AC-091 rebuild pins one target current GeoDataVersion.
- [ ] AC-092 rebuild reuses RouteAnalyzer rather than RouteSegmentMatch as sole truth.
- [ ] AC-093 rebuild reconstructs NEW current progress without mutating historical Walk deltas.
- [ ] AC-094 rebuild replaces visible progress atomically.
- [ ] AC-095 rebuild aborts publication if target geo version changed.
- [ ] AC-096 rebuild is idempotent.
- [ ] AC-097 rebuild equivalence test compares segment set.
- [ ] AC-098 rebuild equivalence test compares visit counts / first-last semantics.
- [ ] AC-099 rebuild equivalence test compares city/district current statistics.

## Walk API

- [ ] AC-100 `POST /api/v1/walks`.
- [ ] AC-101 `GET /api/v1/walks/{id}`.
- [ ] AC-102 `POST /api/v1/walks/{id}/start`.
- [ ] AC-103 `POST /api/v1/walks/{id}/finish`.
- [ ] AC-104 `PUT /api/v1/walks/{id}/route`.
- [ ] AC-105 `POST /api/v1/walks/{id}/complete`.
- [ ] AC-106 `POST /api/v1/walks/{id}/cancel`.
- [ ] AC-107 no product Walk-list/history endpoint is introduced.
- [ ] AC-108 unknown fields/body limits follow existing HTTP conventions.
- [ ] AC-109 ownership/not-found policy is consistent.

## Frontend flow

- [ ] AC-110 Stage 2 preview exposes Start Walk CTA.
- [ ] AC-111 Start materializes DRAFT then starts ACTIVE.
- [ ] AC-112 partial create/start failure can be retried without duplicate Walk.
- [ ] AC-113 ACTIVE screen shows server-based elapsed time.
- [ ] AC-114 ACTIVE screen does not fake GPS/current position.
- [ ] AC-115 Finish transitions to REVIEW.
- [ ] AC-116 REVIEW shows final route.
- [ ] AC-117 REVIEW can open manual correction.
- [ ] AC-118 corrected route can be saved back to Walk.
- [ ] AC-119 Complete waits for backend result without optimistic progress.
- [ ] AC-120 Walk Summary highlights newly explored segments.
- [ ] AC-121 Walk Summary shows new network length.
- [ ] AC-122 Walk Summary shows district before→after where changed.
- [ ] AC-123 zero-new Walk has valid summary UX.
- [ ] AC-124 returning to map refreshes persistent exploration overlay.
- [ ] AC-125 page reload after completion shows same exploration from backend.
- [ ] AC-126 activeWalkId is persisted locally for recovery.
- [ ] AC-127 ACTIVE reload restores backend Walk.
- [ ] AC-128 REVIEW reload restores backend Walk.
- [ ] AC-129 terminal/not-found Walk clears stale local pointer.
- [ ] AC-130 Cancel clears active local pointer without progress change.

## Error UX

- [ ] AC-131 route_preview_stale is recoverable.
- [ ] AC-132 walk_route_geo_version_stale is explicit.
- [ ] AC-133 exploration_rebuild_required is explicit.
- [ ] AC-134 invalid lifecycle transition is explicit.
- [ ] AC-135 errors do not discard recoverable REVIEW waypoint state.

## Observability/performance

- [ ] AC-136 Walk materialization latency is measured.
- [ ] AC-137 lifecycle transition latency is measured.
- [ ] AC-138 completion latency is measured.
- [ ] AC-139 exploration read latency is measured.
- [ ] AC-140 rebuild duration/walk count/segment count are measured.
- [ ] AC-141 exact route/waypoint geometry is not logged by default.
- [ ] AC-142 completion logs new/revisited counts without sensitive geometry.
- [ ] AC-143 materialization representative p95 is within 1–2s or disposition documented.
- [ ] AC-144 lifecycle p95 <500ms or disposition documented.
- [ ] AC-145 completion p95 <2s or bottleneck/disposition documented.
- [ ] AC-146 exploration bbox p95 <500ms or disposition documented.

## Automated validation

- [ ] AC-147 Go unit suite passes.
- [ ] AC-148 Go vet/static checks pass.
- [ ] AC-149 real PostGIS Stage 3 integration suite passes.
- [ ] AC-150 transaction rollback regression passes.
- [ ] AC-151 concurrent completion regression passes.
- [ ] AC-152 version-stale regression passes.
- [ ] AC-153 rebuild equivalence regression passes.
- [ ] AC-154 frontend lint/typecheck/unit tests pass.
- [ ] AC-155 Playwright full first-Walk E2E passes.
- [ ] AC-156 Playwright correction E2E passes.
- [ ] AC-157 Playwright reload-recovery E2E passes.
- [ ] AC-158 Playwright no-new second Walk E2E passes.
- [ ] AC-159 docs-as-code index/check passes.
- [ ] AC-160 docker compose config/check passes.

## Manual/product validation

- [ ] AC-161 dense-center full flow manually validated.
- [ ] AC-162 regular-urban full flow manually validated.
- [ ] AC-163 park full flow manually validated.
- [ ] AC-164 physical mobile full flow manually validated.
- [ ] AC-165 Walk Summary is understandable without debug knowledge.
- [ ] AC-166 distinction between route distance and new-network length is understandable.
- [ ] AC-167 user understands exploration changes only after final confirmation.

## Scope discipline

- [ ] AC-168 no authentication/account system.
- [ ] AC-169 no GPS/GPX capture.
- [ ] AC-170 no Places/Visits/Photos.
- [ ] AC-171 no Sharing/Recommendations/Social.
- [ ] AC-172 no cumulative PARTIAL accumulation.
- [ ] AC-173 no completed Walk edit/delete product flow.
- [ ] AC-174 no background job infrastructure solely for rebuild.
- [ ] AC-175 no user exclusion UI/storage unless scope is explicitly revised.

## Final DoD

- [ ] AC-176 complete end-to-end exploration loop works from arbitrary manual route.
- [ ] AC-177 completion retry/concurrency cannot duplicate progress.
- [ ] AC-178 persistent exploration survives reload/process restart.
- [ ] AC-179 rebuild reconstructs equivalent current materialized progress from COMPLETED Walk.
- [ ] AC-180 Stage 3 validation report is complete and Stage 4 readiness is explicit.
