# ГуляЕм — Stage 3 Requirements: Exploration Core

**Status:** Draft v0.1  
**Stage:** 3 — Exploration Core  
**Backend:** Go  
**Frontend:** React + TypeScript + MapLibre  
**Storage:** PostgreSQL + PostGIS  
**Routing:** Valhalla through frozen Stage 2 port  
**Authentication:** not in scope; configured development actor  
**Primary city:** Санкт-Петербург

---

# 1. Goal

Stage 3 реализует первый полный domain vertical slice:

```text
Map
 ↓
Manual Route Preview
 ↓
Materialize Route + Walk
 ↓
Start
 ↓
Active
 ↓
Finish
 ↓
Route Review
 ↓
Complete
 ↓
ExplorationDelta
 ↓
Personal Exploration Progress
 ↓
Walk Summary
 ↓
Updated Exploration Map
```

До Stage 3 система знает только:

> какие сегменты маршрут потенциально покроет.

После Stage 3 система должна знать:

> какие сегменты конкретный actor уже исследовал и что изменила конкретная завершённая прогулка.

---

# 2. Validation question

Главный product/technical вопрос:

> Является ли `Walk completion → visual city reveal` понятным, мотивирующим и технически
> воспроизводимым core loop?

Stage 3 должен доказать одновременно:

- корректность lifecycle;
- корректность personal progress;
- идемпотентность completion;
- rebuildability;
- понятность summary;
- сохранение состояния после reload.

---

# 3. Stage 3 ownership without auth

Stage 4 добавит accounts/authentication.

Stage 3 уже обязан хранить owner scope для:

```text
Route
Walk
UserStreetSegmentProgress
ExplorationState
ExplorationDelta
```

Но HTTP client не передаёт owner ID.

Stage 3 вводит:

```text
ActorContext
```

Development HTTP resolver получает actor из:

```text
DEVELOPMENT_ACTOR_ID
```

Application services получают `ActorID` явно.

Stage 4 заменяет только transport/auth resolver.

---

# 4. Route materialization from Stage 2 preview

Stage 2 preview остаётся stateless.

Stage 3 НЕ принимает browser-provided geometry как persistent truth.

Preview response расширяется opaque:

```text
previewFingerprint
```

При создании Walk browser отправляет:

```text
cityId
profile
ordered waypoints
expectedPreviewFingerprint
clientRequestId
```

Backend:

```text
recompute preview
↓
compute fingerprint
↓
compare
↓
persist only trusted server result
```

Mismatch:

```text
409 route_preview_stale
```

Frontend должен показать свежий preview и потребовать повторное действие пользователя.

---

# 5. Preview fingerprint semantics

Fingerprint должен меняться, если меняется что-либо, способное изменить materialized Route:

- city;
- ordered input waypoints;
- routing profile;
- `GeoDataVersion`;
- routing engine/version;
- routing graph artifact checksum;
- route geometry;
- normalized geometry / analysis version;
- coverage profile.

Client считает fingerprint opaque.

Алгоритм versioned:

```text
stage3-preview-fingerprint-v1
```

Не использовать fingerprint как authorization/security token.

---

# 6. Persistent Route

Stage 3 вводит persistent `Route`.

Route хранит минимум:

- owner/actor;
- city;
- pinned `GeoDataVersion`;
- source type `MANUAL`;
- ordered waypoints;
- final routed geometry;
- normalized route geometry;
- distance;
- estimated duration;
- routing provenance;
- analysis/coverage provenance;
- revision;
- created/updated/finalized timestamps.

Route может изменяться до completion в разрешённых Walk states.

После Walk `COMPLETED` Route immutable.

---

# 7. Persisted route coverage snapshot

При materialization backend сохраняет aggregate result route analysis по StreetSegment.

Нужно хранить достаточно данных, чтобы completion не выполнял повторно дорогой matching:

```text
street_segment_id
classification
matched_length_m
covered_length_m
direct_length_m
required_length_m
coverage_status
provenance
confidence?
```

Stage 3 exploration mutation использует trusted persisted coverage snapshot.

Rebuildability всё равно опирается на final Route geometry, а не на вечность snapshot rows.

---

# 8. Walk domain

Stage 3 вводит `Walk`:

```text
DRAFT
ACTIVE
REVIEW
COMPLETED
CANCELLED
```

Walk и Route — разные сущности.

Route описывает путь.

Walk описывает фактическое пользовательское событие и lifecycle.

---

# 9. Creating a Walk

`POST /walks`:

