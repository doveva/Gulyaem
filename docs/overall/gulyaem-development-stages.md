# ГуляЕм — Development Stages

**Status:** Draft v0.1  
**Purpose:** общий контекст последовательности разработки проекта «ГуляЕм»  
**Scope:** от первого технического прототипа до recommendation/social capabilities  
**Initial platform:** mobile-first web application  
**Primary test city:** Санкт-Петербург  
**Architecture goal:** city-agnostic, privacy-first, visual-first validation

---

# 1. Назначение документа

Этот документ фиксирует **этапы развития проекта «ГуляЕм» и границы между ними**.

Он не заменяет:

- Product / Project Context;
- Technical Design;
- Geo & Exploration ADR;
- отдельные ADR;
- детальные требования конкретного Stage.

Вместо этого документ отвечает на вопросы:

- что разрабатываем сначала;
- какую гипотезу проверяет каждый этап;
- какие capabilities должны появиться;
- что намеренно откладывается;
- по каким критериям этап считается завершённым;
- в какой момент проект становится пригоден для внешнего тестирования.

Основной принцип roadmap:

> Каждый этап должен давать работающий end-to-end результат, который можно проверить визуально или пользовательским сценарием.

---

# 2. Общий принцип разработки

Для «ГуляЕм» frontend не является последним presentation layer поверх готового backend.

Ключевая часть продукта связана с визуальным восприятием:

- StreetSegment;
- исследованных улиц;
- маршрутов;
- exploration progress;
- новых участков;
- районов;
- Places;
- истории прогулок.

Поэтому разработка начинается сразу с **mobile-first web application**, которое одновременно используется:

1. как будущий пользовательский интерфейс;
2. как инструмент проверки geo-модели;
3. как playground для exploration algorithms;
4. как средство ранней product validation.

Общий подход:

```text
Domain capability
        +
Backend implementation
        +
Visual representation
        ↓
End-to-end validation
```

Не рекомендуется реализовывать большие части geo/exploration backend изолированно и откладывать визуальную проверку до поздних этапов.

---

# 3. Platform strategy

Первым клиентом является:

> **responsive mobile-first web application**

Причины:

- высокая скорость разработки;
- простой deployment;
- простой доступ тестировщиков;
- отсутствие App Store / Google Play distribution на раннем этапе;
- возможность быстро проверять map UX;
- возможность использовать UI для geo-spike;
- desktop остаётся удобным engineering/debugging interface.

При этом выбор web не означает окончательного отказа от:

- PWA;
- cross-platform mobile;
- native iOS;
- native Android.

Финальное решение по mobile platform принимается позднее, когда будет реализовываться настоящий GPS capture и появятся данные о требованиях к:

- background location;
- battery consumption;
- offline mode;
- camera integration;
- notifications;
- map performance.

---

# 4. Архитектурные принципы, действующие на всех этапах

Независимо от конкретного Stage должны сохраняться следующие инварианты.

## 4.1. Modular monolith first

Backend начинается как modular monolith.

Предварительные логические модули:

```text
Identity
Geo
Routing
Walks
Exploration
Places
Media
Sharing
Recommendations
Import / Tracking
```

Microservices не вводятся без эксплуатационной необходимости.

---

## 4.2. PostgreSQL + PostGIS

Основное хранилище:

```text
PostgreSQL
+
PostGIS
```

Spatial domain не должен раскладываться между несколькими базами без необходимости.

---

## 4.3. OSM — upstream, не domain model

OpenStreetMap используется как источник исходных geo data.

Нельзя использовать как persistent identity:

```text
OSM Way ID
OSM Node ID
routing engine edge ID
```

Внутренняя geo-модель принадлежит «ГуляЕм».

---

## 4.4. StreetSegment — единица exploration

Основная единица исследования города:

```text
StreetSegment
```

Не:

```text
Street
OSM Way
GPS point
routing edge
```

---

## 4.5. Raw GPS не является route truth

Источник истины:

```text
Final Normalized Route
```

GPS является только одним из потенциальных источников данных для построения такого маршрута.

---

## 4.6. Route ≠ Walk

