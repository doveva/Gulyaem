# Stage 3 — Domain Model

# 1. Route

`Route` is persistent path data.

Conceptual model:

```text
Route
- id
- actor_id
- city_id
- geo_data_version_id
- source_type = MANUAL
- profile = pedestrian
- waypoints
- geometry
- normalized_geometry
- distance_m
- estimated_duration_sec
- routing_provenance
- analysis_provenance
- revision
- finalized_at?
- created_at
- updated_at
```

`geometry` is the reconstructable final routed LineString.

`normalized_geometry` is the Stage 1/2 analysis representation and may be regenerated.

`Route` is not a Walk.

# 2. Route analysis provenance

Persist enough information to explain/rebuild semantics:

```text
routing:
- engine
- engine_version
- graph_artifact_checksum
- source_checksum

analysis:
- analysis_version
- matching_parameters
- coverage_profile
- geo_normalization_version
```

Raw engine edge identities are not persisted as domain identity.

# 3. RouteSegmentMatch

Stage 3 persists one aggregate coverage row per Route + StreetSegment.

Conceptual:

```text
RouteSegmentMatch
- route_id
- street_segment_id
- classification
- matched_length_m
- covered_length_m
- direct_length_m
- required_length_m
- coverage_status
- provenance
- confidence?
```

`coverage_status`:

```text
COMPLETED
PARTIAL
CONNECTOR
```

`NOT_COVERED` rows are not required for persistence.

# 4. Walk

```text
Walk
- id
- actor_id
- city_id
- route_id
- client_request_id
- status
- started_at?
- finished_at?
- completed_at?
- duration_sec?
- distance_m?
- created_at
- updated_at
```

No comment/photo/place fields in Stage 3.

# 5. Walk state machine

```text
[*] → DRAFT
DRAFT → ACTIVE
ACTIVE → REVIEW
REVIEW → COMPLETED

DRAFT → CANCELLED
ACTIVE → CANCELLED
REVIEW → CANCELLED
```

Terminal:

```text
COMPLETED
CANCELLED
```

# 6. Transition semantics

## DRAFT → ACTIVE

Set `started_at` once.

## ACTIVE → REVIEW

Set `finished_at` once.

## REVIEW → COMPLETED

Persist exploration transaction and set:

```text
completed_at
duration_sec
distance_m = final Route distance
```

## * → CANCELLED

No exploration mutation.

# 7. UserStreetSegmentProgress

Materialized actor state:

```text
UserStreetSegmentProgress
- actor_id
- street_segment_id
- first_explored_at
- last_explored_at
- visit_count
- first_walk_id
- last_walk_id
```

Existence of row means:

> segment is explored for this actor in the represented materialized graph version.

Stage 3 has no partial progress row.

# 8. Visit count

One completed Walk increments a given segment at most once.

Route loops do not increment `visit_count` multiple times inside the same Walk.

# 9. ExplorationState

```text
ExplorationState
- actor_id
- city_id
- geo_data_version_id
- status
- updated_at
- rebuilt_at?
```

Status:

```text
READY
REBUILD_REQUIRED
```

It describes which geo version the materialized progress represents.

# 10. ExplorationDelta

Immutable Walk completion header:

```text
ExplorationDelta
- walk_id
- actor_id
- geo_data_version_id
- new_segments_count
- revisited_segments_count
- new_network_length_m
- created_at
```

# 11. ExplorationDeltaSegment

```text
ExplorationDeltaSegment
- walk_id
- street_segment_id
- kind
- segment_length_m
- covered_length_m
```

Kind:

```text
NEW
REVISITED
```

This table supports Walk Summary without reconstructing historical user state.

# 12. WalkDistrictDelta

```text
WalkDistrictDelta
- walk_id
- district_id
- district_data_version_id
- geo_data_version_id
- eligible_length_m
- explored_before_m
- explored_after_m
- new_length_m
- percentage_before
- percentage_after
```

Persist rows for districts whose progress changes.

# 13. New segment definition

At the instant of one completion transaction:

```text
NEW =
route coverage COMPLETED
AND StreetSegment EXPLORE
AND no actor progress row before completion
```

# 14. Revisited definition

```text
REVISITED =
route coverage COMPLETED
AND StreetSegment EXPLORE
AND actor progress row existed before completion
```

# 15. PARTIAL

`PARTIAL` is route-level evidence, not persistent actor exploration in Stage 3.

Two partial walks do not combine.

This is an explicit MVP simplification.

# 16. Route correction

Route correction replaces current mutable Route data and coverage snapshot.

It does not create a second Walk.

`revision` increments.

No historical route revision log in Stage 3.

# 17. Route finalization

At successful completion:

```text
route.finalized_at = walk.completed_at
```

After this, route and match rows are immutable.

# 18. Rebuild source

Current materialized progress can be reconstructed from:

```text
all actor COMPLETED Walk
→ final Route geometry
→ target GeoDataVersion
→ frozen/analyzable coverage semantics
```

Historical delta rows are not the source of truth for current rebuild.
