# Stage 1 validation tooling

`stage1.py` produces the reproducible machine-readable evidence used by the Stage 1.7 freeze. It
queries the running API for every bbox from the committed validation manifest, measures warmed
GeoJSON response latency and payload size, checks fixture/import invariants, summarizes reason
codes and runs all sample routes with Strict, Balanced and Generous coverage profiles.

Run it through the repository-level target:

```bash
make stage1-validate
```

The command requires an already imported validation fixture and a healthy local API. It writes
`data/validation/spb-stage1/report.json`; measurements describe the local host and are not an SLA.