1. validates actor/city/request;
2. recomputes Stage 2 preview;
3. verifies fingerprint;
4. persists Route + route coverage;
5. persists Walk in `DRAFT`;
6. returns both.

Creation is idempotent by:

```text
clientRequestId
```

unique per actor.

Retry after timeout must return the existing Walk rather than create duplicate.

---

# 10. Starting a Walk

Allowed:

```text
DRAFT → ACTIVE
```

On first successful start:

```text
started_at = server time
```

Repeated start on already ACTIVE returns current state without changing timestamp.

Start does not update exploration.

---

# 11. Active Walk

Stage 3 has no GPS capture.

ACTIVE UI may show:

- planned route;
- elapsed wall-clock time from `started_at`;
- planned route distance;
- Finish / Cancel.

It MUST NOT show invented:

- current GPS position;
- traveled distance;
- live speed;
- fitness metrics.

---

# 12. Finishing a Walk

Allowed:

```text
ACTIVE → REVIEW
```

On first finish:

```text
finished_at = server time
```

Duration is derived from server timestamps.

Repeated finish on REVIEW returns current state.

`finish` does not update exploration.

---

# 13. Route Review

REVIEW is mandatory before exploration mutation.

User can:

- accept current route;
- re-open manual builder;
- add/remove/move/reorder waypoints;
- receive normal Stage 2 preview;
- persist a corrected route.

Route correction uses the same server-side rematerialization and fingerprint verification as initial
Walk creation.

---

# 14. Route correction rules

Allowed in:

```text
DRAFT
REVIEW
```

Forbidden in:

```text
ACTIVE
COMPLETED
CANCELLED
```

Correction:

1. recomputes trusted preview;
2. replaces Route geometry/provenance;
3. increments `route.revision`;
4. replaces route coverage snapshot atomically.

No route-revision history is required in Stage 3.

---

# 15. Geo version and correction

A REVIEW correction may move Route to the current `GeoDataVersion`.

This is intentional: final Route represents user-confirmed geometry at completion time.

Completion requires final Route to reference current READY geo version.

If current geo version changed after Route was built:

```text
409 walk_route_geo_version_stale
```

User must refresh/rematerialize the route in REVIEW before completion.

---

# 16. Completing a Walk

Allowed:

```text
REVIEW → COMPLETED
```

Completion is one strong-consistency transaction.

It must:

1. lock Walk;
2. validate actor and REVIEW status;
3. load final Route;
4. ensure Route geo version is current READY;
5. ensure actor exploration state is compatible/current;
6. read persisted Route coverage;
7. select completed `EXPLORE` segments;
8. compare with actor progress;
9. insert/update progress;
10. create `ExplorationDelta`;
11. create district snapshots;
12. finalize Route;
13. set Walk `COMPLETED`;
14. commit.

No exploration mutation may become visible if any required part fails.

---

# 17. Idempotent completion

Repeated `POST /walks/{id}/complete` after successful commit:

- returns stored completion result;
- does not increment `visit_count` again;
- does not create duplicate delta;
- does not change timestamps.

Concurrent duplicate completion requests must also produce exactly one progress mutation.

Use row locks and database uniqueness as correctness mechanisms.

Do not rely only on frontend request suppression.

---

# 18. Persistent progress semantics

Stage 3 stores actor-scoped binary StreetSegment progress.

Only:

```text
classification = EXPLORE
AND coverage_status = COMPLETED
```

affects persistent exploration.

`PARTIAL`:

- remains visible in Route/Walk analysis;
- does not create `UserStreetSegmentProgress`;
- does not accumulate between Walk.

`ROUTABLE_ONLY`:

- may be part of route;
- never enters exploration denominator;
- never creates progress.

`IGNORE` never creates progress.

---

# 19. UserStreetSegmentProgress semantics

For each actor + explored StreetSegment store minimum:

```text
first_explored_at
last_explored_at
visit_count
first_walk_id
last_walk_id
```

`visit_count` increments at most once per completed Walk per segment even if route loops over the
same segment multiple times.

Stage 3 does not claim exact per-segment visit timestamp; `finished_at` is an acceptable walk-level
timestamp source.

---

# 20. ExplorationDelta

Each COMPLETED Walk stores an immutable historical summary.

Minimum:

```text
walk_id
actor_id
geo_data_version_id
new_segments_count
revisited_segments_count
new_network_length_m
created_at
```

Also persist segment-level delta rows so summary map can identify newly explored segments without
reconstructing historical progress state.

