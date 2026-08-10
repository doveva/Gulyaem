# Backend

Go modular-monolith backend for the Gulyaem geo exploration service. The current implementation
provides the HTTP process, PostgreSQL/PostGIS connectivity and a reproducible OSM PBF import that
builds versioned, classified `StreetSegment` geometry and independently versioned administrative
districts.

## Responsibility

- expose lightweight HTTP endpoints;
- own application and geo-domain use cases added during Stage 1;
- persist data through PostgreSQL/PostGIS;
- import committed OSM PBF fixtures into an owned `GeoDataVersion` lifecycle;
- normalize pedestrian semantics and generate topology-based `StreetSegment`;
- import and publish normalized administrative `District` boundaries;
- analyze committed sample routes with sequential matching and radius coverage without persistence;
- run a reproducible routing-engine comparison without importing engine graph identity into the domain;
- emit structured application logs.

## Boundaries and dependencies

Transport code lives under `internal/transport`, infrastructure adapters under
`internal/platform`, and later application/domain packages must not depend on HTTP. The API uses
`chi` and `pgx`; database schema changes are plain SQL migrations executed by `golang-migrate`.
OSM parser types are isolated in `internal/platform/osm`; raw OSM entities are not persisted.

## Main scenarios

- `GET /health/live` checks that the API process is alive.
- `GET /health/ready` checks that PostgreSQL is reachable and PostGIS is enabled.
- `GET /api/v1/cities/{cityId}/geo-version` returns the current `READY` version.
- `GET /api/v1/geo/segments` returns filtered GeoJSON for a bounded viewport with statistics.
- `GET /api/v1/geo/segments/{segmentId}` returns current or historical segment details; source OSM
  metadata requires `debug=true` and is disabled in production.
- `GET /api/v1/geo/districts` returns the current district layer for a bounded viewport.
- `GET /api/v1/geo/sample-routes` lists version-aware Stage 1.5 route fixtures.
- `POST /api/v1/geo/sample-routes/{routeId}/analyze` returns normalized/matched/unmatched geometry,
  exact coverage, provenance and metrics for a selected profile.
- `cmd/geo-import` verifies the fixture checksum, builds segments in memory and atomically publishes
  them with a version.
- `cmd/district-import` verifies the GeoJSON fixture and atomically publishes an independent
  `DistrictDataVersion`.
- `cmd/routing-spike` benchmarks pinned engines and runs returned geometries through the existing
  `StreetSegment` matcher.

## Structure

```text
cmd/api/                 API executable
cmd/geo-import/          offline OSM import executable
cmd/district-import/     offline administrative district import executable
cmd/routing-spike/       offline Stage 1.6 engine comparison executable
internal/geo/            geo domain and import application boundary
internal/geo/segmenting/ WalkabilityProfile and topology-based segmentation
internal/geo/querying/    bounded read use cases and viewport statistics
internal/geo/routeanalysis/ stateless sequential matching and coverage semantics
internal/routingspike/   engine adapters, benchmark metrics and report generation
internal/config/         environment configuration
internal/platform/       infrastructure adapters
internal/transport/      HTTP transport
migrations/              golang-migrate SQL migrations
```

## Run and verify

From the repository root, start and migrate the database first:

```bash
docker compose up -d db
docker compose run --rm migrate
```

Then run the API from the repository root. `make` reads the root `.env`, derives `DATABASE_URL`
from `POSTGRES_*` (including `POSTGRES_PORT`) and points `GEO_DATA_PATH` at the root `data/`
directory:

```bash
make api
```

Verify it with `curl http://localhost:8080/health/ready`. Run tests with
`CGO_ENABLED=0 go test ./...`.

An explicit connection string remains supported and takes precedence:

```bash
make api DATABASE_URL='postgres://user:password@localhost:55432/gulyaem?sslmode=disable'
```

Import the default committed fixture after migrations:

```bash
make geo-import
make district-import
```

The first invocation reports `outcome=imported`; a repeat with the same checksum and normalization
version reports `outcome=already_ready` and the same version ID. An explicit PBF remains available
through the same environment-aware Make target:

```bash
make geo-import \
  GEO_IMPORT_FILE=/absolute/path/area.osm.pbf \
  GEO_CITY_CODE=spb
```

An existing Stage 1.2 `.env` may still contain `NORMALIZATION_VERSION=stage1-v1`. Change it to
`stage1-segments-v1`, or pass the new value explicitly, so the segment-producing semantics create a
new version:

```bash
make geo-import NORMALIZATION_VERSION=stage1-segments-v1
```

For the committed fixture and `max_segment_length_m=0`, the Stage 1.3 baseline is 6,558 segments:
2,649 `EXPLORE`, 2,338 `ROUTABLE_ONLY`, and 1,571 `IGNORE`.

Run the Stage 1.6 comparison from the repository root:

```bash
make routing-spike
```

This starts the isolated Compose profile, measures relative local setup/resources, runs the same
route and map-matching fixtures against all engines, and updates the frontend report.

## Configuration

| Variable | Default | Purpose |
|---|---|---|
| `DATABASE_URL` | required | PostgreSQL connection string |
| `HTTP_ADDRESS` | `:8080` | API listen address |
| `ENVIRONMENT` | `development` | runtime environment name |
| `GEO_DATA_PATH` | `./data` | future geo-import input root |
| `GEO_TEST_AREA` | `spb-stage1-validation` | combined Stage 1.5 fixture selector; dense-center remains available for regression |
| `GEO_IMPORT_FILE` | empty | explicit local PBF for `make geo-import` |
| `GEO_CITY_CODE` | `spb` | city code used with an explicit PBF |
| `NORMALIZATION_VERSION` | `stage1-segments-v1` | identity of normalization rules used by import |
| `MAX_SEGMENT_LENGTH_M` | `0` | experimental artificial split; `0` disables it |
| `DISTRICT_TEST_AREA` | `spb-administrative-districts` | committed district fixture selector |
| `DISTRICT_NORMALIZATION_VERSION` | `stage1-districts-v1` | district import identity |
| `LOG_LEVEL` | `info` | `debug`, `info`, `warn`, or `error` |
| `CORS_ALLOWED_ORIGINS` | local Vite/Compose origins | comma-separated browser origins |

## Limitations and technical debt

Raw OSM entities remain only in PBF and temporary import memory; district source geometry and
sample routes remain committed fixtures. The bbox endpoint rejects viewports larger than 25 km²
and more than 10,000 matching segments instead of truncating them. Stage 1.5 route matching and
coverage are stateless request-time experiments, not production Walk/progress persistence.
`Street.street_id` remains nullable and pedestrian areas/indoor corridors are not converted into
explorable linear geometry. The PBF parser runs with `CGO_ENABLED=0`; performance is measured
before changing parser or enabling native zlib. Routing-spike resource measurements are local
Docker Desktop comparisons, not production sizing.

## Related documents

- [`Stage 1 requirements`](../docs/stage%201/stage-1-requirements.md)
- [`Architecture contract`](../docs/stage%201/architecture-contract.md)
- [`ADR-0001`](../docs/adr/0001-osm-import-foundation.md)
- [`ADR-0002`](../docs/adr/0002-street-segment-topology-and-walkability.md)
- [`ADR-0003`](../docs/adr/0003-geo-playground-bbox-api.md)
- [`ADR-0004`](../docs/adr/0004-versioned-administrative-districts.md)
- [`ADR-0005`](../docs/adr/0005-sample-route-matching-and-radius-coverage.md)
- [`ADR-0006`](../docs/adr/0006-routing-engine-valhalla.md)
