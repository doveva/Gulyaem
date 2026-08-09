# Saint Petersburg dense center fixture

Immutable Stage 1 snapshot for the first real OSM import and later dense-center topology
validation.

## Area

```text
west:  30.3000
south: 59.9300
east:  30.3300
north: 59.9450
```

The snapshot was requested from the OpenStreetMap API 0.6 map endpoint and converted locally to
PBF with Osmium. Contributor `user`, `uid` and `changeset` attributes are removed because the
import does not use them. Geometry, tags, element versions and timestamps remain intact. Normal
imports read the committed PBF and never contact that endpoint.

## Files

- `spb-dense-center.osm.pbf` — immutable import input;
- `manifest.json` — bbox, provenance, attribution and expected SHA-256.

## Refresh policy

Refreshing is a reviewed maintenance operation, not part of `make geo-import`. A refresh must
replace the PBF, update every manifest provenance field and checksum, and pass the import
idempotency/failure lifecycle checks.

## Attribution

© OpenStreetMap contributors. Data is available under the Open Data Commons Open Database License
(ODbL): https://www.openstreetmap.org/copyright

## Related documents

- [`ADR-0001`](../../../docs/adr/0001-osm-import-foundation.md)