`Route` описывает путь.

`Walk` описывает фактическое пользовательское событие / воспоминание.

Один Route потенциально может использоваться несколькими Walk.

---

## 4.7. Place ≠ Visit

Place представляет географическое место.

Visit представляет отдельное посещение пользователем этого места.

Оценка и комментарий относятся прежде всего к Visit.

---

## 4.8. Exploration rebuildable

`UserStreetProgress` является производным состоянием.

Источник истины:

```text
Completed Walk
+
Final Route Geometry
+
GeoDataVersion
```

Exploration state должен быть возможно полностью пересчитать.

---

## 4.9. Privacy by default

Приватными по умолчанию являются:

- Walk;
- Route;
- Visit;
- Photo;
- пользовательские Place;
- история перемещений.

Публичность появляется только через явную sharing capability.

---

# 5. Общая последовательность

```text
Stage 0 — Web Foundation
        ↓
Stage 1 — Geo Exploration Playground
        ↓
Stage 2 — Manual Route & Exploration Preview
        ↓
Stage 3 — Exploration Core
        ↓
Stage 4 — Accounts & Personal Exploration
        ↓
Stage 5 — Walk Memory
        ↓
Stage 6 — Real Walk Capture
        ↓
Stage 7 — Sharing
        ↓
Stage 8 — Recommendation Engine
        ↓
Stage 9 — Social & Discovery
```

Этапы являются последовательными с точки зрения product capabilities, но отдельные инфраструктурные работы могут начинаться раньше, если они не увеличивают неоправданно scope текущего Stage.

---

# 6. Stage 0 — Web Foundation

## Цель

Создать минимальную техническую оболочку проекта, внутри которой будут реализовываться и визуально проверяться следующие stages.

## Product result

Пользователь открывает приложение и видит интерактивную карту.

Карта является главным canvas приложения.

## Web scope

Реализовать:

- mobile-first responsive application;
- `/map` как основной screen;
- dark map style;
- базовый application shell;
- mobile navigation;
- desktop-compatible layout;
- loading/error states;
- базовую client/backend communication.

Предварительная navigation:

```text
Карта | Прогулки | Места | Профиль
```

Часть разделов может первоначально быть placeholder.

## Backend scope

Минимально:

- приложение/API;
- health endpoints;
- конфигурация;
- DB connection;
- migrations;
- базовая module structure.

## Infrastructure

Минимальный development environment:

```text
Web
Backend
PostgreSQL + PostGIS
Map infrastructure
Reverse proxy
```

Желательна контейнеризация.

## Не входит

- полноценная geo import model;
- exploration;
- routing;
- Walk;
- GPS;
- Places;
- Photos;
- auth.

## Exit criteria

Stage считается завершённым, когда:

1. приложение запускается воспроизводимо;
2. карта работает на desktop и mobile viewport;
3. backend и PostGIS доступны приложению;
4. возможно добавлять произвольные geo overlays;
5. создана базовая структура для дальнейших vertical slices.

---

# 7. Stage 1 — Geo Exploration Playground

## Цель

Проверить фундаментальную geo-модель продукта **до реализации пользовательского exploration loop**.

Главный вопрос Stage:

> Выглядит ли наша модель StreetSegment естественным представлением исследуемого города?

---

## Geo pipeline

Реализовать prototype:

```text
OpenStreetMap
      ↓
Import
      ↓
Normalization
      ↓
Pedestrian graph
      ↓
StreetSegment generation
      ↓
Walkability classification
      ↓
GeoDataVersion
```

---

## Основные сущности

Минимально:

```text
City
District
GeoDataVersion
Street
StreetSegment
```

---

## Walkability

Каждый edge / segment должен получить внутреннюю semantics:

```text
EXPLORE
ROUTABLE_ONLY
IGNORE
```

OSM tags не должны распространяться как доменная модель по всему приложению.

---

## Web playground

Карта должна позволять визуализировать:

- StreetSegment;
- segment boundaries;
- segment length;
- EXPLORE;
- ROUTABLE_ONLY;
- IGNORE;
- District boundaries;
- тестовые маршруты;
- coverage;
- metadata выбранного segment.

