# Stage 2 — Decision Log and Deferred Follow-ups

На freeze 2026-08-12 у Stage 2 нет открытых архитектурных решений, которые блокируют Stage 3.
Ниже зафиксирован результат каждого прежнего открытого вопроса: восемь решений закрыты Stage 2
evidence, два follow-up явно отложены и не маскируются как принятые.

## OD-01 Waypoint UX — resolved

Stage 2 принимает следующий product flow:

- первая точка становится стартом;
- вторая — финишем и запускает preview;
- промежуточная точка добавляется явным действием `+ Точка`;
- markers перемещаются с recalculation только на `dragend`;
- промежуточные точки удаляются и меняются местами явными controls.

Desktop и viewport `390 × 844` подтверждены Playwright. Проверка на физическом мобильном
устройстве остаётся AC-84, но не меняет routing architecture.

## OD-02 Snapped waypoint visualization — resolved for Stage 2

Product marker остаётся в координате пользовательского ввода. Stage 2 не рисует отдельный
«resolved» marker: текущий Valhalla contract не предоставляет надёжную engine-neutral координату
snap, а routing engine IDs и internal edge details не являются product identity.

Отдельная snap visualization допускается позднее только после полевой проверки систематического
видимого offset и расширения routing port явным engine-neutral полем.

## OD-03 Low-match warning threshold — resolved

Warning boundary зафиксирован как:

```text
routeMatchedRatio < 0.95
```

Stage 1 normal fixtures дают приблизительно 99–100%, а intentionally ambiguous
`konyushennaya-capella-moyka` — `0.914470569` (91,447%). Поэтому известный degraded case получает
warning, а значение ровно `0.95` — нет. Это именованная application diagnostic constant, а не
deployment knob или пользовательская настройка.

## OD-04 Product exploration wording — resolved

Product UI использует формулировку «Потенциально исследуется» и различает `COMPLETED`/`PARTIAL`.
Stage 2 не использует `newMeters`, `alreadyExplored`, `newStreetRatio`, `districtProgress` или
другие понятия пользовательской истории.

## OD-05 Product metrics — resolved

В summary отображаются distance, duration, количество потенциально completed и partial segments,
а также route matched ratio. API дополнительно сохраняет диагностические lengths и unmatched
fragments. Доля сети в 225-метровом analysis context не представляется как «новизна маршрута».

## OD-06 Route-analysis performance — resolved

На 30 warmed dense-center запросах total preview составил `p50 1105 ms / p95 1273 ms`, из них
Valhalla `p95 4 ms`, RouteAnalyzer `p95 1268 ms`. Stage 2 target 1–2 секунды выполнен; PostGIS
analysis зафиксирован как доминирующая часть latency. Caching и изменение coverage semantics для
Stage 2 не требуются.

## OD-07 HTTP compression — resolved

HTTP gzip включён. Три representative route-preview payload измерены как `131/27 KB`,
`294/72 KB` и `192/49 KB` raw/gzip. Вместе с Stage 1 bbox measurements это не даёт evidence для
vector tiles. Решение закреплено в [`ADR-0009`](../adr/0009-bbox-geojson-stage2.md).

## OD-08 Background network at overview zoom — resolved

Route и potential coverage остаются видимыми, когда bbox `StreetSegment` network не загружена или
вернула controlled error. UI явно сообщает, что для фоновой сети нужен zoom-in. Такой overview
принят для Stage 2 в [`ADR-0009`](../adr/0009-bbox-geojson-stage2.md).

## OD-09 Stage 3 materialization boundary — deferred to Stage 3

Stage 2 намеренно остаётся stateless и не выбирает между сохранением preview geometry и повторным
routing при создании будущего persistent `Route`/`Walk`.

Stage 3 получает строгие входные гарантии для этого решения:

- preview содержит конкретный `GeoDataVersion`;
- routing graph metadata привязаны к реальному graph artifact;
- RouteAnalyzer pin-ит тот же `geo_data_version_id` для всех spatial queries;
- waypoints, route geometry и potential coverage не создают persistent entity в Stage 2.

Stage 3 должен отдельным ADR решить recompute-versus-persist с учётом времени между preview и
start/save, reproducibility и UX. Это не незакрытое решение Stage 2.

## OD-10 ADR-0009 status — resolved

[`ADR-0009`](../adr/0009-bbox-geojson-stage2.md) имеет status `Accepted` с 2026-08-12. Основание:
Stage 1 bbox p95, Stage 2 gzip/payload measurements и route-builder behavior при отсутствии
фоновой сети. Критерии пересмотра записаны в ADR.

## Remaining validation follow-ups

Не являются архитектурными blockers, но остаются честно неотмеченными в acceptance checklist:

- AC-84 — interaction review на физическом мобильном устройстве;
- AC-86 — unaided comprehension review формулировки и визуализации potential coverage.
