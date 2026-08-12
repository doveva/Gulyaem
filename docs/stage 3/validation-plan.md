# Stage 3 — Validation Plan

# 1. Purpose

Validate the first persistent exploration loop:

```text
route → walk → review → complete → progress → summary → reload → rebuild
```

# 2. Required environments

Reuse Stage 1/2 areas:

- Dense Center;
- Regular Urban;
- Park + Residential.

# 3. Core scenario A — First exploration

Actor has empty progress.

Route completes several EXPLORE segments.

Expected:

- all completed explorable segments are NEW;
- persistent overlay appears;
- district percentage increases;
- retry completion changes nothing.

# 4. Scenario B — Overlapping second Walk

Walk 1:

```text
A B C
```

Walk 2:

```text
B C D
```

Expected Walk 2:

```text
NEW: D
REVISITED: B C
```

`visit_count` B/C increments once.

Progress remains monotonic.

# 5. Scenario C — No-new Walk

Route uses only already explored segments.

Expected:

- completion succeeds;
- `newSegmentsCount = 0`;
- revisited count >0;
- district before == after;
- summary is valid, not error.

# 6. Scenario D — PARTIAL only

Construct route where target EXPLORE segment remains PARTIAL.

Expected:

- no progress row created;
- second independent PARTIAL Walk still does not combine;
- summary does not report it as new completed segment.

# 7. Scenario E — ROUTABLE_ONLY connector

Route traverses connector.

Expected:

- route works;
- connector may be persisted in Route analysis;
- no user progress row;
- no denominator/new-length contribution.

# 8. Scenario F — Review correction

Create/start/finish Walk.

In REVIEW edit waypoint so final route changes.

Expected:

- route revision increments;
- old coverage snapshot replaced;
- completion uses corrected route only;
- summary corresponds to correction.

# 9. Scenario G — Stale preview

Generate Stage 2 preview.

Change relevant materialization fingerprint input in controlled test.

Create Walk with old fingerprint.

Expected:

```text
409 route_preview_stale
```

No Route/Walk persisted.

# 10. Scenario H — Completion concurrency

Send two completion requests concurrently.

Expected:

- one logical mutation;
- one delta;
- progress visit counts increment once;
- both callers receive compatible completed result.

# 11. Scenario I — Transaction rollback

Inject failure after progress calculation but before Walk finalization.

Expected after rollback:

- Walk still REVIEW;
- no new progress;
- no delta;
- route not finalized.

# 12. Scenario J — Geo version update

Create Walk on version A.

Publish B before completion.

Expected:

```text
walk_route_geo_version_stale
```

No completion.

Refresh/correct Route on B.

If actor progress is A:

```text
exploration_rebuild_required
```

After rebuild B, completion succeeds.

# 13. Scenario K — Rebuild equivalence

After several completed Walk:

1. capture explored segment set and current statistics;
2. remove current materialized progress in isolated test transaction/environment;
3. run rebuild;
4. compare.

Expected:

- same segment set;
- same visit counts/first/last semantics;
- same city progress;
- same district current progress within floating tolerance.

Historical delta rows unchanged.

# 14. Scenario L — Browser reload

During ACTIVE:

- reload;
- restore from local activeWalkId + GET Walk.

During REVIEW:

- reload;
- restore review.

After COMPLETED:

- local pointer cleared;
- map persistent overlay remains.

# 15. Actor isolation

Integration tests use two actor IDs.

Expected:

- same Route/Walk IDs cannot be read cross-actor;
- progress does not leak;
- client cannot choose actor.

# 16. Unit tests

Minimum:

- Walk transition matrix;
- fingerprint determinism/change cases;
- materialization stale comparison;
- new/revisited classification;
- binary PARTIAL rule;
- actor-state compatibility;
- summary calculations.

# 17. PostgreSQL integration tests

Use real PostGIS for:

- migrations;
- route/match persistence;
- transaction locking;
- completion;
- district clipping;
- bbox explored segments;
- rebuild publish.

# 18. API tests

Cover:

- create;
- duplicate create retry;
- stale create;
- get;
- start;
- finish;
- correction;
- complete;
- repeat complete;
- cancel;
- invalid states;
- ownership;
- exploration reads;
- rebuild-required errors.

# 19. Frontend tests

Reducer/component:

- Start flow;
- create/start partial failure recovery;
- active timer from server timestamp;
- finish;
- review;
- correction;
- complete;
- no-new summary;
- conflict states;
- local activeWalkId recovery.

# 20. Playwright

Minimum E2E:

## E2E-1 Full first Walk

```text
/map
→ route preview
→ Start
→ Active
→ Finish
→ Review
→ Complete
→ Summary
→ Return to map
→ explored overlay visible
```

## E2E-2 Correction

Review → edit → save corrected route → complete.

## E2E-3 Reload recovery

Reload ACTIVE and REVIEW.

## E2E-4 Idempotent UI retry

Simulate timeout/retry around create or complete where practical.

## E2E-5 No-new second Walk

Complete overlapping/identical route and verify zero-new summary.

Run desktop and mobile viewport.

# 21. Physical mobile validation

On real phone verify:

- Start CTA;
- active screen;
- Finish;
- review correction touch;
- Complete;
- summary map;
- return to map;
- reload recovery;
- no accidental fitness/GPS implication.

# 22. Performance report

Measure at least representative warmed operations:

```text
walk materialization p50/p95
start p50/p95
finish p50/p95
complete p50/p95
city exploration read p50/p95
bbox explored overlay p50/p95
rebuild total for N walks
```

Record dominant query/phase.

# 23. Stage 3 validation report

Create:

```text
docs/stage 3/validation-report.md
```

Include:

- date/build;
- actor fixture;
- GeoDataVersion;
- DistrictDataVersion;
- automated tests;
- E2E;
- physical mobile result;
- new/revisited scenarios;
- idempotency evidence;
- rollback evidence;
- rebuild evidence;
- performance;
- ADR outcomes;
- known limitations;
- Stage 4 readiness.
