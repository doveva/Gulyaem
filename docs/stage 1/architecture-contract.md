# Stage 1 — Architecture Contract

Этот файл фиксирует решения, которые coding agent не должен переопределять в ходе реализации Stage 1 без отдельного изменения требований.

## Fixed decisions

### Backend

- Go.
- Modular monolith.
- Separate `cmd/api` and `cmd/geo-import` executables.
- PostgreSQL + PostGIS.
- `pgx` preferred for PostgreSQL access.
- HTTP stack should stay lightweight.

### Frontend

- React + TypeScript.
- Vite.
- MapLibre GL JS.
- Debug-first Geo Playground.
- Initial data transport: bbox + GeoJSON.

### Geo

- OSM is upstream only.
- Own `GeoDataVersion`.
- Own `StreetSegment`.
- `StreetSegment != OSM Way`.
- Segment identity is not eternal across geo versions.
- Internal classification:
  - `EXPLORE`
  - `ROUTABLE_ONLY`
  - `IGNORE`
- Classification is centralized in normalization.
- Topology-based segmentation is primary.
- Fixed-length segmentation is not primary.
- Source/debug OSM metadata may exist, but must not leak as core domain API semantics.

### Routing

- Do not implement a custom routing engine.
- Compare Valhalla / GraphHopper / OSRM.
- Routing engine IDs never become StreetSegment IDs.

## Dependency direction

Preferred logical dependency:

```text
transport/api
    ↓
application/use cases
    ↓
geo domain
    ↑
repositories / adapters
```

Import-specific infrastructure may depend on OSM parser/adapters but domain code must not require HTTP.

Frontend-specific models may adapt API responses but React component concepts must not shape backend geo entities.

## Persistence rules

Published geo data is version-aware.

A failed import:

```text
must not produce READY GeoDataVersion
```

Published StreetSegments:

```text
must have:
- geometry
- positive length
- classification
- geo_data_version_id
```

## Map API rules

- Never send the entire city graph by default.
- Query by viewport/bbox.
- Enforce bbox/feature limits.
- Return internal domain IDs.
- Debug source metadata is opt-in/internal-only.

## Change policy

If implementation evidence contradicts a fixed decision, document the evidence in a Stage 1 finding. Do not silently rewrite the architecture in code.
