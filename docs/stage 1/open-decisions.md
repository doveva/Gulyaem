# Stage 1 — Open Decisions

These questions are intentionally **not predetermined**. Stage 1 exists partly to answer them.

Coding agents should keep these points configurable/replaceable when practical.

## OD-01 WalkabilityProfile mapping

Need evidence-based OSM → internal classification mapping.

Questions:

- residential → EXPLORE?
- pedestrian → EXPLORE?
- footway → EXPLORE?
- path → context dependent?
- service → EXPLORE or ROUTABLE_ONLY?
- courtyards?
- alleys/passages?
- private/restricted?
- park paths?
- technical connectors?

Expected result: ADR/update with explicit rules and exceptions.

## OD-02 Maximum segment length

Options include:

```text
no artificial maximum
configurable maximum
context-specific maximum
```

Do not hardcode a product value before visual validation.

## OD-03 Additional artificial split points

Need to determine whether topology-only segments are sometimes too long or semantically coarse.

## OD-04 Street normalization depth

Determine how much `Street` normalization is needed now versus later.

`StreetSegment.street_id` may remain nullable during Stage 1.

## OD-05 Coverage threshold

Need to determine:

```text
coverage_ratio
min_required_m
max_required_m
```

through real-route experiments.

## OD-06 Partial coverage

Decide whether MVP needs cumulative partial coverage or whether binary completion is enough after proper segmentation.

## OD-07 Routing engine

Candidates:

```text
Valhalla
GraphHopper
OSRM
```

Decision based on same fixtures, not preference alone.

## OD-08 GeoJSON scalability

Initial decision is bbox + GeoJSON.

Need to measure whether representative test-area payload/rendering is sufficient for next stage.

Do not introduce vector tiles solely as speculative optimization.

## OD-09 Base-map style/provider

Must support:

- MapLibre;
- adequate visual contrast;
- visible required attribution;
- replaceability;
- no domain coupling.

Exact provider/style remains implementation choice unless operational/licensing constraints require ADR.
