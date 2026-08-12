# ГуляЕм — Geo & Exploration ADR

**Status:** Accepted for MVP direction / pending geo-spike validation  
**Scope:** geo source, StreetSegment model, exploration semantics, routing engine boundaries  
**Related Technical Design:** `gulyaem-technical-design.md`

---

# ADR-004 — Geo Data Source

## Context

«ГуляЕм» должен быть city-agnostic, но первый город для тестирования — Санкт-Петербург.

Продукту необходимы:

- уличная сеть;
- pedestrian paths;
- административные границы;
- POI в дальнейшем;
- возможность self-hosted routing;
- возможность использовать одни и те же исходные данные для manual routing, map matching и exploration.

При этом нельзя делать внешнюю geo-модель частью доменного контракта приложения.

---

## Decision

Для MVP основным источником геоданных является **OpenStreetMap**.

OSM используется только как upstream source.

Pipeline:

```text
OpenStreetMap
    ↓
Geo Import
    ↓
Normalization
    ↓
GeoDataVersion
    ↓
Internal Street / StreetSegment model
```

Raw OSM identifiers не являются persistent domain identity.

Нельзя строить пользовательский прогресс напрямую на:

```text
osm_way_id
osm_node_id
routing_engine_edge_id
```

---

## Consequences

Плюсы:

- city-agnostic;
- open-data ecosystem;
- подходит для self-hosting;
- можно использовать один geo source для routing и exploration;
- нет зависимости от коммерческого map provider.

Минусы:

- необходимо поддерживать собственный import pipeline;
- нужно учитывать обновления данных;
- требуется корректная работа с лицензией OSM/ODbL;
- качество данных зависит от региона.

---

# ADR-005 — Exploration Graph and StreetSegment

## Context

OSM Way нельзя использовать как единицу исследования.

OSM Way является технической сущностью представления геометрии и может изменяться после:

- редактирования тегов;
- разделения way;
- объединения;
- изменения topology.

Также одна логическая улица может состоять из множества OSM Way.

---

## Decision

Единицей исследования является собственный:

```text
StreetSegment
```

StreetSegment строится из нормализованного pedestrian graph.

Типичный segment — участок между значимыми graph nodes.

Значимыми считаются:

- перекрёстки;
- развилки;
- конец пути;
- изменение pedestrian accessibility;
- изменение важных semantics;
- при необходимости искусственная точка разбиения слишком длинного ребра.

Пример:

```text
intersection A
      │
      │ StreetSegment #1
      │
intersection B
      │
      │ StreetSegment #2
      │
intersection C
```

Не использовать правило:

> один OSM Way = один StreetSegment

---

## Fixed-length segmentation

Сеть не должна по умолчанию искусственно разбиваться на одинаковые участки, например каждые 20–50 метров.

Причины:

- случайные границы;
- нестабильные ID;
- рост количества сущностей;
- пользователь начинает визуально «закрашивать клетки», а не исследовать реальные улицы.

Очень длинные topology edges допускается дополнительно разбивать.

Конкретная максимальная длина определяется после geo-spike.

---

# Walkability Classification

Не каждый путь, пригодный для routing, должен давать exploration progress.

Для нормализованных edges вводится classification:

```text
EXPLORE
ROUTABLE_ONLY
IGNORE
```

## EXPLORE

Участвует в routing и в exploration.

Примеры-кандидаты:

- residential streets;
- pedestrian streets;
- footways;
- набережные;
- парковые дорожки;
- alley / passage, если это осмысленный городской pedestrian path.

---

## ROUTABLE_ONLY

Используется для связности routing graph, но не увеличивает progress.

Примеры-кандидаты:

- технические соединения;
- короткие connector paths;
- некоторые service accesses;
- переходные участки, не воспринимаемые как отдельная часть города.

---

## IGNORE

Не используется для exploration и, как правило, не должен использоваться для pedestrian routing.

Примеры:

- private;
- недоступные территории;
- технические дороги;
- явно непешеходные объекты.

---

## Consequence

WalkabilityProfile становится отдельной частью geo-normalization.

OSM tags переводятся во внутреннюю semantics, вместо того чтобы распространяться по всему приложению.

---

# ADR-006 — Exploration Semantics

## Context

Простейшая формула:

```text
visited streets / all streets
```

не подходит, потому что улицы существенно различаются по длине.

Поэтому progress должен вычисляться по StreetSegment.

---

## Decision

Архитектурная формула:

```text
district_exploration =
sum(weight(explored segments))
/
sum(weight(explorable segments))
```

