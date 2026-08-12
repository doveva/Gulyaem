# Stage 1 — Acceptance Criteria

## Import and versioning

- [x] AC-01 OSM dataset can be imported by one documented operation.
- [x] AC-02 Import creates `GeoDataVersion`.
- [x] AC-03 Failed import never becomes `READY`.
- [x] AC-04 Source checksum/version metadata is retained.
- [x] AC-05 Raw OSM IDs are not `StreetSegment` domain identity.

## StreetSegment model

- [x] AC-06 Own StreetSegments are generated.
- [x] AC-07 Every published segment has valid geometry.
- [x] AC-08 Every published segment has `length_m > 0`.
- [x] AC-09 Every published segment belongs to a `GeoDataVersion`.
- [x] AC-10 Every published segment has classification.
- [x] AC-11 `EXPLORE`, `ROUTABLE_ONLY`, `IGNORE` are supported.
- [x] AC-12 Classification reason is inspectable in debug mode.
- [x] AC-13 Primary segmentation follows topology, not fixed-length slicing.

## API

- [x] AC-14 Current GeoDataVersion endpoint works.
- [x] AC-15 bbox StreetSegment API works.
- [x] AC-16 segment-detail API works.
- [x] AC-17 district API works.
- [x] AC-18 segment collection is returned as GeoJSON.
- [x] AC-19 excessive bbox is rejected or safely limited.
- [x] AC-20 city-wide graph is not fetched by default.

## Frontend playground

- [x] AC-21 `/debug/geo` renders the base map.
- [x] AC-22 StreetSegments render over the map.
- [x] AC-23 each classification can be toggled independently.
- [x] AC-24 segment click/tap opens inspector.
- [x] AC-25 classification and length filters work.
- [x] AC-26 required attribution remains visible.
- [x] AC-27 viewport loading/error/empty states are handled.
- [x] AC-28 debug statistics are visible.

## Validation fixtures

- [x] AC-29 at least 3 reproducible test areas exist.
- [x] AC-30 test areas cover dense center, regular urban, park+residential.
- [x] AC-31 at least 3–5 sample walking routes exist.
- [x] AC-32 fixtures are stored in repository.

## Route/coverage experiments

- [x] AC-33 sample route can be overlaid.
- [x] AC-34 matched segments are visible.
- [x] AC-35 unmatched route fragments are visible.
- [x] AC-36 `covered_length_m` can be calculated.
- [x] AC-37 coverage threshold variants can be compared.
- [x] AC-38 completed/partial/not-covered are visually distinct.

## Routing-engine spike

- [x] AC-39 Valhalla evaluated or explicit exclusion documented.
- [x] AC-40 GraphHopper evaluated or explicit exclusion documented.
- [x] AC-41 OSRM evaluated or explicit exclusion documented.
- [x] AC-42 routing comparison uses shared fixtures.
- [x] AC-43 final routing-engine decision is documented.

## Testing

- [x] AC-44 classification unit tests exist.
- [x] AC-45 segmentation synthetic-graph tests exist.
- [x] AC-46 PostGIS integration tests exist.
- [x] AC-47 fixture invariants exist.
- [ ] AC-48 frontend interaction tests cover core debug flow.
- [ ] AC-49 manual visual validation completed for all areas.

## Stage-result decisions

- [x] AC-50 topology split rules documented.
- [x] AC-51 `max_segment_length_m` selected or explicitly rejected.
- [x] AC-52 initial WalkabilityProfile documented.
- [x] AC-53 initial coverage parameters documented, or blocker explicitly recorded.
- [x] AC-54 partial-coverage decision documented.
- [ ] AC-55 bbox + GeoJSON confirmed for Stage 2 or replacement justified.

## Scope discipline

- [x] AC-56 no authentication implemented.
- [x] AC-57 no UserStreetProgress/personal exploration implemented.
- [x] AC-58 no production Walk lifecycle implemented.
- [x] AC-59 no production GPS capture implemented.
- [ ] AC-60 no Places/Visits/Photos/Sharing/Recommendations/Social implemented.

## Final Definition of Done

- [ ] Geo Playground allows a reviewer to inspect real city areas, sample routes, segmentation, walkability and coverage and make an evidence-based decision that the model is suitable for Stage 2.