Например debug mode:

```text
/map?debug=geo
```

может предоставлять engineering controls, отсутствующие в обычном product UI.

---

## Test areas

Минимум три разных типа среды Санкт-Петербурга:

### Dense Center

- плотная street grid;
- короткие кварталы;
- много пересечений.

### Regular Urban District

- обычная городская сеть;
- жилые улицы;
- дворы;
- pedestrian connections.

### Park + Residential

- park footways;
- irregular paths;
- service roads;
- residential streets.

---

## Routing engine spike

В рамках Stage сравниваются основные кандидаты:

```text
Valhalla
GraphHopper
OSRM
```

Сравниваются как минимум:

- pedestrian routing;
- geometry quality;
- self-hosting;
- operational complexity;
- suitability для map matching.

Финальный routing engine фиксируется отдельным ADR.

---

## Stage должен определить

### Segmentation

- какие topology nodes создают split;
- нужна ли максимальная длина segment;
- нужны ли дополнительные artificial splits.

### Walkability

- какие пути EXPLORE;
- какие ROUTABLE_ONLY;
- какие IGNORE;
- правила для дворов;
- service roads;
- park paths.

### Coverage

Подготовить параметры для дальнейшей проверки:

- coverage ratio;
- min coverage;
- max coverage;
- необходимость partial coverage.

Не обязательно финализировать значения без реальных route experiments.

---

## Exit criteria

Stage завершён, если:

1. geo import воспроизводим;
2. построен собственный pedestrian graph;
3. StreetSegment отображаются на карте;
4. classification визуально проверена;
5. протестированы разные типы городской среды;
6. нет очевидно неприемлемой fragment/segment explosion;
7. routing engine выбран либо shortlist существенно сокращён;
8. зафиксированы ADR по основным результатам spike.

---

# 8. Stage 2 — Manual Route & Exploration Preview

## Цель

Получить первую пользовательскую capability, работающую поверх реального street graph.

Главный сценарий:

```text
Map
 ↓
+ Прогулка
 ↓
Waypoints
 ↓
Pedestrian route
 ↓
StreetSegment matching
 ↓
Exploration preview
```

---

## Manual Route Builder

Пользователь должен иметь возможность:

- выбрать старт;
- добавить destination;
- добавить промежуточные waypoint;
- удалить waypoint;
- переместить waypoint;
- перестроить route.

Пример:

```text
A → B → C → D
```

---

## Backend pipeline

```text
Waypoint[]
    ↓
Routing Engine
    ↓
Route Geometry
    ↓
Segment Matching
    ↓
Exploration Preview
```

---

## Preview

Пользователь видит:

- route geometry;
- distance;
- approximate duration;
- previously explored parts;
- new parts;
- процент нового маршрута.

Пример:

```text
7.3 km
≈ 1 h 40 min
3.1 km new streets · 42%
```

---

## Domain scope

Появляются или начинают использоваться:

```text
RoutePreview
RouteSegmentMatch
```

Постоянный Walk пока необязателен.

---

## Performance target

Route editing должен ощущаться интерактивным.

Engineering target:

```text
route preview p95 ≈ 1–2 sec
```

на типичном пользовательском маршруте.

---

## Не входит

- active Walk lifecycle;
- persistent exploration progress;
- GPS;
- Places;
- Photos;
- sharing.

---

## Exit criteria

Пользователь может построить реальный pedestrian route и визуально понять:

> какие части прогулки уже исследованы, а какие будут новыми.

---

# 9. Stage 3 — Exploration Core

## Цель

Реализовать главный product loop «ГуляЕм»:

> прогулка изменяет персональную карту исследованного города.

Это первый полный domain vertical slice.

---

## End-to-end flow

```text
Map
 ↓
Manual Route
 ↓
Start Walk
 ↓
Active Walk
 ↓
Finish
 ↓
Route Review
 ↓
Complete
 ↓
Exploration Delta
 ↓
Walk Summary
 ↓
Updated Exploration Map
```

---

## Domain model

Минимально:

