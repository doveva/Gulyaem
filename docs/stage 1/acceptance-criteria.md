# Stage 1 — Acceptance Criteria

## Import and versioning

- [x] AC-01 OSM dataset can be imported by one documented operation.
- [x] AC-02 Import creates `GeoDataVersion`.
- [x] AC-03 Failed import never becomes `READY`.
- [x] AC-04 Source checksum/version metadata is retained.
- [ ] AC-05 Raw OSM IDs are not `StreetSegment` domain identity.

## StreetSegment model

- [ ] AC-06 Own StreetSegments are generated.
- [ ] AC-07 Every published segment has valid geometry.
- [ ] AC-08 Every published segment has `length_m > 0`.
- [ ] AC-09 Every published segment belongs to a `GeoDataVersion`.
- [ ] AC-10 Every published segment has classification.
- [ ] AC-11 `EXPLORE`, `ROUTABLE_ONLY`, `IGNORE` are supported.
- [ ] AC-12 Classification reason is inspectable in debug mode.
- [ ] AC-13 Primary segmentation follows topology, not fixed-length slicing.

## API

- [ ] AC-14 Current GeoDataVersion endpoint works.
- [ ] AC-15 bbox StreetSegment API works.
- [ ] AC-16 segment-detail API works.
- [ ] AC-17 district API works.
- [ ] AC-18 segment collection is returned as GeoJSON.
- [ ] AC-19 excessive bbox is rejected or safely limited.
- [ ] AC-20 city-wide graph is not fetched by default.

## Frontend playground

- [x] AC-21 `/debug/geo` renders the base map.
- [ ] AC-22 StreetSegments render over the map.
- [ ] AC-23 each classification can be toggled independently.
- [ ] AC-24 segment click/tap opens inspector.
- [ ] AC-25 classification and length filters work.
- [x] AC-26 required attribution remains visible.
- [ ] AC-27 viewport loading/error/empty states are handled.
- [ ] AC-28 debug statistics are visible.

## Validation fixtures

- [ ] AC-29 at least 3 reproducible test areas exist.
- [ ] AC-30 test areas cover dense center, regular urban, park+residential.
- [ ] AC-31 at least 3–5 sample walking routes exist.
- [ ] AC-32 fixtures are stored in repository.

## Route/coverage experiments

- [ ] AC-33 sample route can be overlaid.
- [ ] AC-34 matched segments are visible.
- [ ] AC-35 unmatched route fragments are visible.
- [ ] AC-36 `covered_length_m` can be calculated.
- [ ] AC-37 coverage threshold variants can be compared.
- [ ] AC-38 completed/partial/not-covered are visually distinct.

## Routing-engine spike

- [ ] AC-39 Valhalla evaluated or explicit exclusion documented.
- [ ] AC-40 GraphHopper evaluated or explicit exclusion documented.
- [ ] AC-41 OSRM evaluated or explicit exclusion documented.
- [ ] AC-42 routing comparison uses shared fixtures.
- [ ] AC-43 final routing-engine decision is documented.

## Testing

- [ ] AC-44 classification unit tests exist.
- [ ] AC-45 segmentation synthetic-graph tests exist.
- [ ] AC-46 PostGIS integration tests exist.
- [ ] AC-47 fixture invariants exist.
- [ ] AC-48 frontend interaction tests cover core debug flow.
- [ ] AC-49 manual visual validation completed for all areas.

## Stage-result decisions

- [ ] AC-50 topology split rules documented.
- [ ] AC-51 `max_segment_length_m` selected or explicitly rejected.
- [ ] AC-52 initial WalkabilityProfile documented.
- [ ] AC-53 initial coverage parameters documented, or blocker explicitly recorded.
- [ ] AC-54 partial-coverage decision documented.
- [ ] AC-55 bbox + GeoJSON confirmed for Stage 2 or replacement justified.

## Scope discipline

- [ ] AC-56 no authentication implemented.
- [ ] AC-57 no UserStreetProgress/personal exploration implemented.
- [ ] AC-58 no production Walk lifecycle implemented.
- [ ] AC-59 no production GPS capture implemented.
- [ ] AC-60 no Places/Visits/Photos/Sharing/Recommendations/Social implemented.

## Final Definition of Done

- [ ] Geo Playground allows a reviewer to inspect real city areas, sample routes, segmentation, walkability and coverage and make an evidence-based decision that the model is suitable for Stage 2.
