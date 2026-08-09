# Osmium fixture tool

Local multi-architecture utility image used only to convert or inspect committed OSM fixtures.

## Responsibility

Provide a pinned Debian environment with `osmium-tool`, avoiding a host package-manager
requirement for fixture maintenance.

## Boundaries and dependencies

Osmium is not part of the application runtime and does not perform imports. `cmd/geo-import`
reads the resulting PBF through the Go source adapter.

## Main scenarios

- convert a deliberately small OSM XML snapshot to PBF;
- inspect PBF metadata during fixture maintenance.

## Structure

`Dockerfile` is the only runtime input. Fixture-specific commands and provenance live beside the
fixture in `data/test-areas`.

## Run and verify

```bash
docker build -t gulyaem/osmium:bookworm infra/osmium
docker run --rm gulyaem/osmium:bookworm --version
```

## Configuration

Input and output directories are supplied as bind mounts per maintenance command.

## Limitations and technical debt

Live OSM download is intentionally not part of the normal import path. Refreshing a fixture is a
reviewed maintenance operation because it changes the committed checksum and validation input.

## Related documents

- [`ADR-0001`](../../docs/adr/0001-osm-import-foundation.md)
