# ГуляЕм — Stage 1 Requirements: Geo Exploration Playground

**Status:** Draft v0.1  
**Stage:** 1 — Geo Exploration Playground  
**Primary city:** Санкт-Петербург  
**Backend:** Go  
**Frontend:** React + TypeScript  
**Storage:** PostgreSQL + PostGIS  
**Map renderer:** MapLibre GL JS  
**Initial map delivery:** bbox + GeoJSON

---

## 1. Goal

Проверить фундаментальную geo-модель «ГуляЕм» до реализации пользовательского exploration loop.

Главный вопрос:

> Является ли собственный `StreetSegment` естественным и визуально понятным представлением исследуемого города?

Stage должен позволять взять реальные OSM-данные Санкт-Петербурга, построить внутренний pedestrian/exploration graph и проверить его через web playground.

---

## 2. Primary pipeline

```text
OpenStreetMap
      ↓
Geo Import
      ↓
Normalization
      ↓
Pedestrian / Exploration Graph
      ↓
StreetSegment Generation
      ↓
Walkability Classification
      ↓
GeoDataVersion
      ↓
PostgreSQL / PostGIS
      ↓
Go Backend API
      ↓
React + MapLibre Geo Playground
```

Дополнительно:

```text
Sample Route
      ↓
Prototype Route Matching
      ↓
Segment Coverage
      ↓
Coverage Visualization
```

---

## 3. Stage validation areas

Stage должен ответить на четыре группы вопросов.

### Segmentation

- какие topology nodes создают split;
- возникают ли слишком короткие segments;
- возникают ли слишком длинные segments;
- нужен ли `max_segment_length_m`;
- нужны ли artificial split points;
- не превращается ли сеть в визуальную сетку из искусственных одинаковых фрагментов.

### Walkability

Проверить classification:

```text
EXPLORE
ROUTABLE_ONLY
IGNORE
```

В том числе правила для:

- residential streets;
- pedestrian streets;
- footways;
- paths;
- service roads;
- courtyards;
- park paths;
- passages / alleys;
- private / restricted paths;
- technical connectors.

### Coverage

Экспериментально проверить:

```text
coverage_ratio
min_required_m
max_required_m
partial coverage
```

Концептуальная формула:

```text
required = min(
    segment.length,
    clamp(
        segment.length * coverage_ratio,
        min_required_m,
        max_required_m
    )
)
```

### Routing

Сравнить:

```text
Valhalla
GraphHopper
OSRM
```

по pedestrian routing, геометрии, self-hosting, operational footprint, integration complexity и пригодности к будущему map matching.

---

## 4. Backend technology

Использовать Go.

Предпочтительно:

```text
Go
net/http
chi
pgx
```

Не использовать heavyweight framework без доказанной необходимости.

Backend остаётся modular monolith.

Отдельный `geo-import` — executable внутри того же codebase, не microservice.

---

## 5. Frontend technology

Использовать:

```text
React
TypeScript
Vite
MapLibre GL JS
```

React выбран, потому что Geo Playground будет состоять из независимых интерактивных компонентов:

- map;
- layer controls;
- filters;
- segment inspector;
- geo-version information;
- route overlay;
- coverage controls;
- statistics;
- debug panels.

Frontend framework не является частью geo domain contract.

---

## 6. Persistence

Основное хранилище:

```text
PostgreSQL + PostGIS
```

PostGIS используется для:

- geometry storage;
- spatial indexes;
- bbox selection;
- intersections;
- distance;
- length;
- district assignment;
- route/segment experiments.

Минимальные spatial indexes:

```text
GIST(street_segments.geometry)
GIST(districts.boundary)
```

---

## 7. Geo source

Primary upstream:

```text
OpenStreetMap
```

Предпочтительный source artifact:

```text
*.osm.pbf
```

Import должен быть воспроизводимым и не строиться вокруг большого количества online OSM API requests.

---

## 8. OSM isolation

OSM не является domain model.

Не использовать как persistent domain identity:

```text
osm_way_id
osm_node_id
routing_engine_edge_id
```

Raw OSM metadata может храниться для import/debug/troubleshooting, но не должно быть основным публичным контрактом приложения.

---

## 9. GeoDataVersion

Минимальная модель:

```text
GeoDataVersion
- id
- city_id
- source
- source_timestamp?
- imported_at
- source_checksum
- normalization_version
- status
```

Status:

```text
IMPORTING
READY
FAILED
SUPERSEDED
```

Partial failed import не должен становиться current `READY` version.

---

## 10. Core geo entities

Минимально:

```text
City
District
GeoDataVersion
Street
StreetSegment
```

### City

```text
City
- id
- name
- country_code
- timezone
- boundary
```

Первая city fixture — Санкт-Петербург.

City-specific business logic запрещена.

### District

```text
District
- id
- city_id
- external_id?
- name
- kind
- boundary
```

### Street

```text
Street
- id
- city_id
- name?
- normalized_name?
```

`Street` не является единицей exploration.

