# ADR-0002: Начальная topology и WalkabilityProfile для StreetSegment

- **Status:** Accepted
- **Date:** 2026-08-09
- **Owners:** команда «ГуляЕм»
- **Related Stage:** Stage 1.3

## Context

Stage 1.3 должен построить собственные `StreetSegment` из локального OSM PBF, не превращая OSM
ways в доменную identity и не сохраняя raw OSM nodes/ways/relations в PostgreSQL. Результат нужен
для инженерной визуальной проверки: правила topology и walkability ещё могут быть уточнены по
результатам Geo Playground, но начальная реализация должна быть однозначной и воспроизводимой.

## Decision

### Processing boundary

- PBF обрабатывается во временных Go-структурах в памяти. PostgreSQL не получает raw OSM tables.
- На время импорта допустимы source node/way/relation IDs; после импорта они остаются только в
  debug/provenance metadata опубликованных сущностей.
- Такой processing mode используется для Stage 1 fixtures. Переход к disk-backed или иному
  pipeline возможен только после измерения памяти и времени на representative extract.

### Topology model

- Базовый graph строится из последовательных пар nodes линейных candidate ways.
- `StreetSegment` является ненаправленной доменной сущностью. Будущие ограничения прохода
  forward/backward принадлежат routing graph и не создают две копии segment.
- Геометрическое пересечение линий без общего OSM node не создаёт connection или split: это может
  быть мост, тоннель или разные уровни.
- Общий OSM node является topology connection; классификация определяет возможность дальнейшего
  traversal, но не отменяет сам факт intersection для segmentation.

Segment разрывается в:

- начале или конце связного пути;
- node с topology degree, отличным от `2`;
- node с `barrier=*`, entrance/gate или изменением access;
- точке изменения classification либо pedestrian accessibility;
- точке изменения `highway`, `footway`, `surface`, `bridge`, `tunnel`, `indoor` или `level`;
- границе имени улицы;
- точке пересечения границы import bbox;
- искусственной точке `max_segment_length_m`, только когда экспериментальная настройка включена.

Соседние OSM ways объединяются через degree-2 node, когда нормализованные semantics совпадают.
Один OSM Way не является одним `StreetSegment` по умолчанию.

Замкнутый connected loop без значимых topology nodes сохраняется как один замкнутый положительный
`LineString`. Произвольный split только ради выбора начала loop не добавляется.

### WalkabilityProfile v1

Правила применяются в порядке: pedestrian access overrides, специальные представления, затем
базовый highway class. Итог всегда содержит `classification`, стабильный `reason_code` и
релевантные source tags.

Access rules:

- `foot=no/private/use_sidepath` даёт `IGNORE`;
- общий `access=no/private/customers` или иной restricted access даёт `IGNORE`, если более
  специфичный `foot=*` не разрешает проход;
- `foot=yes/designated/permissive` разрешает проход и перекрывает общий access;
- `foot=discouraged` и неразобранные conditional restrictions дают `ROUTABLE_ONLY`;
- `access=permissive` считается доступным, но его отзываемость сохраняется в attributes.

Начальный `EXPLORE`:

- `highway=pedestrian/living_street/residential/unclassified`;
- `highway=footway`, кроме специальных connector types;
- `highway=path/track`, когда walking access не запрещён;
- `highway=steps`;
- `highway=cycleway/bridleway` только с явным `foot=yes/designated/permissive`;
- `highway=primary/secondary/tertiary`, когда walking access не запрещён, sidewalk не запрещён и
  не вынесен в отдельную geometry;
- публичный `highway=service + service=alley`;
- публичный `highway=service + service=track`, даже несмотря на нестандартность subtype;
- `highway=service` с `foot=designated`.

Начальный `ROUTABLE_ONLY`:

- `footway=crossing/traffic_island/link`;
- road `*_link` classes, если pedestrian access не запрещён;
- `highway=service` без отдельного explorable override;
- `service=driveway/parking_aisle/emergency_access/drive-through`;
- roadway с `sidewalk=separate` или `sidewalk:left/right/both=separate`; отдельно нанесённый
  `highway=footway + footway=sidewalk` остаётся `EXPLORE`;
- `primary/secondary/tertiary` с `sidewalk=no`;
- явно распознанный короткий технический connector;
- разрешённые, но discouraged или условно доступные пути.

Начальный `IGNORE` либо исключение из linear graph:

- private/prohibited pedestrian access;
- `motorway`, `trunk`, их links и `raceway`;
- `construction`, `proposed`, `abandoned`;
- `highway=corridor` и `indoor=yes` на Stage 1;
- `cycleway/bridleway` без разрешения для пешеходов;
- `area=yes` и pedestrian areas: они не превращаются в fake boundary segment, а учитываются в
  report как `unsupported_pedestrian_area`.

Простой `oneway=yes` на обычной улице не меняет ненаправленный segment. `oneway:foot=*`,
`foot:forward=*` и `foot:backward=*` сохраняются в normalized attributes для будущего routing.

### Storage and publication