Historical delta is not rewritten by future exploration rebuilds.

---

# 21. New vs revisited

For the Walk completion transaction:

```text
new
=
COMPLETED EXPLORE segment
not present in actor progress before this completion
```

```text
revisited
=
COMPLETED EXPLORE segment
already present before this completion
```

A Walk may validly produce:

```text
0 new segments
```

and still complete successfully.

---

# 22. Walk Summary metrics

Minimum Stage 3 reward:

- route distance;
- duration;
- number of newly explored segments;
- newly explored network length;
- district percentage before → after;
- map with new segments highlighted.

Do not confuse:

```text
new_network_length_m
```

with actual GPS-traveled distance.

No GPS exists yet.

---

# 23. District progress formula

District progress uses explorable network length clipped to district boundary:

```text
percentage =
explored_eligible_length_inside_district
/
total_eligible_EXPLORE_length_inside_district
```

For a StreetSegment crossing district boundary use:

```text
ST_Length(
  ST_Intersection(segment.geometry, district.boundary)::geography
)
```

or an equivalent correct clipped-length calculation.

Do not assign the full StreetSegment length to every intersected district.

---

# 24. WalkDistrictDelta

For districts changed by completion store immutable snapshot:

```text
walk_id
district_id
district_data_version_id
geo_data_version_id
eligible_length_m
explored_before_m
explored_after_m
new_length_m
percentage_before
percentage_after
```

Historical Walk summary remains stable after district/geo updates.

---

# 25. Current exploration state

`UserStreetSegmentProgress` is rebuildable.

Stage 3 introduces actor/city exploration-state metadata to detect version consistency:

```text
actor_id
city_id
geo_data_version_id
status
updated_at
rebuilt_at?
```

Suggested status:

```text
READY
REBUILD_REQUIRED
```

Product exploration reads require:

```text
state.status = READY
AND state.geo_data_version_id = current READY GeoDataVersion
```

Missing state for a new actor may be interpreted as empty current progress.

---

# 26. Geo update behavior

Stage 3 does not implement automatic geo-update orchestration.

If current `GeoDataVersion` changes and actor progress belongs to old version:

```text
exploration_rebuild_required
```

must be explicit.

Do not silently display empty/partial new-version progress as valid.

A rebuild operation moves current materialized progress to the new version.

---

# 27. Rebuildability

Stage 3 must implement a rebuild service/CLI.

Input:

```text
actor
city
target current GeoDataVersion
```

Process:

```text
COMPLETED Walks ordered chronologically
        ↓
final Route geometry
        ↓
RouteAnalyzer pinned to target version
        ↓
completed EXPLORE segments
        ↓
reconstructed first/last/visit_count
        ↓
atomic replacement of materialized progress
        ↓
ExplorationState READY(target version)
```

Historical `ExplorationDelta` / `WalkDistrictDelta` remain unchanged.

---

# 28. Rebuild correctness

Rebuild must:

- pin one target geo version for the entire run;
- abort final publication if that version ceased to be current before commit;
- never leave half-rebuilt progress visible as READY;
- be idempotent;
- emit summary metrics.

No interactive latency target is required.

---

# 29. User exclusions future compatibility

Per-segment and zone exclusions are an accepted future feature.

Stage 3 does not add their storage/UI.

However:

- denominator calculation belongs in actor-aware exploration service;
- do not persist a single global district percentage as user progress;
- design must allow later subtraction of actor-specific excluded segment lengths;
- recommendation code must not be introduced here.

---

# 30. Exploration map API

Stage 3 adds current actor exploration reads.

Minimum:

```text
GET /api/v1/cities/{cityId}/exploration
GET /api/v1/cities/{cityId}/exploration/segments?bbox=...
```

City exploration returns:

- current state/version;
- city explored/eligible length;
- percentage;
- district progress list.

Segments endpoint returns only actor-completed current-version StreetSegments as GeoJSON for the
viewport.

Use existing bbox limits/patterns.

---

# 31. Walk API

Minimum:

```text
POST /api/v1/walks
GET  /api/v1/walks/{walkId}
POST /api/v1/walks/{walkId}/start
POST /api/v1/walks/{walkId}/finish
PUT  /api/v1/walks/{walkId}/route
POST /api/v1/walks/{walkId}/complete
POST /api/v1/walks/{walkId}/cancel
```

No Stage 3 endpoint is required for:

```text
GET /walks
```

Walk history belongs to Stage 4.

---

# 32. Active Walk refresh recovery

Frontend stores:

```text
activeWalkId
```

