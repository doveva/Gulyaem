# Geo fixture tools

Reviewed maintenance helpers for Stage 1 geo source artifacts. They are not part of the runtime
import path.

## Responsibility

- obtain a new candidate snapshot for a named test area;
- convert the source XML into PBF with the repository Osmium image;
- print timestamp, checksum and object metadata for review.

## Boundaries and dependencies

Normal `make geo-import` is offline and never invokes these scripts. Refresh helpers require
network access and Docker, write candidates outside the repository by default, and never update a
manifest automatically.

## Main scenarios

Create a new dense-center candidate:

```bash
scripts/geo/refresh-spb-dense-center.sh
```

Create the combined Stage 1.5 validation candidate (with a hard 20 MB review gate):

```bash
scripts/geo/refresh-spb-stage1-validation.sh
```

An optional first argument selects the candidate output directory.

## Structure

- `refresh-spb-dense-center.sh` — fetch, PBF conversion and metadata inspection.
- `refresh-spb-stage1-validation.sh` — fetch and merge three validation areas, then check size,
  references and metadata.

## Run and verify

Review Osmium counts and spatial bounds, compare the candidate with the current fixture, then
replace the committed PBF and update all manifest provenance fields in one change. Run
`make geo-import` twice and the failure lifecycle test before accepting it.

## Configuration

The bbox and official OSM API URL are intentionally explicit in the area-specific script.

## Limitations and technical debt

This small-area snapshot workflow is temporary. A later import source should derive reviewed
extracts from a versioned regional dataset rather than scale the editing API to city-sized data.

## Related documents

- [`ADR-0001`](../../docs/adr/0001-osm-import-foundation.md)
- [`Dense center fixture`](../../data/test-areas/spb-dense-center/README.md)
