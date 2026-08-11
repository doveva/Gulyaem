# Stage 2 — Acceptance Criteria

## Routing runtime

- [ ] AC-01 Valhalla is reproducibly runnable outside the Stage 1 comparison-only flow.
- [ ] AC-02 Routing graph is built from documented source dataset.
- [ ] AC-03 Routing dataset metadata contains source checksum.
- [ ] AC-04 Routing service has health/readiness check.
- [ ] AC-05 Engine/version remain pinned or explicitly documented.

## Routing adapter

- [ ] AC-06 Go backend owns the Valhalla call; browser does not call Valhalla directly.
- [ ] AC-07 Engine-neutral `RoutingEngine` boundary exists.
- [ ] AC-08 Adapter supports pedestrian route for 2+ waypoints.
- [ ] AC-09 Distance and duration are normalized.
- [ ] AC-10 Route geometry is returned as internal GeoJSON LineString.
- [ ] AC-11 Raw Valhalla edge/tile IDs are not domain identity.
- [ ] AC-12 Valhalla failures are normalized.

## Dataset compatibility

- [ ] AC-13 current READY GeoDataVersion is resolved for preview.
- [ ] AC-14 routing source checksum is compared with GeoDataVersion checksum.
- [ ] AC-15 mismatch returns explicit non-success response.
- [ ] AC-16 mismatch cannot silently produce exploration preview.

## Route analyzer reuse

- [ ] AC-17 arbitrary route geometry can be analyzed without sample fixture files.
- [ ] AC-18 Stage 1 sample routes reuse the same analyzer.
- [ ] AC-19 matching algorithm is not duplicated.
- [ ] AC-20 coverage algorithm is not duplicated.
- [ ] AC-21 Balanced remains product preview default.
- [ ] AC-22 grade-aware regression remains green.

## Route preview service/API

- [ ] AC-23 `POST /api/v1/route-previews` exists.
- [ ] AC-24 preview requires 2–10 waypoints.
- [ ] AC-25 coordinate validation exists.
- [ ] AC-26 only pedestrian profile is accepted in Stage 2.
- [ ] AC-27 preview does not create persistent Route/Walk records.
- [ ] AC-28 success response separates routing and exploration preview semantics.
- [ ] AC-29 GeoDataVersion is present in response.
- [ ] AC-30 completed segment count is available.
- [ ] AC-31 partial segment count is available.
- [ ] AC-32 route matched ratio is available.
- [ ] AC-33 unmatched fragments remain diagnosable.
- [ ] AC-34 invalid body returns controlled 400.
- [ ] AC-35 invalid waypoints return controlled 422.
- [ ] AC-36 no-route returns controlled 422.
- [ ] AC-37 routing unavailable returns controlled 503.
- [ ] AC-38 routing timeout returns controlled 504.
- [ ] AC-39 routing/geo mismatch returns controlled 409.

## Product `/map`

- [ ] AC-40 `/map` supports entering route-builder mode.
- [ ] AC-41 first map point becomes start.
- [ ] AC-42 second map point becomes destination.
- [ ] AC-43 route automatically calculates with 2 points.
- [ ] AC-44 route geometry is visible.
- [ ] AC-45 distance is visible.
- [ ] AC-46 duration is visible.
- [ ] AC-47 potential COMPLETED segments are visible.
- [ ] AC-48 potential PARTIAL segments are visually distinguishable.
- [ ] AC-49 product UI does not call potential coverage “already/new for user”.
- [ ] AC-50 intermediate waypoint can be added.
- [ ] AC-51 waypoint can be dragged.
- [ ] AC-52 intermediate waypoint can be removed.
- [ ] AC-53 intermediate waypoints can be reordered.
- [ ] AC-54 route can be cleared/restarted.
- [ ] AC-55 stale response cannot overwrite latest waypoint state.

## Error UX

- [ ] AC-56 no-route keeps waypoints editable.
- [ ] AC-57 routing-unavailable keeps route draft editable.
- [ ] AC-58 calculating state is visible.
- [ ] AC-59 old preview is not presented as current while recalculating.
- [ ] AC-60 low match can be communicated without blocking valid route preview.

## Map/data behavior

- [ ] AC-61 route remains visible when background StreetSegment bbox is not loaded at overview zoom.
- [ ] AC-62 background network continues to honor bbox/feature limits.
- [ ] AC-63 required map/data attribution remains visible.
- [ ] AC-64 background GeoJSON compression is implemented or explicitly validated/accepted for Stage 2.
- [ ] AC-65 vector tiles are not introduced without measured need.

## Observability

- [ ] AC-66 route-preview total latency is measured.
- [ ] AC-67 Valhalla latency is measured.
- [ ] AC-68 route-analysis latency is measured.
- [ ] AC-69 route-match ratio is observable.
- [ ] AC-70 exact waypoint coordinates are not logged by default.
- [ ] AC-71 routing dataset mismatch is observable.

## Automated tests

- [ ] AC-72 Valhalla adapter contract tests exist.
- [ ] AC-73 compatibility mismatch test exists.
- [ ] AC-74 route-preview service tests exist.
- [ ] AC-75 PostGIS route-analysis integration tests remain green.
- [ ] AC-76 HTTP endpoint tests cover success/errors.
- [ ] AC-77 frontend route-builder unit tests exist.
- [ ] AC-78 Playwright covers end-to-end two-point route.
- [ ] AC-79 Playwright covers waypoint editing.
- [ ] AC-80 Playwright covers stale/error behavior where practical.

## Manual validation

- [ ] AC-81 dense-center route manually validated.
- [ ] AC-82 regular-urban route manually validated.
- [ ] AC-83 park route manually validated.
- [ ] AC-84 mobile viewport interaction manually validated.
- [ ] AC-85 desktop interaction manually validated.
- [ ] AC-86 potential exploration visualization is understandable without debug explanation.

## Performance

- [ ] AC-87 route-preview p50/p95 measured on representative fixtures.
- [ ] AC-88 latency breakdown is documented.
- [ ] AC-89 if p95 misses 1–2 s engineering target, bottleneck and disposition are documented.
- [ ] AC-90 representative mobile payload/render behavior is measured.

## Scope discipline

- [ ] AC-91 no authentication.
- [ ] AC-92 no UserStreetProgress.
- [ ] AC-93 no persistent Route created by preview.
- [ ] AC-94 no Walk lifecycle.
- [ ] AC-95 no GPS/GPX product capture.
- [ ] AC-96 no Places/Visits/Photos.
- [ ] AC-97 no Sharing/Recommendations/Social.

## Final DoD

- [ ] AC-98 reviewer can build/edit a pedestrian route from arbitrary map points and understand its route and potential exploration coverage.
- [ ] AC-99 Stage 2 validation report is complete.
- [ ] AC-100 Stage 3 can begin without reopening Stage 1 geo semantics.
