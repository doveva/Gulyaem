# Stage 1 — Geo Exploration Playground

Цель этапа — проверить фундаментальную geo-модель «ГуляЕм» на реальных данных до реализации
пользовательского exploration loop.

## Документы этапа

| Документ | Назначение |
|---|---|
| [`stage-1-requirements.md`](stage-1-requirements.md) | Полные требования, scope и Definition of Done |
| [`architecture-contract.md`](architecture-contract.md) | Зафиксированные архитектурные границы этапа |
| [`implementation-plan.md`](implementation-plan.md) | Рекомендуемая последовательность реализации |
| [`acceptance-criteria.md`](acceptance-criteria.md) | Проверяемые критерии завершения |
| [`open-decisions.md`](open-decisions.md) | Решения, которые должен закрыть Stage 1 |

Принятые решения Stage 1.2 зафиксированы в
[`ADR-0001`](../adr/0001-osm-import-foundation.md).

## Ожидаемый результат

Воспроизводимый импорт OSM-данных, внутренняя модель `StreetSegment`, классификация
walkability, bbox API и web playground для визуальной проверки сегментации и coverage.

После завершения этапа этот README дополняется фактическими результатами, ссылками на ADR,
отклонениями от плана и зарегистрированным техническим долгом.

## Прогресс реализации

### Stage 1.1 — Foundation bootstrap

Реализовано:

- отдельные Go API и React + TypeScript frontend;
- PostgreSQL/PostGIS и SQL-миграции через `golang-migrate`;
- MapLibre Geo Playground на `/debug/geo` с публичным заменяемым стилем;
- readiness/liveness endpoints и проверка связи frontend → backend → PostGIS;
- локальный host-mode и полный Docker Compose build/run;
- базовые автоматические проверки и документация запуска.

Открытые geo-решения OD-01–OD-08 этим bootstrap не фиксируются. OD-09 пока имеет заменяемый
локальный default OpenFreeMap Liberty без API-ключа; окончательное подтверждение относится к
визуальной проверке Stage 1.

### Stage 1.2 — OSM import foundation

Реализовано:

- `cmd/geo-import` и source adapter для streaming OSM PBF;
- committed `spb-dense-center` fixture с provenance и SHA-256;
- `City` и полный lifecycle `GeoDataVersion`;
- идемпотентность по city/checksum/normalization version;
- атомарные `READY`/`SUPERSEDED` и сохраняемый `FAILED` report;
- structured import summary и unit/PBF/PostGIS integration tests.

Согласованные границы и критерии пересмотра parser зафиксированы в
[`ADR-0001`](../adr/0001-osm-import-foundation.md). Raw OSM entities остаются в PBF;
`StreetSegment` начинается на Stage 1.3.
