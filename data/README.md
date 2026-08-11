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
`test-areas/spb-stage1-validation` combines the three Stage 1.5 matching and coverage areas.
`districts/spb-administrative-districts` contains the full boundaries of the 18 Saint Petersburg
administrative districts as a checksummed GeoJSON fixture.

## Structure

Each fixture directory owns its source artifact, manifest, checksum/provenance and area-specific
README. Street graph fixtures live below `test-areas`; district fixtures live below `districts`.
Curated immutable walking lines used by analysis experiments live below `sample-routes`.
The shared routing-engine benchmark contract, waypoint selection and pinned engine versions live
below `routing-spike`; generated graphs do not belong in this directory.
Stage 1.7 representative viewports and generated validation evidence live below `validation`.

## Run and verify

The directory is mounted automatically by `docker compose up`. Host-mode commands use
`GEO_DATA_PATH=./data` relative to their working directory unless explicitly overridden.

## Configuration

Use `GEO_DATA_PATH` to select a different input root, `GEO_TEST_AREA` to select a street graph
fixture and `DISTRICT_TEST_AREA` to select a district fixture.

## Limitations and technical debt

Only small reviewed PBF snapshots belong in Git. A full city or regional extract must be obtained
outside the repository and referenced by checksum.

## Related documents

- [`Stage 1 requirements`](../docs/stage%201/stage-1-requirements.md)
