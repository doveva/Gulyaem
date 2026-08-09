# ГуляЕм

**ГуляЕм** — mobile-first сервис для исследования города через прогулки. Продукт помогает
постепенно открывать улицы и районы, сохранять историю прогулок и выбирать следующий маршрут.

Архитектура проекта строится как city-agnostic, privacy-first modular monolith. Первый город
для разработки и проверки гипотез — Санкт-Петербург.

## Документация

Документация ведётся вместе с кодом. Общие продуктовые и технические материалы находятся в
[`docs/`](docs/README.md), а каждый логический модуль описывается ближайшим `README.md` в своём
дереве каталогов.

Полные правила и локальные команды описаны в
[`docs/documentation-guide.md`](docs/documentation-guide.md).

<!-- docs:index:start -->
_Этот блок сформирован автоматически. Не редактируйте его вручную._

### `backend/`

- [Backend](backend/README.md)

### `data/`

- [Geo data](data/README.md)

### `data/test-areas/spb-dense-center/`

- [Saint Petersburg dense center fixture](data/test-areas/spb-dense-center/README.md)

### `docs/adr/`

- [ADR-0001: Основа импорта OSM для Stage 1](docs/adr/0001-osm-import-foundation.md)
- [ADR-0002: Начальная topology и WalkabilityProfile для StreetSegment](docs/adr/0002-street-segment-topology-and-walkability.md)
- [ADR-0003: Bbox API и базовый Geo Playground для StreetSegment](docs/adr/0003-geo-playground-bbox-api.md)
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

- [ГуляЕм — Development Stages](docs/source_context/gulyaem-development-stages.md)
- [ГуляЕм — Geo & Exploration ADR](docs/source_context/gulyaem-geo-exploration-adr.md)
- [ГуляЕм — контекст проекта](docs/source_context/gulyaem-project-context.md)
- [ГуляЕм — Technical Design](docs/source_context/gulyaem-technical-design.md)
- [Исходный контекст](docs/source_context/README.md)

### `docs/stage 1/`

- [Stage 1 — Acceptance Criteria](docs/stage%201/acceptance-criteria.md)
- [Stage 1 — Architecture Contract](docs/stage%201/architecture-contract.md)
- [Stage 1 — Recommended Implementation Plan](docs/stage%201/implementation-plan.md)
- [Stage 1 — Open Decisions](docs/stage%201/open-decisions.md)
- [Stage 1 — Geo Exploration Playground](docs/stage%201/README.md)
- [ГуляЕм — Stage 1 Requirements: Geo Exploration Playground](docs/stage%201/stage-1-requirements.md)

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

### `scripts/docs/`

- [Инструменты документации](scripts/docs/README.md)

### `scripts/geo/`

- [Geo fixture tools](scripts/geo/README.md)
<!-- docs:index:end -->

## Текущий статус

Реализован фундамент Stage 1.1 — Geo Exploration Playground: Go API, React + TypeScript shell,
PostgreSQL/PostGIS, миграции, MapLibre и два режима локального запуска. Актуальные требования и
план реализации собраны в [`docs/stage 1/`](docs/stage%201/README.md).

## Быстрый запуск

Полный стек в Docker Compose:

```bash
cp .env.example .env
docker compose up --build -d
```

После готовности сервисов:

- Geo Playground: `http://localhost:3000/debug/geo`;
- API readiness: `http://localhost:8080/health/ready`.

Проверить состояние можно командой `docker compose ps`, остановить — `docker compose down`.
Инструкции для запуска API и frontend на хосте находятся в
[`docs/deployment/README.md`](docs/deployment/README.md).

## Проверка

После установки зависимостей (`make bootstrap`) запустите:

```bash
make check
```
