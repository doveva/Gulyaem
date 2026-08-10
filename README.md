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

### `docs/adr/`

- [ADR-0001: Основа импорта OSM для Stage 1](docs/adr/0001-osm-import-foundation.md)
- [ADR-0002: Начальная topology и WalkabilityProfile для StreetSegment](docs/adr/0002-street-segment-topology-and-walkability.md)
- [ADR-0003: Bbox API и базовый Geo Playground для StreetSegment](docs/adr/0003-geo-playground-bbox-api.md)
- [ADR-0004: Версионируемые административные районы для Geo Playground](docs/adr/0004-versioned-administrative-districts.md)
- [ADR-0005: Последовательный map matching и радиусное exploration coverage](docs/adr/0005-sample-route-matching-and-radius-coverage.md)
- [ADR-0006: Valhalla как routing engine для Stage 2](docs/adr/0006-routing-engine-valhalla.md)
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

### `infra/routing/`

- [Routing spike infrastructure](infra/routing/README.md)

### `scripts/docs/`

- [Инструменты документации](scripts/docs/README.md)

### `scripts/geo/`

- [Geo fixture tools](scripts/geo/README.md)

### `scripts/routing/`

- [Routing spike scripts](scripts/routing/README.md)
<!-- docs:index:end -->

## Текущий статус

Реализованы Stage 1.1–1.6 Geo Exploration Playground: воспроизводимый OSM import,
`StreetSegment`, районы, sample routes, радиусное coverage и сравнение routing engines. Для Stage 2
выбрана Valhalla; результаты находятся в
[`ADR-0006`](docs/adr/0006-routing-engine-valhalla.md). Финальная проверка и freeze параметров
остаются Stage 1.7.

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

Воспроизвести routing-engine spike и обновить статический comparison overlay:

```bash
make routing-spike
```

Команда использует Compose profile `routing-spike`; порты и endpoint URLs можно переопределить
через `.env`. Холодные graphs находятся в `.routing/` и удаляются только явной командой
`make routing-reset`.

## Проверка

После установки зависимостей (`make bootstrap`) запустите:

```bash
make check
```