Для MVP:

```text
weight(segment) = segment.length_m
```

То есть:

```text
district_exploration =
explored walkable length
/
total explorable walkable length
```

---

# Extensible weighting

Модель должна позволять позднее использовать разные веса.

Например, потенциально:

```text
city street         1.0
pedestrian street   1.0
park path           0.5
minor passage       0.2
```

Однако в MVP weighting не усложняется.

Все explorable segments получают вес, равный длине.

---

# Segment Coverage

Сам факт касания segment не означает, что он исследован.

Не использовать:

```text
route intersects segment → explored
```

Нужен coverage threshold.

Концептуально:

```text
covered_length >= required_coverage
```

Порог должен быть параметризован.

Возможная формула для экспериментов:

```text
required =
clamp(
    segment.length * coverage_ratio,
    min_required_m,
    max_required_m
)
```

Конкретные значения:

- `coverage_ratio`;
- `min_required_m`;
- `max_required_m`;

не являются принятым продуктовым решением и должны быть определены на geo-spike.

---

# UserStreetProgress

Рекомендуемая модель:

```text
UserStreetSegmentProgress
- user_id
- street_segment_id
- covered_length_m
- coverage_ratio
- first_visited_at
- last_visited_at
- visit_count
- completed_at?
```

Даже если MVP использует в основном binary state `explored / not explored`, модель должна позволять развить partial coverage.

---

# Partial Coverage

Долгосрочно предпочтительна cumulative coverage semantics.

Пример:

Первая прогулка:

```text
segment:
████████░░░░░░░░░░░░░░░░
```

Вторая прогулка:

```text
segment:
░░░░░░░░░░░░████████████
```

Совокупное покрытие должно учитывать union пройденной геометрии.

Для MVP это может быть отложено, если StreetSegment достаточно короткие и используется completion threshold.

---

# Persistent Exploration vs Walk Summary

Это две разные метрики.

## Persistent district progress

Считает exploration state пользователя.

```text
sum(completed segment weight)
/
sum(explorable segment weight)
```

---

## Walk Summary

Может считать:

```text
unique_new_route_length
```

то есть фактическую длину сегодняшнего маршрута по ранее не исследованной сети.

Поэтому допустим сценарий:

```text
+3.1 km new streets
District: 31.4% → 33.8%
```

Эти числа не обязаны быть прямыми преобразованиями друг друга.

---

# GeoDataVersion

StreetSegment принадлежит конкретной версии geo graph.

```text
GeoDataVersion
- id
- city_id
- source
- imported_at
- checksum
```

```text
StreetSegment
- id
- geo_data_version_id
- geometry
- ...
```

Завершённый Route / Walk также сохраняет версию данных, на которой выполнялся matching.

---

# Segment Identity

Не пытаться создавать «вечные» StreetSegment ID на основе OSM IDs или geometry hashes.

После изменения исходной сети:

```text
GeoDataVersion A
    ↓
Segments A

GeoDataVersion B
    ↓
Segments B
```

идентичность segments может измениться.

---

# Rebuildability

Ключевой инвариант:

> **User exploration state является производным и полностью rebuildable.**

Источник истины:

```text
Completed Walk
+
Final Route Geometry
+
GeoDataVersion
```

Материализованное состояние:

```text
UserStreetProgress
```

можно удалить и пересчитать.

---

# Geo Update Strategy

Для ранних версий не реализовывать сложный segment-to-segment reconciliation между версиями street graph.

После обновления geo data предпочтительный подход:

```text
all completed Walk geometry
        ↓
new GeoDataVersion
        ↓
route-to-segment matching
        ↓
rebuild UserStreetProgress
```

Это вычислительно дороже, но значительно проще и надёжнее.

Incremental reconciliation можно добавить позже при необходимости.

---

# Routing Engine Boundary

Routing engine является infrastructure component, но не владельцем exploration identity.

Нельзя:

```text
Valhalla edge id
      ↓
UserStreetProgress
```

Правильная архитектура:

```text
OSM Dataset
    ↓
Internal Geo Import
    ↓
Internal StreetSegment Graph
       ↑
       │ geometry / topology matching
       │
Routing Engine Route Geometry
```

Routing engine выдаёт маршрут.

Наш Geo/Exploration module определяет, каким StreetSegment он соответствует.

---

# Routing Engine Shortlist

Первый кандидат:

## 1. Valhalla

Почему рассматривается первым:

- open-source;
- OSM-based;
- pedestrian routing;
- map matching;
- подходит к будущему GPS pipeline;
- поддерживает additional geo capabilities.

