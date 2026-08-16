# ГуляЕм — Stage 3 Exploration Core

Текущий scope — **Stage 3: persistent Walk lifecycle и персональная карта исследования**.

Stage 1/2 считаются завершёнными и дают Stage 3 следующие входные решения:

- internal `StreetSegment` graph;
- `WalkabilityProfile v1`;
- topology-first segmentation;
- `max_segment_length_m=0`;
- `StreetSegment` identity scoped by `GeoDataVersion`;
- grade-aware route matching / coverage;
- default coverage profile **Balanced: 100 м / 0.4 / 15–80 м**;
- `PARTIAL` сохраняется;
- routing engine: **Valhalla**;
- map delivery: **bbox + GeoJSON** на раннем этапе;
- React + TypeScript + MapLibre frontend;
- Go + PostgreSQL/PostGIS backend.

## Главная цель Stage 3

Получить первую пользовательскую capability поверх реального geo core:

```text
Map
 ↓
Manual Route Preview
 ↓
Materialize Route + Walk
 ↓
Active → Review → Complete
 ↓
ExplorationDelta + Personal Progress
 ↓
Walk Summary + Persistent Explored Map
```

Главный validation question:

> Является ли `Walk completion → visual city reveal` понятным, мотивирующим и технически воспроизводимым core loop?

## Важная граница Stage 3

Stage 2 preview остаётся stateless, а Stage 3 материализует только повторно проверенный сервером
результат. Browser не передаёт authoritative geometry, segment IDs, progress или actor ID.

Completion сравнивает trusted coverage snapshot с actor-scoped progress и фиксирует `NEW` /
`REVISITED`. Только `COMPLETED EXPLORE` влияет на карту; `PARTIAL`, `ROUTABLE_ONLY` и `IGNORE`
персональный progress не создают.

## Runtime API

API использует `askcel-go` для Runtime Contract v1, OpenTelemetry и health checks. `make api` и
Compose материализуют локальные `ASKCEL_*` значения; управляемое окружение обязано передать их
само. Без `OTEL_EXPORTER_OTLP_ENDPOINT` instrumentation остаётся включённым, но экспорт не
создаётся. Во время graceful shutdown readiness переходит в `draining`, а liveness остаётся
доступным до остановки процесса. Для container build приватной зависимости `askcel-go` Compose
пробрасывает BuildKit SSH agent; в agent должен быть загружен GitHub deploy/user key с read access.

## Основные документы

- `AGENTS.md` — scope guard для coding agent.
- `docs/stage 3/stage-3-requirements.md` — полный Stage 3 scope.
- `docs/stage 3/architecture-contract.md` — фиксированные границы модулей и consistency.
- `docs/stage 3/domain-model.md` и `persistence-model.md` — domain/schema semantics.
- `docs/stage 3/api-contract.md` и `frontend-flow.md` — HTTP и `/map` flow.
- `docs/stage 3/acceptance-criteria.md` — Definition of Done.
- `docs/stage 3/validation-report.md` — фактические результаты и pending field validation.

## Stage 3 Definition of Done

Stage 3 завершён после полного preview → Walk → Review → Complete loop, идемпотентного progress,
сохранения overlay после reload и эквивалентного rebuild из COMPLETED Walk geometry.

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

### `data/validation/spb-stage3-coverage-v2/`

- [Stage 3 coverage-v2 validation evidence](data/validation/spb-stage3-coverage-v2/README.md)

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
- [ADR-0010: Server-side materialization of Stage 2 route preview](docs/adr/0010-stage3-route-materialization.md)
- [ADR-0011: Atomic and idempotent Walk completion](docs/adr/0011-stage3-walk-completion-transaction.md)
- [ADR-0012: Rebuildable actor exploration read model](docs/adr/0012-stage3-exploration-read-model.md)
- [ADR-0013: Actor-scoped Stage 3 data before authentication](docs/adr/0013-stage3-development-actor-context.md)
- [ADR-0014: Stage 3 coverage radius retuning](docs/adr/0014-stage3-coverage-radius-retuning.md)
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

### `docs/stage 3/`

- [Stage 3 — Acceptance Criteria](docs/stage%203/acceptance-criteria.md)
- [Stage 3 — API Contract](docs/stage%203/api-contract.md)
- [Stage 3 — Architecture Contract](docs/stage%203/architecture-contract.md)
- [Stage 3 — Domain Model](docs/stage%203/domain-model.md)
- [Stage 3 — Frontend Flow](docs/stage%203/frontend-flow.md)
- [Stage 3 — Recommended Implementation Plan](docs/stage%203/implementation-plan.md)
- [Stage 3 documentation integration](docs/stage%203/INTEGRATION.md)
- [Stage 3 — Open Decisions](docs/stage%203/open-decisions.md)
- [Stage 3 — Persistence Model](docs/stage%203/persistence-model.md)
- [Stage 3 — Exploration Core](docs/stage%203/README.md)
- [ГуляЕм — Stage 3 Requirements: Exploration Core](docs/stage%203/stage-3-requirements.md)
- [Stage 3 — Validation Plan](docs/stage%203/validation-plan.md)
- [Stage 3 — Validation Report](docs/stage%203/validation-report.md)

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
