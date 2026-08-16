# ГуляЕм — Stage 2 Requirements: Manual Route & Exploration Preview

**Status:** Draft v0.1  
**Stage:** 2 — Manual Route & Exploration Preview  
**Backend:** Go  
**Frontend:** React + TypeScript  
**Routing:** Valhalla, pedestrian profile  
**Geo:** frozen Stage 1 StreetSegment model  
**Storage:** PostgreSQL + PostGIS  
**Persistence of route preview:** none

---

# 1. Goal

Stage 2 должен реализовать первую пользовательскую capability, которая использует реальный geo core:

> пользователь вручную задаёт маршрут на карте и сразу видит построенный pedestrian route и то, какие `StreetSegment` этот маршрут потенциально позволит исследовать.

Основной flow:

```text
Map
 ↓
Select start
 ↓
Select destination
 ↓
Optional waypoints
 ↓
Valhalla
 ↓
Route Geometry
 ↓
StreetSegment Matching
 ↓
Coverage Preview
 ↓
Route Preview UI
```

---

# 2. Validation question

Главный product/UX вопрос Stage 2:

> Понятно ли пользователю, куда он пойдёт и какие участки городской сети эта прогулка потенциально засчитает?

Stage 2 не проверяет retention, Walk lifecycle или persistence.

---

# 3. Inputs inherited from Stage 1

Stage 2 не должен повторно открывать без причины:

- OSM source decision;
- StreetSegment identity;
- topology split rules;
- WalkabilityProfile v1;
- `EXPLORE / ROUTABLE_ONLY / IGNORE`;
- grade-aware matching;
- Balanced coverage semantics;
- routing-engine selection;
- PostGIS/bbox foundation.

Frozen defaults:

```text
Routing engine: Valhalla
Routing profile: pedestrian

Coverage:
radius = 100 m
ratio = 0.4
min_required = 15 m
max_required = 80 m

max_segment_length_m = 0
```

---

# 4. Critical scope boundary: no personal exploration yet

Stage 2 не имеет:

```text
User
UserStreetProgress
Completed Walk history
```

Следовательно Stage 2 не может корректно вычислять:

```text
new streets
already explored
district progress delta
personal new ratio
```

Stage 2 вычисляет только:

> какие explorable segments данный маршрут потенциально покрывает согласно frozen coverage semantics.

Допустимые product labels:

- «Потенциально исследуется»;
- «Будут засчитаны сегменты»;
- «Частично покрывается».

Недопустимо до Stage 3:

- «Новые улицы для вас»;
- «+3 км новых улиц»;
- «район 31% → 34%».

---

# 5. Product scenario

## 5.1. Enter builder

На `/map` пользователь нажимает:

> **+ Прогулка**

Приложение переходит в route-building mode.

---

## 5.2. Start

Первый map tap устанавливает start waypoint:

```text
A
```

---

## 5.3. Destination

Второй map tap устанавливает destination:

```text
A → B
```

После появления минимум двух waypoints автоматически запрашивается preview.

---

## 5.4. Intermediate waypoints

Пользователь может:

- добавить промежуточную точку;
- переместить точку;
- удалить промежуточную точку;
- изменить start;
- изменить destination;
- изменить порядок intermediate waypoints.

Пример:

```text
A → B → C → D
```

где A — start, D — destination.

---

## 5.5. Route preview

После любого завершённого изменения waypoint route пересчитывается.

Пользователь видит минимум:

```text
distance
estimated walking duration
route geometry
potential exploration coverage
```

---

## 5.6. End of Stage 2 flow

Stage 2 заканчивается на preview.

Stage 2 НЕ реализует:

```text
Start Walk
Save Walk
Complete Walk
```

Эти действия относятся к Stage 3.

---

# 6. Waypoint requirements

Минимальное число:

```text
2
```

Рекомендуемый Stage 2 maximum:

```text
10
```

Backend должен валидировать:

- latitude `[-90, 90]`;
- longitude `[-180, 180]`;
- finite numbers;
- minimum waypoint count;
- maximum waypoint count.

Frontend validation не заменяет backend validation.

---

# 7. Waypoint interaction model

Minimum UX:

### Empty builder

```text
Tap map → start
```

### Start selected

```text
Tap map → destination
```

### Route exists

Intermediate waypoint добавляется через явное action:

> **+ Точка**

После activation следующий map tap добавляет waypoint перед destination.

Markers должны позволять drag.

Routing request отправляется на `dragend`, а не на каждый pointer event.

---

# 8. Waypoint list

Bottom sheet / side panel должен отображать ordered list:

```text
Start
Waypoint 1
Waypoint 2
Destination
```

Minimum actions:

- select;
- delete intermediate;
- reorder intermediate;
- clear route.