На Stage 1 `StreetSegment.street_id` может быть nullable.

### StreetSegment

```text
StreetSegment
- id
- city_id
- geo_data_version_id
- street_id?
- geometry
- length_m
- classification
- attributes
```

Classification:

```text
EXPLORE
ROUTABLE_ONLY
IGNORE
```

`StreetSegment.id` имеет смысл внутри `GeoDataVersion`; вечная identity между версиями не требуется.

---

## 11. StreetSegment generation

Основное правило:

> StreetSegment формируется преимущественно между значимыми topology nodes.

Split candidates:

- intersection;
- branch;
- dead end;
- pedestrian accessibility change;
- important semantic change;
- artificial split point только при подтверждённой необходимости.

Не использовать как основную стратегию:

```text
каждые N метров → новый segment
```

Stage должен поддерживать экспериментальный `max_segment_length_m`, но не фиксировать его заранее.

---

## 12. Walkability normalization

Должен существовать единый `WalkabilityProfile`.

Он преобразует upstream semantics во внутренние:

```text
EXPLORE
ROUTABLE_ONLY
IGNORE
```

Classification logic не дублируется во frontend/API/routing layer.

Для debug mode сохраняется объяснение:

```text
classification
reason
relevant source metadata
```

чтобы инженер мог понять, почему segment классифицирован именно так.

---

## 13. Geometry

Preferred CRS for application transport/storage representation:

```text
EPSG:4326
```

`StreetSegment.geometry` предпочтительно связный `LineString`.

Import должен проверять:

- empty geometry;
- invalid coordinates;
- zero length;
- unexpected duplicate geometry;
- disconnected topology anomalies;
- extremely short segments;
- extremely long segments.

---

## 14. Initial map delivery

Stage 1 использует:

> bbox + GeoJSON

Frontend запрашивает только текущий viewport.

Не загружать всю street network Санкт-Петербурга одним ответом.

Backend должен ограничивать чрезмерно большой bbox и/или feature count.

Vector tiles являются future optimization и не входят в Stage 1.

---

## 15. Minimal API

### Current GeoDataVersion

```text
GET /api/v1/cities/{cityId}/geo-version
```

### Street segments

```text
GET /api/v1/geo/segments
```

Parameters:

```text
cityId
bbox
classification?
minLength?
maxLength?
```

Response:

```text
GeoJSON FeatureCollection
```

Feature properties минимум:

```text
id
classification
lengthMeters
streetName?
```

### Segment detail

```text
GET /api/v1/geo/segments/{segmentId}
```

Returns:

- id;
- GeoDataVersion;
- geometry/length;
- classification;
- normalized attributes;
- Street;
- District;
- classification explanation;
- source debug metadata only in explicit internal mode.

### Districts

```text
GET /api/v1/geo/districts?cityId=...&bbox=...
```

Response: GeoJSON FeatureCollection.

---

## 16. Geo Playground

Engineering route:

```text
/debug/geo
```

Будущий product `/map` не должен зависеть от debug UI.

Playground должен поддерживать:

- pan;
- zoom;
- mobile gestures;
- desktop mouse interaction;
- configurable base-map style;
- required attribution.

Default Stage 1 viewport: Санкт-Петербург.

---

## 17. Layer controls

Независимые layers:

```text
Base Map
Districts
EXPLORE
ROUTABLE_ONLY
IGNORE
StreetSegment boundaries
Test Routes
Coverage
```

Classification должна визуально различаться.

Debug visual style не обязан совпадать с final product design.

---

## 18. Segment Inspector

Tap/click по segment показывает минимум:

```text
Segment ID
GeoDataVersion
Classification
Length
Street
District
Normalized attributes
Classification reason
```

Development/internal mode дополнительно может показывать source metadata.

---

## 19. Filters

Обязательные:

```text
classification
length range
```

Optional если scope остаётся контролируемым:

```text
street
district
normalized path type
```

---

## 20. Statistics

Для test area или viewport показывать минимум:

```text
segments total
EXPLORE count
ROUTABLE_ONLY count
IGNORE count
total length
explorable length
min segment length
median segment length
p95 segment length
max segment length
```

---

## 21. Test areas

Минимум три reproducible areas Санкт-Петербурга.

### A — Dense Center

- dense grid;
- intersections;
- short blocks;
- parallel pedestrian choices.

### B — Regular Urban District

- residential streets;
- regular blocks;
- courtyards;
- pedestrian connectors.

### C — Park + Residential

- park paths;
- irregular graph;
- residential;
- service roads;
- connectors.

Test areas должны храниться fixtures в repository.

---

## 22. Sample routes

Минимум:

```text
3–5 real or realistic walking routes
```

Хранить воспроизводимо в repository, предпочтительно GeoJSON.

Playground должен отображать:

- route geometry;
- matched StreetSegments;
- unmatched fragments.

---

## 23. Prototype route matching

Stage 1 содержит prototype geometry-to-segment matching только для geo validation.

Pipeline:

