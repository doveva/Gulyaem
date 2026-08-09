# Geo data

Local input directory for reproducible Stage 1 geo extracts and fixtures. It is mounted read-only
at `/data` in the API container and exposed to local processes through `GEO_DATA_PATH`.

## Responsibility

This directory contains deliberately small, repository-safe test-area artifacts. Full city
extracts and generated databases do not belong in Git.

## Boundaries and dependencies

OSM remains upstream input rather than a domain model. Import code belongs to the backend; this
directory stores inputs and fixture descriptions only.

## Main scenarios

`test-areas/spb-dense-center` is the first immutable OSM PBF snapshot and manifest.

## Structure

Each fixture directory owns its PBF, manifest, checksum/provenance and area-specific README.

## Run and verify

The directory is mounted automatically by `docker compose up`. Host-mode commands use
`GEO_DATA_PATH=./data` relative to their working directory unless explicitly overridden.

## Configuration

Use `GEO_DATA_PATH` to select a different input root and `GEO_TEST_AREA` to select a future test
area.

## Limitations and technical debt

Only small reviewed PBF snapshots belong in Git. A full city or regional extract must be obtained
outside the repository and referenced by checksum.

## Related documents

- [`Stage 1 requirements`](../docs/stage%201/stage-1-requirements.md)
