# Valhalla development runtime

Stage 2 uses the pinned `ghcr.io/valhalla/valhalla-scripted:3.7.0` image as a normal Compose dependency.
The graph and the current internal `GeoDataVersion` are both built from:

```text
data/test-areas/spb-stage1-validation/spb-stage1-validation.osm.pbf
sha256 05fa864bb753ffc4b2c632deae28e6b6ed80b8e677f65a9689d786596aa0e8ae
profile pedestrian
```

Prepare and run the complete development stack:

```bash
make up
```

`make up` runs this chain:

```text
verified source PBF
  -> invalidate incompatible old graph
  -> build/start Valhalla
  -> hash valhalla_tiles.tar
  -> atomically publish routing-dataset.json
  -> start API with graph and metadata mounted read-only
```

`scripts/routing/prepare.sh` verifies the PBF against its committed manifest. If the source checksum
changed, the previous metadata is incomplete, or the existing graph artifact no longer matches its
recorded checksum, the script removes the generated graph before Valhalla starts. A separate
`routing-metadata` one-shot runs only after Valhalla is healthy and writes metadata containing:

- engine and engine version;
- city and pedestrian profile;
- source PBF checksum;
- graph artifact filename and SHA-256;
- graph build completion timestamp.

The API reads `/routing/routing-dataset.json`; it does not accept the routing source checksum from an
environment variable. It verifies the SHA-256 of the mounted graph artifact, then compares the
metadata city/source checksum with the current READY `GeoDataVersion`. A mismatch or corrupt/missing
graph keeps `/health/ready` at HTTP 503. The same mismatch returns HTTP 409 from a route-preview
request, before `/route` or exploration analysis is called.

Generated graphs are not committed. Source changes are invalidated automatically. A manual cold
rebuild remains available for performance measurements or recovery:

```bash
make routing-reset
make up
```