```text
Route
RouteSegmentMatch
Walk
UserStreetSegmentProgress
ExplorationDelta
WalkDistrictDelta
```

---

## Walk lifecycle

```text
DRAFT
  ↓
ACTIVE
  ↓
REVIEW
  ↓
COMPLETED
```

Дополнительно:

```text
CANCELLED
```

---

## Route Review

Route Review является обязательной domain capability.

Пользователь должен иметь возможность до completion:

- подтвердить маршрут;
- удалить ошибочный участок;
- добавить пропущенный участок;
- перестроить часть маршрута.

Даже если первоначально маршрут был построен вручную, архитектурно Review необходим для будущего GPS pipeline.

---

## Exploration calculation

После подтверждения маршрута:

```text
Final Route
    ↓
Segment Matching
    ↓
Coverage Threshold
    ↓
New / Existing Segments
    ↓
User Progress
    ↓
Exploration Delta
```

---

## District progress

MVP formula:

```text
explored walkable length
/
total explorable walkable length
```

Progress считается по длине explorable segments, а не количеству Street.

---

## Walk Summary

Summary является главным reward screen.

Например:

```text
Новая часть города открыта

+3.1 km новых улиц
7.3 km прогулка

Центральный район
31% → 34%

17 новых сегментов
```

На карте должны визуально отличаться:

- ранее explored;
- новые segments этой прогулки;
- remaining unexplored.

---

## Correctness requirements

Walk completion должна быть:

- транзакционно устойчивой;
- идемпотентной;
- повторяемой после network retry.

Повторный completion не должен увеличивать progress дважды.

---

## Exit criteria

Stage завершён, когда:

1. пользователь строит Walk;
2. проходит lifecycle до COMPLETED;
3. получает ExplorationDelta;
4. районный progress изменяется;
5. новые StreetSegment остаются на карте после перезагрузки;
6. повторный completion безопасен;
7. progress можно пересчитать из завершённых Walk;
8. reward screen визуально понятен.

---

# 10. Milestone — Exploration Prototype

После Stage 3 технически доказан основной механизм продукта:

```text
StreetSegment
→ Route
→ Walk
→ Exploration
→ Map
```

На этом milestone необходимо отдельно оценить:

> вызывает ли визуальное исследование города желание совершить следующую прогулку.

Если core mechanic не воспринимается естественно, дальнейшее расширение продукта не должно маскировать проблему дополнительными функциями.

---

# 11. Stage 4 — Accounts & Personal Exploration

## Цель

Превратить работающий exploration prototype в персональный продукт, пригодный для первых внешних пользователей.

---

## Identity

Добавляются:

- registration;
- login;
- logout;
- session / refresh management;
- authorization.

Конкретный identity provider определяется отдельным ADR.

---

## Ownership

Проверяется ownership всех персональных сущностей:

```text
Route
Walk
UserStreetProgress
```

Позднее тот же механизм расширяется на:

```text
Visit
Photo
User Place
Share
```

---

## Personal map

Каждый пользователь должен видеть только собственный exploration state.

---

## History

Добавляется минимальный список:

```text
Walk[]
```

с возможностью открыть завершённую прогулку.

---

## Basic profile

Минимальный профиль:

- account information;
- current / selected city;
- базовая exploration statistics.

---

## Exit criteria

Новый пользователь способен самостоятельно:

```text
Register
 ↓
Open map
 ↓
Create walk
 ↓
Complete walk
 ↓
See exploration progress
 ↓
Open walk history
 ↓
Return and continue exploring
```

---

# 12. Milestone — Exploration Alpha

После Stage 4 продукт можно отдавать ограниченной группе внешних тестировщиков для проверки основного exploration loop.

Основные вопросы Alpha:

- понятна ли карта;
- понятна ли разница explored/unexplored;
- понятно ли, почему segment засчитан;
- интересно ли увеличивать процент района;
- понятен ли route builder;
- хочется ли возвращаться.

---

# 13. Stage 5 — Walk Memory

## Цель

Добавить вторую основную причину пользоваться продуктом:

> сохранить память о прогулке.

