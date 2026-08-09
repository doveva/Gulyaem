# Backend

Go modular-monolith backend for the Gulyaem geo exploration service. Stage 1.1 provides the
HTTP process, PostgreSQL/PostGIS connectivity and the migration boundary needed by later geo
modules.

## Responsibility

- expose lightweight HTTP endpoints;
- own application and geo-domain use cases added during Stage 1;
- persist data through PostgreSQL/PostGIS;
- emit structured application logs.

## Boundaries and dependencies

Transport code lives under `internal/transport`, infrastructure adapters under
`internal/platform`, and later application/domain packages must not depend on HTTP. The API uses
`chi` and `pgx`; database schema changes are plain SQL migrations executed by `golang-migrate`.

## Main scenarios

- `GET /health/live` checks that the API process is alive.
- `GET /health/ready` checks that PostgreSQL is reachable and PostGIS is enabled.

## Structure

```text
cmd/api/                 API executable
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

Verify it with `curl http://localhost:8080/health/ready`. Run tests with `go test ./...`.

An explicit connection string remains supported and takes precedence:

```bash
make api DATABASE_URL='postgres://user:password@localhost:55432/gulyaem?sslmode=disable'
```

## Configuration

| Variable | Default | Purpose |
|---|---|---|
| `DATABASE_URL` | required | PostgreSQL connection string |
| `HTTP_ADDRESS` | `:8080` | API listen address |
| `ENVIRONMENT` | `development` | runtime environment name |
| `GEO_DATA_PATH` | `./data` | future geo-import input root |
| `GEO_TEST_AREA` | empty | optional future fixture selector |
| `LOG_LEVEL` | `info` | `debug`, `info`, `warn`, or `error` |
| `CORS_ALLOWED_ORIGINS` | local Vite/Compose origins | comma-separated browser origins |

## Limitations and technical debt

Stage 1.1 intentionally exposes only health endpoints. Geo models, import and bbox APIs begin in
the subsequent implementation stages.

## Related documents

- [`Stage 1 requirements`](../docs/stage%201/stage-1-requirements.md)
- [`Architecture contract`](../docs/stage%201/architecture-contract.md)
