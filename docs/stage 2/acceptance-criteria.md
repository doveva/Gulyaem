# Stage 2 — Acceptance Criteria

**Freeze status:** 98 / 100 checked.

Каждый отмеченный пункт подтверждён кодом, automated checks или измерениями из
[`validation-report.md`](validation-report.md). Два неотмеченных пункта требуют именно полевой
проверки на физическом мобильном устройстве и unaided product-comprehension review; они не скрыты
за общим утверждением «validation complete».

## Routing runtime

- [x] AC-01 Valhalla is reproducibly runnable outside the Stage 1 comparison-only flow.
- [x] AC-02 Routing graph is built from documented source dataset.
- [x] AC-03 Routing dataset metadata contains source checksum.
- [x] AC-04 Routing service has health/readiness check.
- [x] AC-05 Engine/version remain pinned or explicitly documented.

## Routing adapter

- [x] AC-06 Go backend owns the Valhalla call; browser does not call Valhalla directly.
- [x] AC-07 Engine-neutral `RoutingEngine` boundary exists.
- [x] AC-08 Adapter supports pedestrian route for 2+ waypoints.
- [x] AC-09 Distance and duration are normalized.
- [x] AC-10 Route geometry is returned as internal GeoJSON LineString.
- [x] AC-11 Raw Valhalla edge/tile IDs are not domain identity.
- [x] AC-12 Valhalla failures are normalized.

## Dataset compatibility

- [x] AC-13 current READY GeoDataVersion is resolved for preview.
- [x] AC-14 routing source checksum is compared with GeoDataVersion checksum.
- [x] AC-15 mismatch returns explicit non-success response.
- [x] AC-16 mismatch cannot silently produce exploration preview.

## Route analyzer reuse

- [x] AC-17 arbitrary route geometry can be analyzed without sample fixture files.
- [x] AC-18 Stage 1 sample routes reuse the same analyzer.
- [x] AC-19 matching algorithm is not duplicated.
- [x] AC-20 coverage algorithm is not duplicated.
- [x] AC-21 Balanced remains product preview default.
- [x] AC-22 grade-aware regression remains green.

## Route preview service/API

- [x] AC-23 `POST /api/v1/route-previews` exists.
- [x] AC-24 preview requires 2–10 waypoints.
- [x] AC-25 coordinate validation exists.
- [x] AC-26 only pedestrian profile is accepted in Stage 2.
- [x] AC-27 preview does not create persistent Route/Walk records.
- [x] AC-28 success response separates routing and exploration preview semantics.
- [x] AC-29 GeoDataVersion is present in response.
- [x] AC-30 completed segment count is available.
- [x] AC-31 partial segment count is available.
- [x] AC-32 route matched ratio is available.
- [x] AC-33 unmatched fragments remain diagnosable.
- [x] AC-34 invalid body returns controlled 400.
- [x] AC-35 invalid waypoints return controlled 422.
- [x] AC-36 no-route returns controlled 422.
- [x] AC-37 routing unavailable returns controlled 503.
- [x] AC-38 routing timeout returns controlled 504.
- [x] AC-39 routing/geo mismatch returns controlled 409.

## Product `/map`

- [x] AC-40 `/map` supports entering route-builder mode.
- [x] AC-41 first map point becomes start.
- [x] AC-42 second map point becomes destination.
- [x] AC-43 route automatically calculates with 2 points.
- [x] AC-44 route geometry is visible.
- [x] AC-45 distance is visible.
- [x] AC-46 duration is visible.
- [x] AC-47 potential COMPLETED segments are visible.
- [x] AC-48 potential PARTIAL segments are visually distinguishable.
- [x] AC-49 product UI does not call potential coverage “already/new for user”.
- [x] AC-50 intermediate waypoint can be added.
- [x] AC-51 waypoint can be dragged.
- [x] AC-52 intermediate waypoint can be removed.
- [x] AC-53 intermediate waypoints can be reordered.
- [x] AC-54 route can be cleared/restarted.
- [x] AC-55 stale response cannot overwrite latest waypoint state.

## Error UX

- [x] AC-56 no-route keeps waypoints editable.
- [x] AC-57 routing-unavailable keeps route draft editable.
- [x] AC-58 calculating state is visible.
- [x] AC-59 old preview is not presented as current while recalculating.
- [x] AC-60 low match can be communicated without blocking valid route preview.

## Map/data behavior

- [x] AC-61 route remains visible when background StreetSegment bbox is not loaded at overview zoom.
- [x] AC-62 background network continues to honor bbox/feature limits.
- [x] AC-63 required map/data attribution remains visible.
- [x] AC-64 background GeoJSON compression is implemented or explicitly validated/accepted for Stage 2.
- [x] AC-65 vector tiles are not introduced without measured need.

## Observability

- [x] AC-66 route-preview total latency is measured.
- [x] AC-67 Valhalla latency is measured.
- [x] AC-68 route-analysis latency is measured.
- [x] AC-69 route-match ratio is observable.
- [x] AC-70 exact waypoint coordinates are not logged by default.
- [x] AC-71 routing dataset mismatch is observable.

## Automated tests

- [x] AC-72 Valhalla adapter contract tests exist.
- [x] AC-73 compatibility mismatch test exists.
- [x] AC-74 route-preview service tests exist.
- [x] AC-75 PostGIS route-analysis integration tests remain green.
- [x] AC-76 HTTP endpoint tests cover success/errors.
- [x] AC-77 frontend route-builder unit tests exist.
- [x] AC-78 Playwright covers end-to-end two-point route.
- [x] AC-79 Playwright covers waypoint editing.
- [x] AC-80 Playwright covers stale/error behavior where practical.

## Manual validation

- [x] AC-81 dense-center route manually validated.
- [x] AC-82 regular-urban route manually validated.
- [x] AC-83 park route manually validated.
- [ ] AC-84 mobile viewport interaction manually validated (automated `390 × 844` pass only).
- [x] AC-85 desktop interaction manually validated.
- [ ] AC-86 potential exploration visualization is understandable without debug explanation.

## Performance

- [x] AC-87 route-preview p50/p95 measured on representative fixtures.
- [x] AC-88 latency breakdown is documented.
- [x] AC-89 if p95 misses 1–2 s engineering target, bottleneck and disposition are documented.
- [x] AC-90 representative mobile payload/render behavior is measured.

## Scope discipline

- [x] AC-91 no authentication.
- [x] AC-92 no UserStreetProgress.
- [x] AC-93 no persistent Route created by preview.
- [x] AC-94 no Walk lifecycle.
- [x] AC-95 no GPS/GPX product capture.
- [x] AC-96 no Places/Visits/Photos.
- [x] AC-97 no Sharing/Recommendations/Social.

## Final DoD

- [x] AC-98 reviewer can build/edit a pedestrian route from arbitrary map points and understand its route and potential exploration coverage.
- [x] AC-99 Stage 2 validation report is complete.
- [x] AC-100 Stage 3 can begin without reopening Stage 1 geo semantics.