Start/destination могут перемещаться, но не удаляются по отдельности так, чтобы оставалось некорректное однопунктовое preview state без явного UI state.

---

# 9. Routing architecture

Stage 2 вводит production-shaped routing boundary.

Conceptual interface:

```text
RoutingEngine
  Route(ctx, RouteRequest) -> RouteResult
```

Domain/application request не должен содержать Valhalla-specific fields.

Input:

```text
waypoints
profile = PEDESTRIAN
```

Output минимум:

```text
geometry
distance_m
duration_sec
resolved_waypoints?
engine_metadata
```

Не возвращать наружу как domain identity:

```text
Valhalla edge ID
tile ID
internal graph ID
```

---

# 10. Valhalla adapter

Valhalla adapter должен:

- преобразовывать application request в Valhalla API;
- использовать pedestrian costing/profile;
- устанавливать upstream timeout;
- проверять response;
- преобразовывать geometry во внутренний GeoJSON `LineString`;
- преобразовывать distance/duration;
- нормализовать error types.

Raw Valhalla response не должен проксироваться frontend.

---

# 11. Valhalla development runtime

Valhalla из Stage 1 spike становится обычной Stage 2 development dependency.

Development flow должен позволять:

```text
source OSM PBF
 ↓
Valhalla graph build
 ↓
routing dataset metadata
 ↓
Valhalla service
 ↓
Go backend
```

Graph artifacts могут оставаться ignored generated data.

Pinned engine version из Stage 1 decision сохраняется до отдельного upgrade decision.

---

# 12. Routing dataset metadata

Рядом с generated routing graph должен существовать machine-readable metadata artifact.

Минимально:

```json
{
  "engine": "valhalla",
  "engineVersion": "...",
  "sourceChecksum": "...",
  "profile": "pedestrian"
}
```

Допустимо добавить:

```text
builtAt
sourceFile
buildConfigVersion
```

---

# 13. Geo/routing compatibility

Перед route preview backend получает current READY `GeoDataVersion`.

Проверяется:

```text
GeoDataVersion.source_checksum
==
RoutingDataset.sourceChecksum
```

При mismatch preview не должен молча продолжаться.

Expected error:

```text
409 routing_geo_version_mismatch
```

Reason:

routing geometry и StreetSegment analysis должны относиться к совместимой source topology.

---

# 14. Stateless RoutePreview

Stage 2 RoutePreview — application result, а не persistent aggregate.

Preview:

- не получает permanent product ID;
- не записывается в `routes`;
- не создаёт Walk;
- может быть пересчитан сколько угодно раз.

Stage 3 решит, как materialize preview в persistent `Route`.

---

# 15. RoutePreview pipeline

```text
RoutePreviewRequest
       ↓
Validate waypoints
       ↓
Current GeoDataVersion
       ↓
Routing dataset compatibility
       ↓
Valhalla Route
       ↓
Route geometry
       ↓
Geo route analysis
       ↓
Balanced coverage
       ↓
RoutePreviewResponse
```

---

# 16. Reuse of Stage 1 route analysis

Stage 1 matcher и coverage algorithm должны использоваться повторно.

Current fixture-dependent service следует разделить концептуально:

```text
RouteAnalyzer
- repository
- matching
- coverage

SampleRouteService
- fixtures
- RouteAnalyzer
```

Stage 2:

```text
RoutePreviewService
- RoutingEngine
- RouteAnalyzer
```

Не делать второй matcher специально для product route preview.

---

# 17. Stage 2 matching parameters

По умолчанию Stage 2 использует frozen/default Stage 1 matching parameters.

Debug/internal configuration может позволять их просматривать, но product UI не должен предоставлять пользователю tuning controls.

Coverage profile product preview:

```text
Balanced
```

Strict/Generous остаются `/debug/geo` instrumentation.

---

# 18. Potential exploration preview

Preview должен различать минимум:

```text
COMPLETED
PARTIAL
NOT_COVERED
CONNECTOR
```

Product map обычно отображает:

- `COMPLETED` explorable segments;
- `PARTIAL` explorable segments;
- main route.

`ROUTABLE_ONLY/CONNECTOR` используется для route connectivity, но не представляется как exploration reward.

`NOT_COVERED` background network не обязательно подсвечивать в product preview.

---

# 19. Product preview metrics

Minimum visible metrics:

```text
route distance
estimated duration
potential completed segment count
potential partial segment count
```

Дополнительно допустимо показывать:

```text
completed explorable network length
```

но wording должен явно относиться к покрываемой сети, а не к фактической длине сегодняшнего маршрута.

Не использовать `CompletedNetworkRatio` из 225 м context как «процент нового маршрута».

---

# 20. Stage 2 analysis metrics

Backend response должен сохранить диагностические metrics:

