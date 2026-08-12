# Routing spike scripts

- `prepare.sh` validates the shared PBF, creates ignored local graph directories and invalidates a
  graph whose recorded source or artifact checksum is stale.
- `finalize-metadata.sh` runs after Valhalla is healthy and atomically writes graph-bound
  `routing-dataset.json` metadata, including the PBF and `valhalla_tiles.tar` SHA-256 values.
- `prepare_test.sh` proves in an isolated temporary directory that stale graph artifacts are
  invalidated while a graph matching both recorded checksums is preserved.
- `up.sh` starts each pinned engine, waits for readiness and records relative setup/resource metrics.
- `measure.py` samples container memory while graph preparation or startup is running.
- `reset.sh` removes only generated `.routing` data so a cold graph build can be measured again.

Use the Make targets documented in the project README instead of calling these files directly.
The committed fixture contract is in `data/routing-spike/spb-stage1`; the final JSON report is
written to `frontend/public/routing-spike/comparison.json` for the debug UI.

Stage 2 also uses the Valhalla portion as a normal development dependency: `make up` invokes
`prepare.sh`, force-recreates Valhalla when needed, waits for `/status`, finalizes metadata, then
starts the API. The API hashes the mounted graph artifact and compares its source checksum with the
current READY `GeoDataVersion`; it never treats environment metadata or engine edge IDs as proof of
dataset identity.