Exploration отвечает на:

> Что я ещё не исследовал?

Memory отвечает на:

> Где я был и что происходило во время прогулки?

---

## Domain additions

```text
Place
Visit
Photo
Walk Comment
```

---

## Place

Поддерживаются:

```text
EXTERNAL
USER
```

Пользовательские Place могут быть приватными.

---

## Visit

Place может иметь множество Visits.

```text
Place
 ├── Visit 1
 ├── Visit 2
 └── Visit 3
```

Visit может содержать:

- date;
- rating;
- comment;
- photos;
- link to Walk.

---

## Photos

Photo может относиться:

- ко всему Walk;
- к Visit;
- к географической точке.

Файлы хранятся в S3-compatible object storage.

Metadata хранится в PostgreSQL.

Предпочтительный upload flow:

```text
Client
 ↓
Backend
 ↓
Presigned URL
 ↓
Object Storage
 ↓
Metadata confirmation
```

---

## Walk Detail

Walk становится полноценной memory entity:

```text
Walk
├── Map / Route
├── Exploration Result
├── Places
├── Visits
├── Photos
└── Comment
```

---

## Places section

Пользователь может:

- просматривать сохранённые места;
- видеть историю посещений;
- видеть оценки разных Visits;
- открывать связанные Walk.

---

## Exit criteria

Пользователь может вернуться к старой прогулке и восстановить контекст:

- где гулял;
- какие места посещал;
- что думал о них;
- какие фотографии сделал;
- какую часть города тогда открыл.

---

# 14. Milestone — Personal MVP

После Stage 5 реализованы две основные ценности продукта:

```text
Exploration
+
Memory
```

Продукт уже реализует исходную формулировку:

> персональная карта мест, маршрутов и воспоминаний.

---

# 15. Stage 6 — Real Walk Capture

## Цель

Перейти от преимущественно планируемых/manual routes к записи реально пройденной прогулки.

Этот Stage является главным checkpoint для окончательного решения о mobile platform.

---

## Capture sources

Добавляются:

```text
GPS
GPX
External Route Import
```

---

## GPS pipeline

Критический инвариант:

```text
Raw GPS ≠ Exploration
```

Правильный pipeline:

```text
Raw GPS Track
      ↓
Noise Filtering
      ↓
Map Matching
      ↓
Candidate Normalized Route
      ↓
Confidence Analysis
      ↓
Route Review / Correction
      ↓
Final Route
      ↓
Exploration
```

---

## Raw data

Для GPS могут храниться:

```text
GpsTrackPoint
- timestamp
- lat
- lon
- accuracy
- speed?
- altitude?
```

Raw tracking хранится отдельно от final route.

---

## Confidence

Map matching должен позволять выделять сомнительные части маршрута.

Пользовательскому UI полезнее показывать:

> этот участок требует проверки

чем одну абстрактную GPS accuracy для всей прогулки.

---

## Route correction

Пользователь должен иметь возможность:

- удалить false GPS jumps;
- добавить пропущенный путь;
- изменить matched street;
- перестроить fragment.

---

## GPX

Импорт GPX должен проходить через тот же normalization pipeline.

```text
GPX
 ↓
Candidate Route
 ↓
Review
 ↓
Final Route
```

Он не должен создавать отдельную exploration semantics.

---

## Platform checkpoint

Во время Stage отдельно проверяются:

- browser background tracking;
- PWA capabilities;
- OS limitations;
- battery consumption;
- background suspension;
- offline capture;
- camera integration.

После spike принимается ADR:

```text
Web / PWA
vs
Cross-platform
vs
Native
```

Web frontend при любом результате может сохраниться как desktop client.

---

## Exit criteria

Реальную прогулку можно записать даже при шумных координатах, проверить и превратить в корректный normalized route.

---

# 16. Stage 7 — Sharing

## Цель

Позволить пользователю явно делиться отдельными результатами, не превращая personal domain в публичный.

---

## Shareable resources

Минимально:

- Walk;
- Route;
- Photos;
- Places associated with Walk.

---

## Share model

Sharing является отдельной capability:

