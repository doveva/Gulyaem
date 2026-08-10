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

### `data/test-areas/spb-dense-center/`

- [Saint Petersburg dense center fixture](../data/test-areas/spb-dense-center/README.md)

### `docs/adr/`

- [ADR-0001: Основа импорта OSM для Stage 1](adr/0001-osm-import-foundation.md)
- [ADR-0002: Начальная topology и WalkabilityProfile для StreetSegment](adr/0002-street-segment-topology-and-walkability.md)
- [ADR-0003: Bbox API и базовый Geo Playground для StreetSegment](adr/0003-geo-playground-bbox-api.md)
- [ADR-0004: Версионируемые административные районы для Geo Playground](adr/0004-versioned-administrative-districts.md)
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

- [ГуляЕм — Development Stages](source_context/gulyaem-development-stages.md)
- [ГуляЕм — Geo & Exploration ADR](source_context/gulyaem-geo-exploration-adr.md)
- [ГуляЕм — контекст проекта](source_context/gulyaem-project-context.md)
- [ГуляЕм — Technical Design](source_context/gulyaem-technical-design.md)
- [Исходный контекст](source_context/README.md)

### `docs/stage 1/`

- [Stage 1 — Acceptance Criteria](stage%201/acceptance-criteria.md)
- [Stage 1 — Architecture Contract](stage%201/architecture-contract.md)
- [Stage 1 — Recommended Implementation Plan](stage%201/implementation-plan.md)
- [Stage 1 — Open Decisions](stage%201/open-decisions.md)
- [Stage 1 — Geo Exploration Playground](stage%201/README.md)
- [ГуляЕм — Stage 1 Requirements: Geo Exploration Playground](stage%201/stage-1-requirements.md)

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

### `scripts/docs/`

- [Инструменты документации](../scripts/docs/README.md)

### `scripts/geo/`

- [Geo fixture tools](../scripts/geo/README.md)
<!-- docs:index:end -->