---

## 2. GraphHopper

Резервный кандидат.

Проверить:

- pedestrian routing quality;
- self-hosting complexity;
- map matching;
- operational footprint.

---

## 3. OSRM

Использовать как baseline для сравнения.

Основное внимание — suitability для pedestrian/exploration use case.

---

# Decision Status

Routing engine пока **не выбран окончательно**.

Необходимо провести spike:

```text
Valhalla
vs
GraphHopper
vs
OSRM
```

на одном и том же SPb dataset.

---

# Geo Spike

До финализации exploration formula необходимо сделать reproducible prototype.

## Цель

Проверить, что модель:

```text
OSM
→ walkable graph
→ StreetSegments
→ route matching
→ exploration overlay
```

визуально и семантически соответствует продукту.

---

# Test Areas

В Санкт-Петербурге взять минимум три разных типа городской среды.

## Area A — Dense center

Характеристики:

- плотная street grid;
- перекрёстки;
- короткие кварталы;
- большое число parallel pedestrian choices.

---

## Area B — Regular urban district

Например типичная сетка Петроградской стороны / Васильевского острова.

Цель:

- проверить обычные городские улицы;
- длину segments;
- intersections;
- дворовые соединения.

---

## Area C — Park + Residential

Характеристики:

- park footways;
- irregular paths;
- residential streets;
- service roads;
- pedestrian connectors.

Цель:

- проверить `EXPLORE / ROUTABLE_ONLY / IGNORE`;
- не перегружают ли парк и мелкие paths процент района.

---

# Real Routes

На каждую или суммарно по областям использовать 3–5 реальных прогулок.

Проверять:

1. Как route ложится на StreetSegment.
2. Нет ли слишком крупных segments.
3. Нет ли чрезмерно мелких segments.
4. Какие OSM ways неожиданно получают `EXPLORE`.
5. Какие полезные pedestrian paths ошибочно исключены.
6. Как выглядит закрашенная exploration map.
7. Какой coverage threshold ощущается естественно.
8. Насколько стабилен route matching.

---

# Geo Spike Outputs

Spike должен дать следующие артефакты:

```text
1. Imported test OSM dataset
2. Normalized pedestrian graph
3. Generated StreetSegments
4. Walkability classification
5. Map visualization
6. Sample route overlays
7. Segment coverage visualization
8. Comparison of threshold variants
9. Routing engine comparison
10. Recommended MVP parameters
```

---

# Questions the Spike Must Answer

## Segmentation

- Нужна ли максимальная длина StreetSegment?
- Если да, какая?
- Нужны ли дополнительные split points кроме graph topology?

## Walkability

- Какие OSM highway/path classes относятся к EXPLORE?
- Какие должны быть ROUTABLE_ONLY?
- Как обрабатывать service roads?
- Как обрабатывать дворы?
- Как учитывать парковые дорожки?

## Coverage

- Какой ratio выглядит корректно?
- Нужен ли minimum threshold?
- Нужен ли maximum threshold?
- Нужна ли partial exploration уже в MVP?

## Routing

- Какой engine лучше работает на реальных pedestrian routes SPb?
- Насколько отличаются route geometry?
- Как работает map matching на специально испорченных GPS traces?

---

# MVP Decision After Spike

После spike необходимо финализировать:

```text
ADR-005a StreetSegment generation parameters
ADR-006a Exploration threshold parameters
ADR-003 Routing Engine
```

До spike не фиксировать числа только теоретически.

---

# Accepted Principles

На текущем этапе считаются принятыми следующие принципы:

1. OpenStreetMap — основной upstream geo source.
2. Raw OSM entities не являются domain identity.
3. Exploration строится на собственном StreetSegment graph.
4. StreetSegment не равен OSM Way.
5. Segmentation следует topology, а не произвольной фиксированной сетке.
6. Пути классифицируются как `EXPLORE / ROUTABLE_ONLY / IGNORE`.
7. MVP progress взвешивается по длине explorable segments.
8. Coverage threshold обязателен и параметризован.
9. Walk Summary и persistent district progress считаются отдельно.
10. Exploration state rebuildable.
11. Final Walk geometry сохраняется.
12. Geo graph versioned.
13. Routing engine internal IDs не используются как exploration identity.
14. На раннем этапе geo updates безопаснее обрабатывать полным rebuild, а не сложным reconciliation.
15. Перед финализацией параметров обязателен geo-spike на реальных данных Санкт-Петербурга.
