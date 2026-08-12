# ADR-0011: Atomic and idempotent Walk completion

**Status:** Proposed  
**Date:** 2026-08-12  
**Stage:** 3

## Context

`Walk completion` is the first operation that changes persistent exploration.

It must update several related states:

```text
Walk
Route finalization
UserStreetSegmentProgress
ExplorationDelta
ExplorationDeltaSegment
WalkDistrictDelta
```

A network retry or concurrent duplicate request must not double-count exploration.

## Decision

`REVIEW → COMPLETED` is one PostgreSQL transaction.

The operation locks the actor-owned Walk row and:

1. validates state;
2. validates final Route/current geo version;
3. validates actor exploration state;
4. calculates NEW/REVISITED from persisted route coverage and pre-transaction progress;
5. upserts progress;
6. inserts one immutable ExplorationDelta and its segment/district rows;
7. finalizes Route;
8. transitions Walk to COMPLETED.

Uniqueness constraints and terminal Walk state make repeated/concurrent completion idempotent.

A request for an already COMPLETED Walk returns the persisted completion result.

`finish` does not mutate exploration.

## Consequences

Pros:

- no partially applied progress;
- network retries safe;
- clear source of exploration mutation;
- summary and progress become consistent.

Cons:

- completion contains spatial/district work inside a transactional boundary;
- long completion may increase lock duration;
- performance must be measured before considering async architecture.

## Alternatives rejected

### Eventually consistent completion events

Deferred. Unnecessary complexity for current scale and weakens immediate reward correctness.

### Client-side optimistic exploration

Rejected. Client cannot be the source of persistent progress.

### Multiple independent transactions

Rejected because partial failures could desynchronize Walk, progress and summary.
