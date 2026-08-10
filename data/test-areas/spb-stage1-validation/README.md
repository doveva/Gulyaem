# Saint Petersburg Stage 1 validation fixture

Immutable OSM input for validating route matching and exploration coverage in three deliberately
different environments: the dense historic center, the Kalininsky urban corridor and Sosnovka
park with its residential edges.

The committed PBF is a merge of five small OSM API snapshots. The importer clips streets to the
three named `areas` from `manifest.json`; the outer `bbox` is provenance only and does not include
the empty space between the areas. The original OSM nodes, ways and relations remain only in the
PBF.

## Files

- `spb-stage1-validation.osm.pbf` — immutable combined import input;
- `manifest.json` — named clip areas, provenance, attribution and expected SHA-256.

The refresh helper writes a candidate outside the repository, checks references and enforces the
20 MB review gate. It never replaces these files automatically:

```bash
scripts/geo/refresh-spb-stage1-validation.sh
```

## Attribution

© OpenStreetMap contributors. Data is available under the Open Data Commons Open Database License
(ODbL): https://www.openstreetmap.org/copyright

## Related documents

- [`ADR-0001`](../../../docs/adr/0001-osm-import-foundation.md)
- [`Geo fixture tools`](../../../scripts/geo/README.md)
