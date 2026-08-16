# Stage 3 coverage-v2 validation evidence

This directory is reserved for machine-readable validation of ADR-0014 against the committed
Stage 1 geographic fixture and sample routes. It does not replace or reinterpret the Stage 1.7
freeze report.

Generate `report.json` against a healthy current API:

```bash
make stage3-coverage-validate
```

The contract is `route-analysis-v2` with Strict/Balanced/Generous radii `50/100/200 м`, Balanced
ratio `0.4`, and the existing `225 м` analysis context. Local latency measurements are evidence for
that run, not a production SLA.