```text
routeMatchedRatio
routeUnmatchedLengthMeters
completedNetworkLengthMeters
contextExplorableLengthMeters
completedNetworkRatio
```

Рекомендуется добавить для product interpretation:

```text
matchedExplorableRouteLengthMeters
matchedRoutableOnlyRouteLengthMeters
completedSegmentCount
partialSegmentCount
```

Эти метрики должны иметь однозначную семантику и тесты.

---

# 21. Low match behavior

Route preview не блокируется автоматически только из-за неполного StreetSegment matching.

Если:

```text
routeMatchedRatio < 0.95
```

response содержит warning, например:

```text
low_route_match
```

Unmatched fragments остаются доступны для diagnostics.

Product UI может показать компактное сообщение:

> Часть маршрута не удалось точно сопоставить с городской сетью.

Debug UI показывает детали.

Threshold `0.95` зафиксирован Stage 2 evidence: нормальные fixtures дают около 99–100%, а
intentionally ambiguous fixture — `0.914470569`. Ровно `0.95` не создаёт warning. Это именованная
application constant, а не user setting или deployment configuration.

---

# 22. API endpoint

Primary endpoint:

```text
POST /api/v1/route-previews
```

Preview не создаёт ресурс, поэтому отдельные:

```text
GET /route-previews/{id}
DELETE /route-previews/{id}
```

не требуются.

---

# 23. Request contract

Example:

```json
{
  "cityId": "uuid",
  "waypoints": [
    { "lat": 59.935, "lon": 30.325 },
    { "lat": 59.931, "lon": 30.340 }
  ],
  "profile": "pedestrian"
}
```

Stage 2 поддерживает только:

```text
profile = pedestrian
```

Unknown fields should be rejected consistently with existing API conventions.

---

# 24. Response contract

Conceptual response:

```json
{
  "geoDataVersion": {
    "id": "...",
    "sourceChecksum": "...",
    "normalizationVersion": "stage1-segments-v1"
  },
  "routing": {
    "engine": "valhalla",
    "profile": "pedestrian",
    "distanceMeters": 4200,
    "durationSeconds": 3600,
    "geometry": {
      "type": "LineString",
      "coordinates": []
    },
    "waypoints": []
  },
  "explorationPreview": {
    "normalizedRoute": {
      "type": "MultiLineString",
      "coordinates": []
    },
    "coverageProfile": {
      "name": "balanced",
      "radiusMeters": 100,
      "coverageRatio": 0.4,
      "minRequiredMeters": 15,
      "maxRequiredMeters": 80
    },
    "matchedFragments": [],
    "unmatchedFragments": [],
    "coverageSegments": [],
    "metrics": {}
  },
  "warnings": []
}
```

Final field names may follow repository conventions, but semantic separation between routing result and exploration analysis must remain.

---

# 25. Response geometry format

Stage 2 uses GeoJSON consistently with Stage 1:

```text
LineString
MultiLineString
FeatureCollection
```

No encoded polyline is required until measured payload pressure justifies it.

---

# 26. Error contract

Minimum normalized errors:

### Invalid request

```text
400 invalid_body
```

### Invalid waypoint semantics

```text
422 invalid_waypoints
```

### No pedestrian route

```text
422 route_not_found
```

### Routing dataset / GeoDataVersion mismatch

```text
409 routing_geo_version_mismatch
```

### Routing service unavailable

```text
503 routing_unavailable
```

### Routing timeout

```text
504 routing_timeout
```

### Current geo data unavailable

```text
503 geo_data_unavailable
```

Internal Valhalla error bodies must not leak directly to client.

---

# 27. Request concurrency

Frontend must handle rapid edits correctly.

Required:

- cancel previous request where possible (`AbortController`);
- assign logical request sequence/version;
- ignore responses for stale waypoint state;
- never apply old route geometry over a newer waypoint layout.

---

# 28. Recalculation policy

Do NOT call backend on every pointer move.

Recalculate on:

- destination set;
- waypoint added;
- waypoint removed;
- waypoint reordered;
- marker `dragend`.

A small debounce is allowed for discrete UI events, but continuous routing while dragging is not required.

---

# 29. Loading behavior

While new preview is calculating:

- waypoint state updates immediately;
- previous route must be visually marked stale or temporarily hidden;
- UI shows calculating state;
- stale preview must not look current.

Avoid map flicker when possible.

---

# 30. `/map` product screen

Stage 2 introduces first real product interaction on:

```text
/map
```

`/debug/geo` remains engineering playground.

Shared map primitives may be reused, but:

- debug filters do not appear by default in `/map`;
- source metadata does not appear in product UI;
- route builder state does not depend on sample-route fixtures.

---

# 31. Route builder visual hierarchy

Map remains primary canvas.