- `street_segments` содержит UUID, `city_id`, `geo_data_version_id`, nullable `street_id`,
  `geometry(LineString, 4326)`, положительный `length_m`, classification и `attributes jsonb`.
- Geometry получает GiST index; version и classification получают обычные индексы.
- Source way IDs и граничные node IDs допустимы только в debug/provenance attributes и не являются
  domain identity.
- Таблица `streets` создаётся как часть схемы, но автоматическая name normalization пока не
  выполняется; `street_segments.street_id` остаётся nullable.
- Все segments новой версии записываются и валидируются в publish transaction. Только после этого
  новая `GeoDataVersion` становится `READY`, а предыдущая — `SUPERSEDED`.
- Ошибка publication откатывает новые segments. Попытка становится `FAILED` и не меняет current
  `READY` version.

### Import boundary and geometry anomalies

- Для fixture authoritative boundary берётся из manifest bbox. Geometry обрезается по bbox;
  созданные на границе endpoints получают `boundary_clip=true`.
- Для явного PBF используется bbox из PBF header. Если header bbox отсутствует, import не обрезает
  geometry и добавляет warning в report.
- Отсутствующая координата referenced node или разорванная node sequence делает импорт `FAILED`.
- Candidate с менее чем двумя различными точками, некорректными координатами или нулевой длиной
  не публикуется и учитывается в report.
- `READY` version может содержать только валидные `LineString` с `length_m > 0`.

Exact duplicate geometry с одинаковыми normalized semantics объединяется, source references
объединяются. Одинаковая geometry с конфликтующими semantics/classification не разрешается
автоматически: anomaly и обе source interpretations остаются inspectable, а report содержит
`conflicting_duplicate_geometry`.

### Length experiments and report

- `max_segment_length_m=0` по умолчанию: artificial length splitting отключён.
- Положительное значение включает экспериментальный split слишком длинной geometry; placement
  проверяется отдельными synthetic tests и не становится основной topology strategy.
- Диагностический `short` threshold — менее `5 m`; `long` threshold — более `500 m`. Эти пороги не
  меняют topology.
- Report содержит минимум counts по classification, generated/rejected/clipped/deduplicated
  segments, invalid and duplicate anomalies, total/explorable length, min/median/p95/max length,
  short/long counts и import duration.

### Required synthetic tests

До tuning на реальном PBF фиксируются тестами:

- simple line, T-junction и intersection;
- geometry crossing без общего node;
- merge двух OSM ways через degree-2 node;
- split при изменении semantics, access и barrier/gate;
- closed loop и bbox clipping;
- missing node и degenerate geometry;
- exact и conflicting duplicate geometry;
- включённый и отключённый `max_segment_length_m`;
- сохранение directional metadata без дублирования segment.

## Alternatives considered

- Сохранять raw OSM graph в PostgreSQL: отклонено согласованной границей Stage 1.2.
- Один OSM Way превращать в один segment: отклонено, потому что OSM split не равен product
  topology.
- Делить всё каждые N метров: отклонено как основная стратегия; оставлено только как отключённый
  эксперимент.
- Делать направленные копии каждого segment: отклонено; direction относится к traversal.
- Считать все `highway=service` explorable: отклонено из-за driveway, parking и технических
  подъездов; публичные alley/service-track остаются явными исключениями.
- Автоматически превращать pedestrian area boundary в street: отклонено как ложная linear
  geometry.
- Молча выбирать одну classification при конфликтующем duplicate: отклонено до визуальной
  проверки evidence.

## Consequences

- Initial profile централизован и inspectable, но намеренно консервативен для service, indoor и
  area semantics.
- Отсутствие raw OSM tables сохраняет чистую persistence boundary, но ограничивает размер import
  доступной памятью процесса.
- Ненаправленный segment остаётся стабильной единицей exploration; routing сможет добавить
  directed traversal отдельно.
- Bbox clipping создаёт явные artificial endpoints только на границе fixture.
- Параметры profile, duplicate handling и maximum length должны быть проверены в Geo Playground до
  Stage 1 freeze; изменение evidence оформляется новым ADR или superseding decision.

## Validation

- synthetic graph tests дают ожидаемые boundaries и classifications;
- повторный import того же fixture остаётся идемпотентным;
- все published segments имеют валидную geometry, положительную длину и текущую version;
- import report объясняет rejects, duplicates, clipping и classification distribution;
- визуальная проверка dense center показывает intersections, service paths, sidewalks, crossings,
  stairs и boundary clips;
- memory/time измеряются до перехода к более крупному extract.

## Links

- [`ADR-0001`](0001-osm-import-foundation.md)
- [`Stage 1 requirements`](../stage%201/stage-1-requirements.md)
- [`Stage 1 architecture contract`](../stage%201/architecture-contract.md)
- [`Stage 1 open decisions`](../stage%201/open-decisions.md)
- [OSM `service=*`](https://wiki.openstreetmap.org/wiki/Key:service)
- [OSM access tags](https://wiki.openstreetmap.org/wiki/Access_tags)
