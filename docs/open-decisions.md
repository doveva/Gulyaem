# Stage 2 — Open Decisions

These items should be answered by implementation/validation evidence.

## OD-01 Exact waypoint UX

Baseline is:

- first tap start;
- second tap destination;
- explicit `+ Точка`;
- draggable markers;
- reorder list.

Validate whether this feels natural on mobile.

Do not redesign routing architecture based on UX preference.

---

## OD-02 Resolved/snapped waypoint visualization

Question:

Should product marker remain at exact user tap or move/show snap returned by routing?

Possible approaches:

```text
input marker only
resolved marker only
input + subtle snap indication
```

Need validation in areas where nearest pedestrian edge is offset.

---

## OD-03 Low-match warning threshold

Stage 1 normal fixtures give ~99–100%, ambiguous case ~91%.

Stage 2 should determine a useful warning threshold.

Candidate engineering starting point may be around `0.95`, but it is not frozen until Stage 2 evidence.

---

## OD-04 Product exploration wording

Need choose wording that is understandable while being semantically correct without UserStreetProgress.

Examples:

```text
Потенциально засчитывается
Исследуемые сегменты
Покрытие маршрута
```

Avoid “new for you”.

---

## OD-05 Product metric selection

Potential candidates:

- completed segment count;
- partial segment count;
- matched explorable route length;
- completed network length.

Need avoid presenting 225 м context ratio as route-newness percentage.

---

## OD-06 Route analysis performance

Stage 1 observed route analysis up to ~2.3 s in broad analysis context.

Need determine whether Stage 2 requires:

- query optimization;
- reduced payload;
- parallel work;
- caching;
- algorithmic optimization.

Coverage semantics must not change merely for speed.

---

## OD-07 HTTP compression

Stage 1 representative raw GeoJSON is ~0.7–2.8 MB.

Need measure:

- gzip/Brotli ratio;
- transfer;
- parse;
- render on target mobile.

Vector tiles only if evidence requires.

---

## OD-08 Background network at overview zoom

Stage 1 ADR proposes route overview without full network.

Validate product comprehension when route is visible but detailed StreetSegment background appears only after zoom-in.

---

## OD-09 Stage 3 materialization boundary

Do not implement yet, but Stage 2 should leave a clear answer:

When user proceeds in Stage 3, should persistent Route be created from:

```text
waypoints + preview geometry + GeoDataVersion
```

or should backend recompute before save/start?

This becomes Stage 3 design input.

---

## OD-10 ADR-0009 status

The provided Stage 1 repository still marks ADR-0009 as `Proposed`.

Before Stage 2 freeze, reviewer should either:

- mark bbox + GeoJSON `Accepted`, or
- replace it with a revised decision based on Stage 2 measurements.

Stage 2 implementation may continue using bbox + GeoJSON as the currently agreed baseline.
