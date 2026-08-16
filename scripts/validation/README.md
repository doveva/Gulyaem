# Stage 1 validation tooling

`stage1.py` produces the reproducible machine-readable evidence used by the Stage 1.7 freeze. Its
coverage contract is frozen at Strict/Balanced/Generous radii `35/50/100 м` and Balanced ratio
`0.6`. It queries every bbox from the committed validation manifest, measures warmed GeoJSON
response latency and payload size, checks fixture/import invariants, summarizes reason codes and
runs all sample routes.

Run it through the repository-level target:

```bash
make stage1-validate
```

The command requires an already imported validation fixture and a Stage 1-compatible healthy API.
It replaces `data/validation/spb-stage1/report.json` only when every frozen check passes. A failed
run writes `report.failed.json`, preserving the accepted historical report.

ADR-0014 coverage-v2 validation is intentionally separate:

```bash
make stage3-coverage-validate
```

`coverage_v2.py` uses radii `50/100/200 м`, Balanced ratio `0.4`, records
`route-analysis-v2`, and writes only under `data/validation/spb-stage3-coverage-v2/`. Both runners
reuse the committed Stage 1 geographic fixtures so their coverage results remain comparable.
