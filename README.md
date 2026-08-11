# ГуляЕм — Stage 2 Code Context Pack

Этот пакет фиксирует scope **Stage 2 — Manual Route & Exploration Preview**.

Stage 1 считается завершённым и даёт Stage 2 следующие входные решения:

- internal `StreetSegment` graph;
- `WalkabilityProfile v1`;
- topology-first segmentation;
- `max_segment_length_m=0`;
- `StreetSegment` identity scoped by `GeoDataVersion`;
- grade-aware route matching / coverage;
- default coverage profile **Balanced: 50 м / 0.6 / 15–80 м**;
- `PARTIAL` сохраняется;
- routing engine: **Valhalla**;
- map delivery: **bbox + GeoJSON** на раннем этапе;
- React + TypeScript + MapLibre frontend;
- Go + PostgreSQL/PostGIS backend.

## Главная цель Stage 2

Получить первую пользовательскую capability поверх реального geo core:

```text
Map
 ↓
Manual Waypoints
 ↓
Valhalla Pedestrian Route
 ↓
Normalized Route Geometry
 ↓
StreetSegment Matching
 ↓
Coverage Preview
 ↓
Visual Route Preview
```

Главный validation question:

> Понятно ли пользователю по карте, какой маршрут он построил и какие части городской сети эта прогулка потенциально позволит исследовать?

## Важная граница Stage 2

Stage 2 **не имеет персонального exploration state**.

Поэтому UI и API не должны называть покрываемые сегменты:

```text
"новыми для пользователя"
```

или вычислять:

```text
new vs already explored
```

до появления `UserStreetProgress` в Stage 3.

Stage 2 показывает:

- route;
- distance;
- duration;
- matched/unmatched route;
- `EXPLORE` segments, которые будут `COMPLETED`;
- `PARTIAL` segments;
- routing-only connectivity;
- potential exploration coverage.

## Основные документы

- `AGENTS.md` — scope guard для coding agent.
- `docs/stage-2-requirements.md` — полный Stage 2 scope.
- `docs/architecture-contract.md` — фиксированные архитектурные решения.
- `docs/api-contract.md` — целевой HTTP contract.
- `docs/frontend-flow.md` — route-builder UX/state model.
- `docs/implementation-plan.md` — рекомендуемый порядок реализации.
- `docs/acceptance-criteria.md` — Definition of Done.
- `docs/open-decisions.md` — вопросы, которые Stage 2 должен проверить.
- `docs/validation-plan.md` — automated + manual validation.

## Stage 2 Definition of Done

Stage 2 завершён, когда пользователь может на `/map` интерактивно задать start/destination/waypoints, получить pedestrian route от Valhalla, увидеть distance/duration и понятную визуализацию потенциального exploration coverage без сохранения `Walk` или персонального progress.
