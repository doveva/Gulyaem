# Routing spike infrastructure

Pinned local containers for the Stage 1.6 comparison of Valhalla, GraphHopper and OSRM. They are
isolated behind the optional Compose profile `routing-spike` and all consume the same committed
OSM PBF.

## Responsibility

- build a reproducible GraphHopper image and pedestrian profile;
- configure engine graph directories without adding generated data to Git;
- support the benchmark runner, not production deployment.

Valhalla and OSRM use pinned upstream images directly from `compose.yaml`. GraphHopper uses the
Dockerfile and config in this directory because its release artifact is distributed as a JAR.

## Run and verify

Use the repository-level targets:

```bash
make routing-spike
make routing-down
make routing-reset
```

`routing-reset` removes only ignored `.routing/` graphs so the next run measures a cold build.
Resource measurements are relative Docker Desktop values and must not be used as production
sizing.

## Related documents

- [`Routing engine ADR`](../../docs/adr/0006-routing-engine-valhalla.md)
- [`Routing fixture`](../../data/routing-spike/spb-stage1/README.md)
- [`Routing scripts`](../../scripts/routing/README.md)
