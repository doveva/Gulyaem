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
- `docs/stage 2/stage-2-requirements.md` — полный Stage 2 scope.
- `docs/stage 2/architecture-contract.md` — фиксированные архитектурные решения.
- `docs/stage 2/api-contract.md` — целевой HTTP contract.
- `docs/stage 2/frontend-flow.md` — route-builder UX/state model.
- `docs/stage 2/implementation-plan.md` — рекомендуемый порядок реализации.
- `docs/stage 2/acceptance-criteria.md` — Definition of Done.
- `docs/stage 2/open-decisions.md` — принятые решения и явно deferred follow-ups.
- `docs/stage 2/validation-plan.md` — automated + manual validation.
- `docs/stage 2/validation-report.md` — результаты автоматизированной и runtime-проверки.

## Stage 2 Definition of Done

Stage 2 завершён, когда пользователь может на `/map` интерактивно задать start/destination/waypoints, получить pedestrian route от Valhalla, увидеть distance/duration и понятную визуализацию потенциального exploration coverage без сохранения `Walk` или персонального progress.

## Индекс документации

<!-- docs:index:start -->
_Этот блок сформирован автоматически. Не редактируйте его вручную._

### `backend/`

- [Backend](backend/README.md)

### `data/districts/spb-administrative-districts/`

- [Административные районы Санкт-Петербурга](data/districts/spb-administrative-districts/README.md)

### `data/`

- [Geo data](data/README.md)

### `data/routing-spike/spb-stage1/`

- [Stage 1.6 routing fixtures](data/routing-spike/spb-stage1/README.md)

### `data/sample-routes/spb-stage1/`

- [Saint Petersburg Stage 1 sample routes](data/sample-routes/spb-stage1/README.md)

### `data/test-areas/spb-dense-center/`

- [Saint Petersburg dense center fixture](data/test-areas/spb-dense-center/README.md)

### `data/test-areas/spb-stage1-validation/`

- [Saint Petersburg Stage 1 validation fixture](data/test-areas/spb-stage1-validation/README.md)

### `data/validation/spb-stage1/`

- [Saint Petersburg Stage 1 validation report](data/validation/spb-stage1/README.md)

### `docs/adr/`

- [ADR-0001: Основа импорта OSM для Stage 1](docs/adr/0001-osm-import-foundation.md)
- [ADR-0002: Начальная topology и WalkabilityProfile для StreetSegment](docs/adr/0002-street-segment-topology-and-walkability.md)
- [ADR-0003: Bbox API и базовый Geo Playground для StreetSegment](docs/adr/0003-geo-playground-bbox-api.md)
- [ADR-0004: Версионируемые административные районы для Geo Playground](docs/adr/0004-versioned-administrative-districts.md)
- [ADR-0005: Последовательный map matching и радиусное exploration coverage](docs/adr/0005-sample-route-matching-and-radius-coverage.md)
- [ADR-0006: Valhalla как routing engine для Stage 2](docs/adr/0006-routing-engine-valhalla.md)
- [ADR-0007: Freeze topology и WalkabilityProfile после Stage 1](docs/adr/0007-street-segment-stage1-freeze.md)
- [ADR-0008: Начальные exploration coverage параметры для Stage 2](docs/adr/0008-coverage-parameters-stage1-freeze.md)
- [ADR-0009: Bbox + GeoJSON как начальная map delivery Stage 2](docs/adr/0009-bbox-geojson-stage2.md)
- [ADR-NNNN: Краткое название решения](docs/adr/adr-template.md)
- [Architecture Decision Records](docs/adr/README.md)

### `docs/architecture/`

- [Архитектура](docs/architecture/README.md)

### `docs/deployment/`

- [Деплой и эксплуатация](docs/deployment/README.md)

### `docs/`

- [Правила документации](docs/documentation-guide.md)

### `docs/product/`

- [Продукт](docs/product/README.md)

### `docs/source_context/`

- [ГуляЕм — Geo & Exploration ADR](docs/source_context/gulyaem-geo-exploration-adr.md)
- [ГуляЕм — контекст проекта](docs/source_context/gulyaem-project-context.md)
- [Исходный контекст](docs/source_context/README.md)

### `docs/stage 1/`

- [Stage 1 — Acceptance Criteria](docs/stage%201/acceptance-criteria.md)
- [Stage 1 — Architecture Contract](docs/stage%201/architecture-contract.md)
- [Stage 1 — Recommended Implementation Plan](docs/stage%201/implementation-plan.md)
- [Stage 1 — Open Decisions](docs/stage%201/open-decisions.md)
- [Stage 1 — Geo Exploration Playground](docs/stage%201/README.md)
- [ГуляЕм — Stage 1 Requirements: Geo Exploration Playground](docs/stage%201/stage-1-requirements.md)
- [Stage 1.7 — Validation Report](docs/stage%201/validation-report.md)

### `docs/stage 2/`

- [Stage 2 — Acceptance Criteria](docs/stage%202/acceptance-criteria.md)
- [Stage 2 — API Contract](docs/stage%202/api-contract.md)
- [Stage 2 — Architecture Contract](docs/stage%202/architecture-contract.md)
- [Stage 2 — Frontend Route Builder Flow](docs/stage%202/frontend-flow.md)
- [Stage 2 — Recommended Implementation Plan](docs/stage%202/implementation-plan.md)
- [Stage 2 — Decision Log and Deferred Follow-ups](docs/stage%202/open-decisions.md)
- [ГуляЕм — Stage 2 Requirements: Manual Route & Exploration Preview](docs/stage%202/stage-2-requirements.md)
- [Stage 2 — Validation Plan](docs/stage%202/validation-plan.md)
- [Stage 2 — Validation Report](docs/stage%202/validation-report.md)

### `docs/stages/`

- [Этапы разработки](docs/stages/README.md)
- [Stage N — Название этапа](docs/stages/stage-template.md)

### `docs/technical-debt/`

- [Технический долг](docs/technical-debt/README.md)

### `frontend/`

- [Frontend](frontend/README.md)

### `infra/osmium/`

- [Osmium fixture tool](infra/osmium/README.md)

### `infra/postgis/`

- [PostGIS image](infra/postgis/README.md)

### `infra/routing/`

- [Routing spike infrastructure](infra/routing/README.md)

### `infra/routing/valhalla/`

- [Valhalla development runtime](infra/routing/valhalla/README.md)

### `scripts/docs/`

- [Инструменты документации](scripts/docs/README.md)

### `scripts/geo/`

- [Geo fixture tools](scripts/geo/README.md)

### `scripts/routing/`

- [Routing spike scripts](scripts/routing/README.md)

### `scripts/validation/`

- [Stage 1 validation tooling](scripts/validation/README.md)
<!-- docs:index:end -->
