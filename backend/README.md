# Backend

Go modular-monolith backend for the Gulyaem geo exploration service. The current foundation
provides the HTTP process, PostgreSQL/PostGIS connectivity and a reproducible OSM PBF import with
version lifecycle metadata.

## Responsibility

- expose lightweight HTTP endpoints;
- own application and geo-domain use cases added during Stage 1;
- persist data through PostgreSQL/PostGIS;
- import committed OSM PBF fixtures into an owned `GeoDataVersion` lifecycle;
- emit structured application logs.

## Boundaries and dependencies

Transport code lives under `internal/transport`, infrastructure adapters under
`internal/platform`, and later application/domain packages must not depend on HTTP. The API uses
`chi` and `pgx`; database schema changes are plain SQL migrations executed by `golang-migrate`.
OSM parser types are isolated in `internal/platform/osm`; raw OSM entities are not persisted.

## Main scenarios

- `GET /health/live` checks that the API process is alive.
- `GET /health/ready` checks that PostgreSQL is reachable and PostGIS is enabled.
- `cmd/geo-import` verifies the fixture checksum, streams PBF objects and publishes a version.

## Structure

```text
cmd/api/                 API executable
cmd/geo-import/          offline OSM import executable
internal/geo/            geo domain and import application boundary
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
```

The first invocation reports `outcome=imported`; a repeat with the same checksum and normalization
version reports `outcome=already_ready` and the same version ID. An explicit PBF remains available
through the same environment-aware Make target:

```bash
make geo-import \
  GEO_IMPORT_FILE=/absolute/path/area.osm.pbf \
  GEO_CITY_CODE=spb
```

## Configuration

| Variable | Default | Purpose |
|---|---|---|
| `DATABASE_URL` | required | PostgreSQL connection string |
| `HTTP_ADDRESS` | `:8080` | API listen address |
| `ENVIRONMENT` | `development` | runtime environment name |
| `GEO_DATA_PATH` | `./data` | future geo-import input root |
| `GEO_TEST_AREA` | `spb-dense-center` | committed fixture selector |
| `GEO_IMPORT_FILE` | empty | explicit local PBF for `make geo-import` |
| `GEO_CITY_CODE` | `spb` | city code used with an explicit PBF |
| `NORMALIZATION_VERSION` | `stage1-v1` | identity of normalization rules used by import |
| `LOG_LEVEL` | `info` | `debug`, `info`, `warn`, or `error` |
| `CORS_ALLOWED_ORIGINS` | local Vite/Compose origins | comma-separated browser origins |

## Limitations and technical debt

Stage 1.2 deliberately counts and validates source objects but does not persist raw OSM entities or
generate `StreetSegment`. Topology and normalization behavior begin in Stage 1.3. The PBF parser
runs with `CGO_ENABLED=0`; performance is measured before changing parser or enabling native zlib.

## Related documents

- [`Stage 1 requirements`](../docs/stage%201/stage-1-requirements.md)
- [`Architecture contract`](../docs/stage%201/architecture-contract.md)
- [`ADR-0001`](../docs/adr/0001-osm-import-foundation.md)