in durable local browser storage.

On reload:

1. read ID;
2. fetch Walk;
3. restore ACTIVE or REVIEW UI;
4. clear local pointer if backend Walk is terminal/not found.

Backend remains source of truth.

Do not store final progress only in browser.

---

# 33. Updated map

After completion:

- newly explored segments appear in summary;
- returning to `/map` loads persistent exploration overlay;
- reload retains the same overlay;
- future route previews may visually distinguish already explored background from potential route
  coverage, but Stage 2 preview semantic itself remains unchanged.

---

# 34. Interaction between existing and potential layers

Suggested map order:

```text
base
unexplored/background network
persistent explored network
current route potential PARTIAL
current route potential COMPLETED
route line
waypoints
```

During Walk Summary additionally highlight:

```text
newly explored this Walk
```

as a distinct temporary layer.

---

# 35. Cancellation

Allowed from:

```text
DRAFT
ACTIVE
REVIEW
```

Cancellation:

- transitions to `CANCELLED`;
- does not update exploration;
- does not delete Route automatically;
- terminal state is queryable by ID.

No restore-from-cancel is required.

---

# 36. Completed Route immutability

After Walk completion:

- Route correction forbidden;
- route coverage snapshot immutable;
- Walk completion snapshot immutable.

Edit/delete of historical Walk is deferred.

Later edit/delete semantics may trigger full rebuild.

---

# 37. Transaction boundaries

Must be strongly consistent:

- Walk creation with Route materialization;
- route correction + replacement coverage;
- lifecycle transition;
- completion + progress + delta + route finalization.

Route-preview calculation itself remains stateless and outside persistent transaction until trusted
result is ready.

---

# 38. Performance targets

Engineering targets:

### Walk creation/materialization
Same order as Stage 2 preview:

```text
p95 ≈ 1–2 s
```

### Lifecycle transitions
Without routing/analysis:

```text
p95 < 500 ms
```

### Walk completion
Target:

```text
p95 < 2 s
```

If district/progress calculation exceeds target, measure before introducing asynchronous completion.

### Exploration bbox read

```text
p95 < 500 ms
```

for representative viewport.

---

# 39. Observability

Metrics/logging minimum:

```text
walk_create_duration
walk_start_duration
walk_finish_duration
walk_route_update_duration
walk_complete_duration
walk_cancel_duration

walk_completion_new_segments
walk_completion_revisited_segments
walk_completion_new_network_meters
walk_completion_conflicts

exploration_read_duration
exploration_rebuild_duration
exploration_rebuild_walk_count
exploration_rebuild_segment_count
exploration_rebuild_failures
```

Do not log exact route geometry/waypoint coordinates by default.

---

# 40. Security / ownership

Even without auth:

- actor comes from server context;
- every user-owned repository query filters by actor;
- a Walk ID belonging to another actor returns not found/forbidden according to one consistent
  policy;
- browser cannot set actor ID;
- `clientRequestId` uniqueness is scoped by actor.

Stage 4 will replace actor resolver with authentication.

---

# 41. Explicit non-goals

Stage 3 does NOT implement:

- registration/login;
- social login;
- Walk list/history;
- completed Walk edit/delete;
- GPS;
- GPX product import;
- background location;
- Places;
- Visits;
- Photos;
- Comments;
- Sharing;
- Recommendations;
- Social;
- user exclusion UI;
- cumulative partial coverage;
- automatic background rebuild jobs;
- notifications.

---

# 42. Expected Stage outputs

```text
1. ActorContext boundary
2. persistent Route schema/domain
3. persistent Walk schema/domain
4. route materialization service
5. preview fingerprint
6. route coverage persistence
7. Walk lifecycle service/API
8. route correction flow
9. UserStreetSegmentProgress
10. ExplorationState
11. atomic completion service
12. ExplorationDelta + delta segments
13. WalkDistrictDelta
14. exploration map API
15. rebuild service/CLI
16. full /map flow
17. active/review refresh recovery
18. Walk Summary
19. automated tests
20. validation report
21. accepted/revised Stage 3 ADR
```

---

# 43. Definition of Done

Stage 3 завершён, когда:

```text
Build route
→ Start
→ Finish
→ Review/correct
→ Complete
→ See new segments + district delta
→ Return to map
→ Reload
→ Same persistent exploration remains
```

и отдельно:

```text
delete materialized progress in validation environment
→ run rebuild from COMPLETED Walk
→ receive equivalent current exploration state
```

without duplicating progress on completion retries.
