# Stage 3 — Architecture Contract

Этот документ фиксирует границы, которые implementation не должен менять без отдельного решения.

# 1. High-level architecture

```text
React /map
   ↓
HTTP transport
   ↓
ActorContext
   ↓
Walks application module
   ├──────────────→ Stage 2 RoutePreview service
   │                       ↓
   │                  Valhalla + RouteAnalyzer
   │
   └──────────────→ Exploration application module
                            ↓
                      PostgreSQL/PostGIS
```

# 2. Module responsibility

## `routing/preview`

Остаётся stateless calculator.

Не знает:

- Walk lifecycle;
- persistent progress;
- district before/after;
- actor history.

## `walks`

Владеет:

- persistent Route materialization;
- Walk lifecycle;
- route mutability/finalization;
- ownership;
- creation idempotency.

Не владеет exploration formula.

## `exploration`

Владеет:

- new/revisited classification relative to actor state;
- progress update;
- district progress;
- completion delta;
- current exploration reads;
- rebuild.

Не вызывает Valhalla.

## `geo/routeanalysis`

Остаётся reusable geometry → StreetSegment analyzer.

Не знает Walk/User/progress.

# 3. Allowed dependencies

```text
transport
   ↓
walks / exploration application services
   ↓
domain ports
   ↑
PostGIS repositories
```

`walks` may call Stage 2 `RoutePreview` application service.

`exploration` may use RouteAnalyzer for rebuild.

Forbidden:

```text
HTTP handler → SQL exploration mutation rules
repository → Walk state machine
RouteAnalyzer → UserStreetSegmentProgress
frontend → Valhalla
frontend → actor ID selection
```

# 4. Actor boundary

Application method signatures for actor-owned operations include:

```text
ActorID
```

Actor is resolved before application service.

Stage 3 implementation should make replacement with Stage 4 authenticated actor resolver local to
transport/security composition.

# 5. Materialization trust boundary

Client sends waypoints + opaque expected fingerprint.

Client never sends authoritative:

```text
route geometry
normalized route
coverage segments
new segment IDs
district percentages
```

Backend recomputes all trusted persistent values.

# 6. Route mutability boundary

Route is mutable only while owning Walk is:

```text
DRAFT
REVIEW
```

Walk service is the only component allowed to replace route geometry/coverage.

Once finalized:

```text
Route.finalized_at != NULL
```

all mutation attempts fail.

# 7. Completion boundary

The only operation allowed to create/update personal exploration is:

```text
WalkCompletionService.Complete(...)
```

No mutation from:

- route preview;
- Walk start;
- Walk finish;
- map read;
- frontend optimistic state.

# 8. Progress vs snapshot

```text
UserStreetSegmentProgress
```

is mutable/rebuildable current read model.

```text
ExplorationDelta
WalkDistrictDelta
```

are immutable historical completion snapshots.

Rebuild changes the former, not the latter.

# 9. Binary progress boundary

Persistent progress contains only fully completed explorable segments.

Do not implement:

```text
progress.covered_m += walk.partial_covered_m
```

because overlap between walks makes arithmetic accumulation incorrect.

# 10. Version boundaries

## Route

Pins one `GeoDataVersion`.

## Route analysis snapshot

Must reference the same version.

## Completion

Requires final Route version equal to current READY version.

The completion transaction holds a shared row lock on that READY `GeoDataVersion` until commit,
so a geo publisher cannot supersede it between validation and progress publication.

## Materialization

Route creation and correction hold a shared row lock on the preview-pinned READY
`GeoDataVersion` until commit. If the expected version is no longer READY, persistence aborts with
the recoverable `route_preview_stale` result.

The database requires every Walk and its Route to have the same actor and city. Segment matches are
materialized only when their StreetSegment belongs to the Route's pinned `GeoDataVersion`; a
cross-version match rolls the transaction back as a stale preview.

## ExplorationState

Pins the version represented by current materialized progress.

## District delta

Pins one `DistrictDataVersion` at completion.

# 11. Current exploration read safety

Exploration read must not silently return progress if:

```text
ExplorationState.geo_data_version_id
!=
current READY GeoDataVersion.id
```

Return explicit rebuild-required state/error.

# 12. Rebuild publication

Rebuild analyzes against one pinned target version.

Only final replacement/publish transaction changes visible materialized progress.

If target version ceases to be current before publish, abort publication.

# 13. District geometry semantics

Current district progress uses clipped line length, not segment membership by intersection alone.

A future materialized district-segment weight table is allowed as an optimization, but must preserve
the same clipping semantics.

# 14. Future exclusions seam

Stage 3 denominator implementation should conceptually use:

```text
EligibilityPolicy(actor, segment)
```

Current policy:

```text
segment.classification == EXPLORE
```

Future policy may additionally exclude actor-selected segment/zone.

No exclusion table is required in Stage 3.

# 15. Data ownership and privacy

All Stage 3 user-owned records are private by construction.

No share/public endpoints.

Do not expose development actor ID in ordinary product payload unless needed for debugging.

# 16. Consistency over availability

If correct new/revisited calculation cannot be guaranteed because state/version is stale:

```text
fail explicitly
```

rather than complete with silently incorrect progress.

Geo updates are rare enough at this stage for this tradeoff.