```text
Private Resource
       ↓
Share Settings
       ↓
Public Projection
       ↓
Token / Link
```

---

## User controls

Пользователь выбирает:

- показывать ли route;
- показывать ли Places;
- показывать ли Photos;
- показывать ли comments.

---

## Endpoint privacy

Необходимо поддержать возможность скрыть:

- начало маршрута;
- конец маршрута.

Это снижает риск раскрытия места проживания пользователя.

---

## Photo privacy

При публичной публикации необходимо контролировать EXIF/location metadata.

---

## Revoke

Share link должен быть возможно отозвать.

---

## Не входит

- followers;
- friend feed;
- public profiles;
- likes;
- leaderboard;
- social graph.

---

## Exit criteria

Пользователь безопасно делится конкретной прогулкой с человеком, не публикуя остальные данные аккаунта.

---

# 17. Milestone — Public Beta Candidate

После Stage 7 сформирован достаточно полный продуктовый loop:

```text
Plan
 ↓
Walk
 ↓
Review
 ↓
Explore
 ↓
Remember
 ↓
Share
 ↓
Return
```

Это подходящая точка для расширения Beta-аудитории.

---

# 18. Stage 8 — Recommendation Engine

## Цель

Решить следующий пользовательский вопрос:

> Куда мне пойти сегодня?

Recommendation не заменяет routing и exploration core.

Он генерирует подходящие `CandidateRoute`.

---

## Inputs

Recommendation может использовать:

```text
UserStreetProgress
Walk History
Saved Places
Visit History
Start Location
Available Time
User Constraints
```

---

## Constraints

Примеры:

```text
duration <= 80 min
new_street_ratio >= 0.70
return_to_start = true
include_coffee_place = true
```

---

## Result

```text
CandidateRoute[]
```

Каждый кандидат затем проходит обычный pipeline:

```text
Candidate Route
      ↓
Route Preview
      ↓
Walk
      ↓
Exploration
```

---

## Initial implementation

Первый recommendation engine должен быть:

> heuristic / rule-based

а не ML.

Score потенциально учитывает:

- new street coverage;
- repeated streets penalty;
- distance;
- duration;
- route diversity;
- Place relevance;
- user preferences;
- walkability.

---

## Первые пользовательские сценарии

### New streets

> Хочу погулять час и увидеть максимум нового.

### Round trip

> Построй прогулку на 6 км с возвратом сюда.

### Place-aware

> Хочу гулять 1.5 часа и зайти за кофе.

### Continue exploration

> Покажи район рядом, который я почти не исследовал.

---

## Exit criteria

Пользователь может получить несколько разумных маршрутов без ручного выбора всех waypoints.

---

# 19. Stage 9 — Social & Discovery

## Цель

Добавить discovery через других пользователей после того, как подтверждена ценность personal-first продукта.

---

## Возможные capabilities

- public Walk;
- public Route;
- friends;
- shared Place;
- collections;
- сохранить чужой Route себе;
- discovery feed;
- public profile;
- тематические подборки.

---

## Architectural constraint

Social domain не должен автоматически делать personal entities публичными.

Используются отдельные public projections.

Например:

```text
Private Walk
     ↓
Publication
     ↓
Shared / Public Walk
```

---

## Не является приоритетом

Не требуется делать основой продукта:

- competitive leaderboards;
- aggressive streaks;
- engagement notifications;
- follower counts как основной reward.

---

# 20. Roadmap milestones

Укрупнённо развитие выглядит так:

```text
Stage 0
Web Foundation
      ↓
Stage 1
Geo Feasibility
      ↓
Stage 2
Route Feasibility
      ↓
Stage 3
Exploration Prototype
      ↓
Stage 4
Exploration Alpha
      ↓
Stage 5
Personal MVP
      ↓
Stage 6
Real-world Capture
      ↓
Stage 7
Public Beta Candidate
      ↓
Stage 8
Smart Recommendations
      ↓
Stage 9
Social Expansion
```

---

# 21. Product validation gates

Каждый крупный этап должен отвечать на конкретный вопрос.

## Gate A — после Stage 1