```text
Sample Route Geometry
        ↓
Candidate StreetSegments
        ↓
Geometry Matching
        ↓
Matched Length
        ↓
Coverage
```

Это не production `Route` domain Stage 2/3.

---

## 24. Coverage experiments

Для matched segment:

```text
covered_length_m
coverage_ratio
```

Threshold parameters должны быть изменяемыми без переписывания алгоритма.

Playground должен различать:

- completed segment;
- partially covered segment;
- not covered segment;
- route fragment without confident match.

---

## 25. Routing engine spike

Сравнить:

```text
Valhalla
GraphHopper
OSRM
```

на одинаковом dataset и test fixtures.

Для каждого зафиксировать:

```text
route geometry
distance
calculation time
setup complexity
memory footprint
container/image footprint
pedestrian quality
map matching capability
```

Routing engine output рассматривается как geometry/capability, а не владелец StreetSegment identity.

---

## 26. Configuration

Backend минимум:

```text
DATABASE_URL
HTTP_ADDRESS
ENVIRONMENT
GEO_DATA_PATH
GEO_TEST_AREA?
LOG_LEVEL
```

Frontend минимум:

```text
VITE_API_URL
VITE_MAP_STYLE_URL
```

Secrets/environment-specific values не hardcode в source.

---

## 27. Reproducibility

Новый разработчик должен иметь documented flow:

```text
clone
↓
start dependencies
↓
run migrations
↓
run geo import
↓
start backend
↓
start frontend
↓
open /debug/geo
↓
see StreetSegments
```

---

## 28. Performance targets

Engineering targets, not SLA:

- typical bbox API p95 < 500 ms on representative warmed local environment;
- frontend map remains interactive for representative test-area viewport;
- API protects browser from accidentally receiving massive city-wide FeatureCollection.

Vector tiles вводятся только после измеренной необходимости.

---

## 29. Observability

Structured logs.

Import summary минимум:

```text
source objects processed
pedestrian candidates
segments generated
EXPLORE count
ROUTABLE_ONLY count
IGNORE count
total length
explorable length
invalid geometries
zero-length segments
short-segment count
long-segment count
import duration
```

API metrics минимум:

```text
request count
latency
error count
returned feature count
bbox size
```

---

## 30. Testing

### Unit

- OSM attributes → classification;
- synthetic graph segmentation;
- geometry helpers.

### Integration

Against real PostgreSQL/PostGIS:

- migrations;
- spatial indexes;
- bbox;
- district intersection;
- persistence roundtrip.

### Fixture invariants

Каждый test area:

```text
segments > 0
EXPLORE > 0
invalid geometry = 0
all segment lengths > 0
all segments belong to current GeoDataVersion
```

### Frontend

Минимально:

- layer toggle;
- filters;
- segment selection;
- error state;
- empty dataset;
- viewport reload.

### Manual

Обязательная визуальная проверка всех test areas.

---

## 31. Manual validation checklist

Для каждой area проверить:

1. слишком длинные segments;
2. слишком короткие segments;
3. splits на intersections;
4. бессмысленное дробление;
5. ошибочные `EXPLORE`;
6. ошибочные `ROUTABLE_ONLY`;
7. пропущенные полезные pedestrian paths;
8. private paths в graph;
9. влияние park paths;
10. естественность exploration overlay;
11. route matching;
12. coverage behavior.

---

## 32. Explicit non-goals

Stage 1 НЕ реализует:

- authentication;
- User;
- UserStreetProgress;
- Walk;
- production Manual Route Builder;
- GPS tracking;
- GPX product import;
- Places;
- Visits;
- Photos;
- Sharing;
- Recommendations;
- Social;
- offline maps;
- production vector tiles;
- sophisticated reconciliation between GeoDataVersions.

---

## 33. Expected Stage outputs

```text
1. Go backend
2. React Geo Playground
3. PostgreSQL/PostGIS schema
4. OSM import command
5. GeoDataVersion
6. internal StreetSegment generator
7. WalkabilityProfile
8. 3 test areas
9. 3–5 sample routes
10. classification visualization
11. coverage visualization
12. statistics
13. routing engine comparison
14. automated tests
15. manual validation report
16. resulting ADR decisions
```

---

## 34. Resulting decisions

Stage должен завершиться решениями:

- Map renderer;
- Routing engine;
- StreetSegment split rules;
- maximum segment length or explicit decision not to use one;
- initial walkability mapping;
- coverage threshold parameters or documented blocker;
- partial-coverage MVP decision;
- confirmation/rejection of bbox + GeoJSON for next stage.

---

## 35. Definition of Done

Stage 1 завершён не тогда, когда OSM импортирован или segments появились в БД.

Stage завершён, когда Geo Playground позволяет воспроизводимо проверить разные типы городской среды, реальные маршруты и coverage, после чего можно обоснованно утверждать:

> Внутренняя StreetSegment-модель достаточно хорошо соответствует тому, как человек воспринимает исследование улиц, чтобы использовать её как основу Stage 2.
