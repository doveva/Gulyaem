# Stage 3 — Open Decisions

These are evidence-driven Stage 3 questions. Baselines below are proposed, not yet frozen.

# OD-01 Preview fingerprint canonicalization

Proposed:

```text
stage3-preview-fingerprint-v1
SHA-256(canonical server-side materialization snapshot)
```

Need freeze exact fields/canonical serialization after implementation tests.

Client must never calculate it.

---

# OD-02 Route materialization stale UX

Architecture decision is already clear:

```text
stale → do not persist silently
```

Validate UX:

- automatically show refreshed preview;
- exact user copy;
- whether re-pressing Start is understandable.

---

# OD-03 Persistent Route analysis detail

Proposed minimum aggregate match table is enough for completion.

Validate whether Stage 3 needs individual matched-fragment rows for:

- Route Review;
- explanation;
- future correction.

Default: do not persist fragment-level diagnostics unless a product/debug need appears.

---

# OD-04 Completion low-match policy

Stage 2 warns below:

```text
0.95
```

Stage 3 must decide whether low match:

- remains warning but allows completion;
- requires explicit review confirmation;
- blocks completion under a lower hard threshold.

Baseline proposal:

```text
<0.95 → warning in REVIEW
no hard block solely from ratio
```

because manual route is already generated on the pedestrian graph.

---

# OD-05 District calculation materialization

Baseline:

```text
calculate clipped lengths with PostGIS
```

Measure completion/read p95.

Only introduce precomputed district-segment weights if needed.

---

# OD-06 Walk Summary network metric

Proposed primary reward:

```text
new_network_length_m
```

= sum of full weight/length contributions of newly completed explorable network.

Validate wording so user does not interpret it as exact physically walked GPS distance.

---

# OD-07 Development actor lifecycle

Baseline:

```text
DEVELOPMENT_ACTOR_ID
```

No actor/user table in Stage 3.

Stage 4 will add identity/account relation.

Validate whether adding a minimal domain user row now materially improves referential integrity.
Default: defer until Stage 4.

---

# OD-08 Exploration state after geo update

Stage 3 baseline:

```text
explicit REBUILD_REQUIRED
manual/CLI rebuild
```

No automatic jobs.

Validate developer workflow and make sure product never silently displays invalid zero progress.

---

# OD-09 PARTIAL persistence

Stage 3 baseline is resolved conceptually:

```text
PARTIAL does not accumulate
```

During validation confirm representative StreetSegment lengths make this acceptable for MVP.

If users repeatedly miss completion unnaturally, reopen through a dedicated cumulative-geometry ADR,
not by adding covered-meter sums.

---

# OD-10 User exclusions integration point

Exclusions are future accepted semantics.

Stage 3 must identify the exact denominator service/query seam where future:

```text
actor exclusions
```

will be subtracted.

No storage/UI required now.

Freeze the extension point in validation report.

---

# OD-11 Rebuild artifact comparison

Define exact acceptance equivalence:

Recommended:

```text
same current explored StreetSegment ID set
same visit_count / first/last semantics
same city/district current progress within numeric tolerance
```

Historical Walk deltas intentionally need not be regenerated.

---

# OD-12 Completion performance

If completion p95 >2 s, measure split:

- locking/state;
- progress lookup/upsert;
- district clipping;
- serialization.

Do not move completion asynchronous until evidence shows synchronous transaction is not acceptable.
