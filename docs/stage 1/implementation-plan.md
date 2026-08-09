# Stage 1 — Recommended Implementation Plan

Порядок ориентирован на быстрый feedback и минимизацию невидимой geo-работы.

## Stage 1.1 — Foundation bootstrap

Deliverables:

```text
Go API
React + TypeScript app
PostgreSQL + PostGIS
MapLibre
local orchestration
migrations
health endpoint
/debug/geo shell
```

Done when:

- app launches reproducibly;
- PostGIS connection works;
- MapLibre renders base map;
- frontend can call backend.

---

## Stage 1.2 — OSM import foundation

Deliverables:

```text
cmd/geo-import
GeoDataVersion
PBF input
source checksum
import lifecycle
initial normalization boundary
```

First target should be a small test-area extract, not all Saint Petersburg.

Done when:

- same fixture can be imported repeatedly;
- import creates a version;
- failed import is not READY;
- metrics/report are emitted.

---

## Stage 1.3 — Topology + StreetSegment

Deliverables:

```text
pedestrian candidates
normalized attributes
topology
WalkabilityProfile
StreetSegment generator
PostGIS persistence
spatial indexes
```

Start from explicit synthetic graph tests before tuning real OSM.

Done when:

- test fixtures create predictable segmentation;
- real test area generates valid positive-length segments;
- classification reason can be inspected.

---

## Stage 1.4 — bbox API + Geo Playground

Deliverables:

```text
GET geo version
GET segments bbox
GET segment detail
GET districts
GeoJSON
layer controls
filters
segment inspector
statistics
```

Done when:

- engineer can inspect all classifications on map;
- no city-wide accidental fetch;
- basic viewport remains interactive.

---

## Stage 1.5 — Sample routes + coverage

Deliverables:

```text
3–5 route fixtures
route overlay
prototype segment matching
coverage metrics
threshold controls/profiles
coverage visualization
```

Do not turn this into production Walk/Route domain.

Done when:

- same sample route can be compared under different thresholds;
- unmatched fragments are visible;
- route matching errors can be inspected.

---

## Stage 1.6 — Routing engine comparison

Run comparable fixtures against:

```text
Valhalla
GraphHopper
OSRM
```

Record:

- route output;
- distance;
- latency;
- deployment/setup;
- resource footprint;
- pedestrian-quality notes;
- future map-matching support.

Conclude with ADR decision.

---

## Stage 1.7 — Validation and freeze

Review all three test areas.

Produce:

```text
validation report
routing ADR
StreetSegment parameter ADR
coverage parameter ADR
map-delivery confirmation/revision
```

Only after this freeze begin Stage 2.
