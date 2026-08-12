# ADR-0012: Rebuildable actor exploration read model

**Status:** Proposed  
**Date:** 2026-08-12  
**Stage:** 3

## Context

The project already defines exploration as derived/rebuildable state.

StreetSegment identity is version-scoped, and future geo updates can replace the current graph.

Historical Walk summary should remain stable while current map progress may need rebuilding against
a new graph.

## Decision

`UserStreetSegmentProgress` is a materialized actor read model.

Persistent source of truth remains:

```text
COMPLETED Walk
+
final Route geometry
+
analysis/GeoDataVersion semantics
```

Stage 3 also stores immutable completion snapshots:

```text
ExplorationDelta
WalkDistrictDelta
```

Rebuild changes current materialized progress only and does not rewrite historical snapshots.

`ExplorationState(actor, city)` records which GeoDataVersion current progress represents and whether
rebuild is required.

Stage 3 persistent progress is binary: only `COMPLETED EXPLORE` segments become explored.
`PARTIAL` does not accumulate across Walk.

## Consequences

Pros:

- progress can recover from algorithm/data problems;
- geo updates do not destroy Walk history;
- current map can detect stale state explicitly;
- no incorrect summation of overlapping partial coverage.

Cons:

- rebuild cost grows with completed Walk count;
- automatic rebuild jobs are still future work;
- cumulative partial exploration is deferred.

## Alternatives rejected

### Treat progress rows as source of truth

Rejected because geo graph changes and algorithm corrections would make them irreversible.

### Sum partial covered meters across walks

Rejected because overlapping geometries make scalar summation incorrect.

### Rewrite historical Walk deltas on rebuild

Rejected because Walk Summary should remain a historical snapshot of what was shown at completion.
