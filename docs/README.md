# Документация «ГуляЕм»

Это единая точка входа в продуктовую и техническую документацию проекта. Вводная часть и
правила ниже поддерживаются вручную; каталог документов между служебными маркерами формируется
автоматически.

## Как устроена документация

- `product/` описывает продуктовые границы и устойчивые понятия.
- `architecture/` содержит актуальную архитектуру системы.
- `adr/` фиксирует значимые архитектурные решения и их последствия.
- `deployment/` описывает локальный запуск, окружения, доставку и эксплуатацию.
- `technical-debt/` содержит единый реестр технического долга.
- `stages/` задаёт правила и шаблон документации этапов; материалы конкретного этапа лежат в
  `stage N/`.
- `source_context/` хранит исходные документы, из которых формируется актуальная документация.

## Правило актуальности

Изменение поведения логического модуля должно сопровождаться изменением ближайшего
`README.md`, который описывает этот модуль. Тесты, сгенерированные файлы и артефакты сборки
исключены из автоматической проверки. Подробности — в
[`documentation-guide.md`](documentation-guide.md).

## Общий индекс

<!-- docs:index:start -->
_Этот блок сформирован автоматически. Не редактируйте его вручную._

### `backend/`

- [Backend](../backend/README.md)

### `data/districts/spb-administrative-districts/`

- [Административные районы Санкт-Петербурга](../data/districts/spb-administrative-districts/README.md)

### `data/`

- [Geo data](../data/README.md)

### `data/routing-spike/spb-stage1/`

- [Stage 1.6 routing fixtures](../data/routing-spike/spb-stage1/README.md)

### `data/sample-routes/spb-stage1/`

- [Saint Petersburg Stage 1 sample routes](../data/sample-routes/spb-stage1/README.md)

### `data/test-areas/spb-dense-center/`

- [Saint Petersburg dense center fixture](../data/test-areas/spb-dense-center/README.md)

### `data/test-areas/spb-stage1-validation/`

- [Saint Petersburg Stage 1 validation fixture](../data/test-areas/spb-stage1-validation/README.md)

### `data/validation/spb-stage1/`

- [Saint Petersburg Stage 1 validation report](../data/validation/spb-stage1/README.md)

### `docs/adr/`

- [ADR-0001: Основа импорта OSM для Stage 1](adr/0001-osm-import-foundation.md)
- [ADR-0002: Начальная topology и WalkabilityProfile для StreetSegment](adr/0002-street-segment-topology-and-walkability.md)
- [ADR-0003: Bbox API и базовый Geo Playground для StreetSegment](adr/0003-geo-playground-bbox-api.md)
- [ADR-0004: Версионируемые административные районы для Geo Playground](adr/0004-versioned-administrative-districts.md)
- [ADR-0005: Последовательный map matching и радиусное exploration coverage](adr/0005-sample-route-matching-and-radius-coverage.md)
- [ADR-0006: Valhalla как routing engine для Stage 2](adr/0006-routing-engine-valhalla.md)
- [ADR-0007: Freeze topology и WalkabilityProfile после Stage 1](adr/0007-street-segment-stage1-freeze.md)
- [ADR-0008: Начальные exploration coverage параметры для Stage 2](adr/0008-coverage-parameters-stage1-freeze.md)
- [ADR-0009: Bbox + GeoJSON как начальная map delivery Stage 2](adr/0009-bbox-geojson-stage2.md)
- [ADR-NNNN: Краткое название решения](adr/adr-template.md)
- [Architecture Decision Records](adr/README.md)

### `docs/architecture/`

- [Архитектура](architecture/README.md)

### `docs/deployment/`

- [Деплой и эксплуатация](deployment/README.md)

### `docs/`

- [Правила документации](documentation-guide.md)

### `docs/product/`

- [Продукт](product/README.md)

### `docs/source_context/`

- [ГуляЕм — Geo & Exploration ADR](source_context/gulyaem-geo-exploration-adr.md)
- [ГуляЕм — контекст проекта](source_context/gulyaem-project-context.md)
- [Исходный контекст](source_context/README.md)

### `docs/stage 1/`

- [Stage 1 — Acceptance Criteria](stage%201/acceptance-criteria.md)
- [Stage 1 — Architecture Contract](stage%201/architecture-contract.md)
- [Stage 1 — Recommended Implementation Plan](stage%201/implementation-plan.md)
- [Stage 1 — Open Decisions](stage%201/open-decisions.md)
- [Stage 1 — Geo Exploration Playground](stage%201/README.md)
- [ГуляЕм — Stage 1 Requirements: Geo Exploration Playground](stage%201/stage-1-requirements.md)
- [Stage 1.7 — Validation Report](stage%201/validation-report.md)

### `docs/stage 2/`

- [Stage 2 — Acceptance Criteria](stage%202/acceptance-criteria.md)
- [Stage 2 — API Contract](stage%202/api-contract.md)
- [Stage 2 — Architecture Contract](stage%202/architecture-contract.md)
- [Stage 2 — Frontend Route Builder Flow](stage%202/frontend-flow.md)
- [Stage 2 — Recommended Implementation Plan](stage%202/implementation-plan.md)
- [Stage 2 — Decision Log and Deferred Follow-ups](stage%202/open-decisions.md)
- [ГуляЕм — Stage 2 Requirements: Manual Route & Exploration Preview](stage%202/stage-2-requirements.md)
- [Stage 2 — Validation Plan](stage%202/validation-plan.md)
- [Stage 2 — Validation Report](stage%202/validation-report.md)

### `docs/stages/`

- [Этапы разработки](stages/README.md)
- [Stage N — Название этапа](stages/stage-template.md)

### `docs/technical-debt/`

- [Технический долг](technical-debt/README.md)

### `frontend/`

- [Frontend](../frontend/README.md)

### `infra/osmium/`

- [Osmium fixture tool](../infra/osmium/README.md)

### `infra/postgis/`

- [PostGIS image](../infra/postgis/README.md)

### `infra/routing/`

- [Routing spike infrastructure](../infra/routing/README.md)

### `infra/routing/valhalla/`

- [Valhalla development runtime](../infra/routing/valhalla/README.md)

### `scripts/docs/`

- [Инструменты документации](../scripts/docs/README.md)

### `scripts/geo/`

- [Geo fixture tools](../scripts/geo/README.md)

### `scripts/routing/`

- [Routing spike scripts](../scripts/routing/README.md)

### `scripts/validation/`

- [Stage 1 validation tooling](../scripts/validation/README.md)
<!-- docs:index:end -->