> Наша модель города визуально выглядит естественно?

Если нет — не имеет смысла строить поверх неё exploration.

---

## Gate B — после Stage 2

> Пользователю понятно, какие улицы маршрут позволит исследовать?

---

## Gate C — после Stage 3

> Сам процесс «открывания» карты приносит ощущение прогресса?

Это основной product risk проекта.

---

## Gate D — после Stage 4

> Пользователь хочет вернуться и продолжить исследование?

---

## Gate E — после Stage 5

> История прогулок имеет самостоятельную ценность как память?

---

## Gate F — после Stage 6

> Реальную прогулку можно надёжно восстановить несмотря на проблемы GPS?

---

## Gate G — после Stage 8

> Автоматическая рекомендация действительно снижает friction выбора следующей прогулки?

---

# 22. Что намеренно не делается заранее

До появления подтверждённой необходимости не вводятся:

- Kubernetes;
- microservices;
- Redis по умолчанию;
- ML recommendation;
- сложная social architecture;
- realtime infrastructure;
- полноценный offline maps stack;
- segment reconciliation между версиями OSM;
- сложная incremental exploration migration;
- отдельная analytics database.

Приоритет:

> сначала корректный domain и подтверждённый пользовательский сценарий, затем инфраструктурная оптимизация.

---

# 23. Cross-cutting backlog

Некоторые задачи не принадлежат одному Stage и развиваются постепенно.

## Observability

По мере появления capabilities добавляются:

- API latency;
- route preview latency;
- route matching latency;
- walk completion latency;
- DB latency;
- error rate;
- tracing.

Не логировать точную геолокацию пользователя без необходимости.

---

## Offline / bad network

Последовательно развиваются:

1. retries;
2. idempotent mutations;
3. local active Walk state;
4. offline mutation queue;
5. GPS buffering;
6. photo upload queue.

Полноценные offline maps являются отдельной будущей capability.

---

## GeoData lifecycle

На всех этапах сохраняется version awareness.

После обновления street graph ранняя стратегия:

```text
Completed Walk Geometry
       ↓
New GeoDataVersion
       ↓
Rematch
       ↓
Rebuild Exploration
```

Сложный segment lineage откладывается.

---

## Performance

Оптимизация делается по измерениям.

Основные области наблюдения:

- map rendering;
- route calculation;
- PostGIS matching;
- exploration tiles;
- photo delivery.

---

# 24. Требования к документации каждого Stage

Перед началом реализации конкретного Stage создаётся отдельный документ:

```text
stage-N-requirements.md
```

Он должен содержать минимум:

1. Goal.
2. User scenarios.
3. Functional requirements.
4. Non-functional requirements.
5. Domain changes.
6. API surface.
7. UI states.
8. Persistence changes.
9. Error scenarios.
10. Observability.
11. Explicit non-goals.
12. Acceptance criteria.
13. Open questions.
14. Required ADR.
15. Test plan.

После реализации Stage документ обновляется фактическими решениями.

---

# 25. Принцип изменения roadmap

Нумерация Stage отражает последовательность capabilities, но не является неизменным контрактом.

Допустимы промежуточные stages:

```text
Stage 3.1
Stage 3.2
```

если во время разработки появляется значимый, но ограниченный scope.

Новый Stage следует выделять, когда задача:

- имеет самостоятельную product/technical цель;
- требует отдельной validation;
- существенно увеличивает scope;
- создаёт новую domain capability;
- требует отдельного архитектурного решения.

Мелкие исправления не должны становиться отдельными stages.

---

# 26. Краткая формулировка roadmap

Разработка «ГуляЕм» начинается не с GPS tracker и не с социальной сети.

Сначала строится визуальная модель исследуемого города:

```text
OSM
↓
StreetSegment
↓
Map
↓
Route
↓
Walk
↓
Exploration
```

После подтверждения этой механики продукт последовательно расширяется:

```text
Exploration
+
Memory
+
Capture
+
Sharing
+
Recommendations
+
Social
```

Главный принцип:

> Каждый следующий слой усиливает персональную систему исследования города, но не заменяет её ядро.
