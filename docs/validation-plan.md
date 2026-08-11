# Stage 2 — Validation Plan

# 1. Purpose

Validate:

1. arbitrary manual route construction;
2. routing correctness;
3. StreetSegment analysis reuse;
4. coverage preview comprehension;
5. editing responsiveness;
6. mobile usability;
7. performance.

---

# 2. Fixed environments

Use at least the three Stage 1 environments:

```text
Dense Center
Regular Urban District
Park + Residential
```

Reuse existing fixture/test-area coordinates where practical.

---

# 3. Manual route scenarios

## Scenario A — Simple two-point center route

Goal:

- start/destination;
- ordinary pedestrian routing;
- distance/duration;
- high match ratio.

Validate that route line and potential coverage are readable over dense map.

---

## Scenario B — Center with intermediate waypoint

Goal:

```text
A → B → C
```

Move/reorder B.

Validate:

- route materially changes;
- stale request never wins;
- new preview corresponds to markers.

---

## Scenario C — Residential connectors

Choose route where Valhalla may use connectors/service access.

Validate:

- route remains connected;
- `ROUTABLE_ONLY` contributes to traversal;
- it does not appear as exploration completion.

---

## Scenario D — Park route

Validate:

- park paths are naturally selected;
- public tracks/paths classification remains consistent;
- coverage visualization is not excessively wide/confusing.

---

## Scenario E — Hard/no-route case

Pick inaccessible or disconnected points.

Validate:

- controlled error;
- waypoints remain editable;
- user can recover without restarting app.

---

# 4. Automated backend tests

## Routing adapter

Fake HTTP server:

- successful route;
- malformed upstream JSON;
- 4xx/no route;
- 5xx;
- timeout;
- missing geometry;
- non-finite distance/duration if applicable.

## Compatibility

- matching checksum succeeds;
- mismatch blocks preview;
- missing READY geo version fails explicitly.

## Preview service

- 2 waypoints;
- multiple waypoints;
- invalid waypoints;
- analyzer warning propagation;
- route-analysis failure handling;
- no persistence side effect.

## PostGIS

Existing Stage 1 route-analysis integration suite remains mandatory.

---

# 5. Frontend tests

Unit/component tests:

- initial state;
- start/destination state;
- waypoint add/delete/reorder;
- preview loading;
- success summary;
- route-not-found;
- routing unavailable;
- stale result rejection.

---

# 6. Playwright

Minimum E2E:

### E2E-1 two-point preview

```text
open /map
enter builder
set A
set B
wait preview
assert route + distance + duration + coverage
```

### E2E-2 edit waypoint

```text
add/move waypoint
ensure new preview
ensure old route not restored
```

### E2E-3 failure recovery

Mock/trigger controlled route failure and ensure waypoint draft remains editable.

Run at desktop and one mobile viewport.

---

# 7. Performance measurement

For representative routes collect at least 30 warmed requests where practical.

Report:

```text
total p50/p95
routing p50/p95
analysis p50/p95
response bytes
```

Separately measure background segment GeoJSON:

```text
raw bytes
compressed bytes
```

On a real mobile device record qualitative:

- route response feel;
- map pan/zoom;
- marker drag;
- layer update;
- sheet interaction.

---

# 8. Visual review questions

Reviewer should answer:

1. Is it obvious where the route starts/ends?
2. Is it obvious that markers are editable?
3. Is route geometry visually primary?
4. Are potentially completed segments understandable?
5. Is PARTIAL useful or visually noisy?
6. Is ROUTABLE_ONLY correctly de-emphasized?
7. Does background network overwhelm route?
8. Is low matching communicated without becoming debug noise?
9. Can route be edited comfortably on phone?
10. Does user understand that preview is not yet a saved/completed walk?

---

# 9. Stage 2 validation report

Create:

```text
docs/stage 2/validation-report.md
```

Minimum content:

- build/version/date;
- source GeoDataVersion;
- routing dataset checksum;
- automated results;
- route fixture results;
- latency;
- payload;
- manual review;
- open limitations;
- Stage 3 readiness decision.
