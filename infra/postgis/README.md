# PostGIS image

Small multi-architecture database image for local Gulyaem development.

## Responsibility

Extend the official PostgreSQL 17 Debian image with the distribution's PostGIS 3 package so the
same Compose project runs natively on AMD64 and ARM64 machines.

## Boundaries and dependencies

The image changes only installed PostgreSQL extensions. Database initialization, credentials,
health checks, storage and extension activation remain owned by `compose.yaml` and SQL migrations.

## Main scenarios

Docker Compose builds this image before starting the `db` service. The first backend migration
then executes `CREATE EXTENSION postgis` in the application database.

## Structure

`Dockerfile` is intentionally the only build input.

## Run and verify

```bash
docker compose build db
docker compose up -d db
docker compose run --rm migrate
```

The API readiness endpoint verifies `PostGIS_Version()` after startup.

## Configuration

PostgreSQL major version and the matching package name are pinned together in the Dockerfile.
Database settings are documented in `docs/deployment/README.md`.

## Limitations and technical debt

This image is for local development, not a production database distribution. Upgrade testing and
backup/restore policy will be defined with the first remote environment.

## Related documents

- [`Deployment and operations`](../../docs/deployment/README.md)
