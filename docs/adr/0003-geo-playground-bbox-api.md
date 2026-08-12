# ADR-0003: Bbox API и базовый Geo Playground для StreetSegment

- **Status:** Accepted
- **Date:** 2026-08-09
- **Owners:** команда «ГуляЕм»
- **Related Stage:** Stage 1.4a

## Context

После Stage 1.3 в PostGIS опубликованы собственные version-aware `StreetSegment`. Следующий шаг —
визуально проверить fragmentation, topology boundaries, classification и распределение длин до
добавления district data, sample routes, coverage и routing engine complexity.

## Decision

### Scope

Stage 1.4a реализует:

- current `GeoDataVersion` endpoint;
- viewport/bbox `StreetSegment` GeoJSON endpoint;
- segment detail endpoint;
- Geo Playground со слоями classification, boundaries, length filters, statistics и inspector.

District schema/API/import, sample routes, coverage и routing engines не входят в этот подэтап.
District contract будет согласован после базовой визуальной проверки segment network.

### City selection

- API использует UUID `cityId`.
- Geo Playground получает city UUID через `VITE_CITY_ID`.
- Локальная конфигурация использует стабильный seed UUID Санкт-Петербурга; city-specific branching
  в backend/frontend запрещён.
- Отдельный cities collection endpoint сейчас не добавляется.

### API

Endpoints:

```text
GET /api/v1/cities/{cityId}/geo-version
GET /api/v1/geo/segments?cityId=...&bbox=west,south,east,north
GET /api/v1/geo/segments/{segmentId}
```

Segments query дополнительно принимает:

```text
classification=EXPLORE,ROUTABLE_ONLY
minLength=5
maxLength=500
```

- Bbox использует EPSG:4326 и валидируется как `west < east`, `south < north`.
- Максимальная площадь запроса — `25 km²`.
- Максимальный результат — `10 000` features; repository читает не более `10 001`.
- Превышение limit возвращает `422 feature_limit_exceeded`; silent truncation запрещён.
- Некорректные query values возвращают структурированный `400`.
- GeoJSON `FeatureCollection` содержит foreign member `meta` с version, bbox, returned count и
  статистикой результата.
- Statistics соответствуют применённым bbox и filters: classification counts, total/explorable
  length, min/median/p95/max и diagnostic counts `< 5 m` / `> 500 m`.

Segment detail доступен по UUID и для superseded version, содержит version status и `isCurrent`.
Street остаётся nullable. Classification reason и normalized attributes доступны всегда; source
way/node IDs и source tags возвращаются только при `debug=true` и только вне production.

### Viewport loading

- Default map extent — `spb-dense-center`.
- Ниже zoom `13` segment requests не выполняются; UI просит приблизить карту.
- Fetch выполняется после `moveend` с debounce около `250 ms`.
- Предыдущий запрос отменяется; stale response не меняет source/map state.
- UI различает `loading`, `ready`, `empty`, `zoom-in`, `limit-exceeded` и `error`.

### Visualization

- `EXPLORE` — зелёно-бирюзовый, `ROUTABLE_ONLY` — янтарный, `IGNORE` — приглушённый
  красно-серый.
- Classification layers переключаются независимо.
- Выбранный segment получает контрастную белую обводку.
- Отдельный `Segment boundaries` layer показывает endpoints и помогает оценить fragmentation.
- Base map можно скрыть независимо от overlay.
- Length range применяется через backend filters.
- Statistics показываются для текущего отфильтрованного viewport.
- Inspector расположен справа на desktop и как bottom sheet на mobile; он показывает ID/version,
  classification, reason, length, normalized tags и optional debug source metadata.

## Alternatives considered

- Загружать всю city network: отклонено из-за browser payload и render cost.
- Молча возвращать первые 10 000 features: отклонено, потому что карта и statistics становились бы
  вводящими в заблуждение.
- Сразу добавить vector tiles: отложено до измерений bbox + GeoJSON.
- Добавить district data сейчас: отложено, чтобы сначала проверить базовую topology без нового
  dataset/import workflow.
- Добавить sample routes и coverage: оставлено Stage 1.5.
- Добавить cities collection endpoint: не требуется для single-configured-city playground.

## Consequences

- GeoJSON остаётся заменяемой delivery strategy и измеряется на реальном fixture.
- UI честно сообщает о zoom/area/feature limits вместо частичной визуализации.
- Настройка города остаётся environment-level, а не city-specific business logic.
- District fields в detail остаются вне контракта Stage 1.4a.
- Source debug metadata не становится публичным default API.

## Validation

- API tests покрывают validation, limits, filters, error shape и debug source gating.
- PostGIS integration tests подтверждают bbox/index query и statistics.
- Frontend tests покрывают viewport state, filters, layer toggles и selection helpers.
- На `spb-dense-center` доступны все три classification, boundaries, statistics и inspector.
- Pan/zoom не применяет stale response и не запрашивает segments ниже minimum zoom.
- Payload/rendering измеряются и фиксируются до решения OD-08.

## Links

- [`ADR-0002`](0002-street-segment-topology-and-walkability.md)
- [`Stage 1 requirements`](../stage%201/stage-1-requirements.md)
- [`Stage 1 architecture contract`](../stage%201/architecture-contract.md)
- [`Stage 1 open decisions`](../stage%201/open-decisions.md)