Product UI should prioritize:

1. waypoint markers;
2. primary route line;
3. potential completed/partial exploration;
4. distance/duration sheet;
5. editing actions.

Background StreetSegment network should not overpower route preview.

---

# 32. Product layers

Suggested product layers:

```text
base-map
optional background StreetSegments at detailed zoom
potential-completed
potential-partial
route-line
waypoint-markers
```

Unmatched/matching debug layers belong primarily to `/debug/geo`, although a compact warning may appear on `/map`.

---

# 33. Route summary sheet

Minimum:

```text
4.2 км
≈ 1 ч
Потенциально засчитывается: 27 сегментов
Частично: 6
```

Exact copy is UX-tunable.

Не показывать personal progress delta.

---

# 34. Responsive behavior

Stage 2 remains mobile-first.

Mobile:

- full-screen map;
- bottom sheet;
- touch-friendly waypoint markers/actions.

Desktop:

- full map;
- side/bottom panel is acceptable;
- same route-building capability.

No separate desktop-only route builder.

---

# 35. Base map / StreetSegment loading

Background network continues using Stage 1 bbox + GeoJSON limits.

Route preview geometry itself must remain visible even at a zoom where full StreetSegment background is not requested.

This preserves ADR-0009 concept:

> route overview and detailed network inspection are different zoom states.

---

# 36. HTTP compression

Stage 1 measured 0.7–2.8 MB raw representative GeoJSON responses.

Stage 2 should add or explicitly validate HTTP compression for map GeoJSON before external product validation.

Vector tiles are still not automatically required.

Measure at least:

```text
raw bytes
compressed bytes
transfer time
parse/render behavior on a real mobile device
```

---

# 37. Performance targets

Existing project engineering target:

```text
route preview ≈ 1–2 seconds
```

Stage 2 must measure end-to-end preview latency and components:

```text
Valhalla route latency
candidate query latency
matching latency
coverage latency
serialization latency
total latency
```

If representative p95 misses the target, bottleneck must be understood before Stage 2 sign-off.

Do not change frozen coverage semantics solely to make benchmark green without an ADR.

---

# 38. Observability

Structured logs must not log exact waypoint coordinates by default.

Recommended metrics:

```text
route_preview_requests_total
route_preview_duration
routing_duration
route_analysis_duration
coverage_duration
waypoint_count
route_distance_bucket
route_match_ratio
route_preview_errors
routing_dataset_mismatch_total
```

Correlation/request ID must flow through backend calls.

---

# 39. Security / abuse basics

Even before auth:

- request body size limit;
- waypoint count limit;
- coordinate validation;
- upstream timeout;
- response size awareness;
- no arbitrary Valhalla parameter passthrough.

User-provided costing options are out of scope.

---

# 40. Persistence

Stage 2 requires no product persistence changes.

No table for:

```text
RoutePreview
Waypoint
Route
Walk
```

unless a separate technical need is proven.

GeoDataVersion and StreetSegment remain existing persisted inputs.

---

# 41. Offline behavior

Offline route generation is not required.

If network is unavailable:

- current waypoint draft may remain in frontend memory;
- route preview shows unavailable state.

Durable offline route drafts belong to later stages.

---

# 42. Explicit non-goals

Stage 2 does not implement:

- auth;
- personal progress;
- persistent Route;
- Walk;
- route start/finish;
- Route Review;
- GPS;
- GPX product import;
- Places;
- Visits;
- Photos;
- sharing;
- recommendations;
- automatic round trips;
- route alternatives;
- avoid/prefer street controls;
- POI-aware routing;
- social.

---

# 43. Stage outputs

Expected artifacts:

```text
1. Valhalla production-shaped adapter
2. normal dev Valhalla service/profile
3. routing graph metadata
4. geo/routing checksum compatibility
5. stateless route preview service
6. POST /api/v1/route-previews
7. reusable arbitrary-geometry RouteAnalyzer
8. /map route builder
9. draggable/reorderable waypoints
10. route visualization
11. distance/duration
12. potential exploration visualization
13. normalized error handling
14. latency metrics
15. automated test suite
16. manual validation report
```

---

# 44. Definition of Done

Stage 2 завершён, когда новый пользователь без заранее подготовленных route fixtures может:

```text
Open /map
 ↓
Set start
 ↓
Set destination
 ↓
Add/move/remove waypoint
 ↓
Receive pedestrian route
 ↓
See route distance and duration
 ↓
See which StreetSegments would be completed/partial
 ↓
Edit route and receive a correct updated preview
```

при этом:

- preview не сохраняется;
- user history не требуется;
- routing dataset совместим с current GeoDataVersion;
- stale route responses не перезаписывают новый UI state;
- Stage 1 geo semantics остаются неизменными.
