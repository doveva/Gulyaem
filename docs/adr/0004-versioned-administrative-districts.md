# ADR-0004: Версионируемые административные районы для Geo Playground

- **Status:** Accepted
- **Date:** 2026-08-09
- **Owners:** команда «ГуляЕм»
- **Related Stage:** Stage 1.4b

## Context

Базовая визуализация `StreetSegment` из Stage 1.4a проверена на `spb-dense-center`. Для завершения
Stage 1.4 нужен районный слой, но локальный PBF вырезан по плотному центру и не содержит полные
границы всех районов. Районы также обновляются независимо от street graph и не должны менять
identity или topology сегментов.

## Decision

### Source and fixture

- Используются 18 официальных административных районов Санкт-Петербурга: OSM
  `boundary=administrative`, `admin_level=5`, `addr:region=Санкт-Петербург`.
- Муниципальные образования уровня 8 не входят в набор.
- Полные Polygon/MultiPolygon geometry хранятся в committed GeoJSON fixture вместе с OSM relation
  IDs, OSM base timestamp, provenance, ODbL attribution, размером и SHA-256.
- Список relations проверяется через Overpass, geometry получена через официальный Nominatim lookup.
- Runtime network access отсутствует; обновление fixture выполняется отдельно и осознанно.

### Versioning and persistence

- `DistrictDataVersion` имеет независимый от `GeoDataVersion` lifecycle:
  `IMPORTING → READY | FAILED`, предыдущий `READY → SUPERSEDED`.
- Для города допускается по одной `IMPORTING` и одной `READY` district version.
- Identity идемпотентного импорта: `(city_id, source_checksum, normalization_version)`.
- Публикация всех `District`, смена current version и обновление `City.boundary` выполняются одной
  транзакцией.
- PostgreSQL хранит нормализованные District и version/import metadata; исходный GeoJSON остаётся
  fixture.
- Полная geometry хранится в PostGIS. API использует `ST_SimplifyPreserveTopology` только для
  display geometry. Label point вычисляется через `ST_PointOnSurface`.

### Segment relation

- `district_id` не добавляется в `StreetSegment`.
- Сегменты не разрезаются на административных границах.
- Detail сегмента возвращает `districts[]`, вычисленный пространственным пересечением с текущей
  `READY` district version. Сегмент на границе может относиться к нескольким районам.

### API and visualization

```text
GET /api/v1/geo/districts?cityId=...&bbox=west,south,east,north
```

- Bbox использует общий лимит `25 km²`.
- Ответ — GeoJSON FeatureCollection с district version/source metadata.
- Geo Playground показывает полупрозрачную заливку, границы, подписи, независимый toggle и
  минимальный inspector: name, kind, source и normalization version.
- District progress, coverage и statistics не входят в Stage 1.4.

## Alternatives considered

- Извлекать районы из `spb-dense-center.osm.pbf`: отклонено из-за неполных boundary relations.
- Загружать районы из сети при старте: отклонено из-за невоспроизводимости и runtime dependency.
- Связать district version со street `GeoDataVersion`: отклонено из-за разных источников и ритма
  обновления.
- Хранить один `district_id` в сегменте: отклонено, потому что пересечение на границе не является
  однозначным.
- Разрезать сегменты по границам районов: отклонено, потому что административная граница не должна
  менять street topology.

## Consequences

- Районный слой можно обновлять независимо и воспроизводимо.
- Полная geometry остаётся доступной для точных spatial queries, а UI получает меньший display
  payload.
- Spatial association вычисляется при чтении; при росте нагрузки её можно материализовать без
  изменения domain identity сегмента.
- Следующий отдельный scope — Stage 1.5 sample routes и coverage.

## Validation

- Fixture содержит ровно 18 уникальных relations и проверяется по checksum.
- Integration tests проверяют lifecycle, idempotency, atomic publication, valid geometry,
  `City.boundary`, bbox query и segment-to-district relation.
- Compose import повторно возвращает тот же `READY` version.
- Geo Playground проверяется на desktop и mobile viewport.

## Links

- [`ADR-0003`](0003-geo-playground-bbox-api.md)
- [`District fixture`](../../data/districts/spb-administrative-districts/README.md)
- [`Stage 1 requirements`](../stage%201/stage-1-requirements.md)

