# Saint Petersburg Stage 1 validation report

`cases.json` fixes three representative street-level viewports inside the larger source areas.
The complete source-area bboxes are probed separately to verify that the API either serves them or
rejects them with its documented safety limits.

`report.json` is the machine-readable Stage 1.7 evidence generated from the committed
`spb-stage1-validation` PBF and five sample routes. It records representative bbox GeoJSON
latency/payloads, feature and reason-code distributions, import invariants and Strict/Balanced/
Generous coverage results.

Regenerate it against the healthy local API:

```bash
make stage1-validate
```

The report contains local performance measurements rather than production sizing. Manual visual
observations and final product decisions are recorded separately in the Stage 1 validation report
and ADRs.
