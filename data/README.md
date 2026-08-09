# Geo data

Local input directory for reproducible Stage 1 geo extracts and fixtures. It is mounted read-only
at `/data` in the API container and exposed to local processes through `GEO_DATA_PATH`.

## Responsibility

This directory will contain deliberately small, repository-safe test-area artifacts introduced
in Stage 1.2 and later. Full city extracts and generated databases do not belong in Git.

## Boundaries and dependencies

OSM remains upstream input rather than a domain model. Import code belongs to the backend; this
directory stores inputs and fixture descriptions only.

## Main scenarios

Stage 1.1 only establishes the mount point. No geo artifact has been selected or committed yet.

## Structure

Fixture subdirectories and their provenance/checksums will be documented when the OSM import is
implemented.

## Run and verify

The directory is mounted automatically by `docker compose up`. Host-mode commands use
`GEO_DATA_PATH=./data` relative to their working directory unless explicitly overridden.

## Configuration

Use `GEO_DATA_PATH` to select a different input root and `GEO_TEST_AREA` to select a future test
area.

## Limitations and technical debt

The precise repository policy for binary PBF fixtures is deferred to Stage 1.2, when real fixture
sizes and provenance are known.

## Related documents

- [`Stage 1 requirements`](../docs/stage%201/stage-1-requirements.md)
