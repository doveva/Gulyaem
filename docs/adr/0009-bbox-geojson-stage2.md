# ADR-0009: Bbox + GeoJSON как начальная map delivery Stage 2

- **Status:** Accepted
- **Date:** 2026-08-10
- **Updated:** 2026-08-12
- **Owners:** команда «ГуляЕм»
- **Related Stage:** Stage 1.7, Stage 2

## Context

Stage 1 использует bounded GeoJSON вместо city-wide graph и vector tiles. До freeze Stage 2 нужно
подтвердить, что representative street-level viewports укладываются в target `p95 < 500 мс`,
защитные ограничения не скрывают частичные данные, а route-builder остаётся понятным без полной
фоновой сети на overview zoom.

## Decision

1. Сохранить bbox + GeoJSON для Stage 2 с текущими ограничениями `25 км²`, `10 000 features` и
   zoom `>= 13`.
2. Превышение limit продолжает возвращать явный `422`, а не truncated FeatureCollection.
3. Overview route показывает route/coverage без всей фоновой `StreetSegment` network; для
   инспекции сети пользователь приближает карту. Ошибка фоновой сети не скрывает построенный
   маршрут и сопровождается явным сообщением.
4. GeoJSON responses поддерживают HTTP gzip.
5. Не вводить vector tiles до измеренного нарушения target или появления обязательного city-wide
   overview use case.

## Evidence

Stage 1: 30 warmed requests на каждый representative viewport.

| Viewport | Features | Raw GeoJSON | Warm p50 | Warm p95 |
|---|---:|---:|---:|---:|
| Dense center | 6 558 | 2,76 MB | 52,2 мс | 65,2 мс |
| Akademicheskaya | 5 462 | 2,30 MB | 46,1 мс | 86,5 мс |
| Sosnovka | 1 676 | 0,71 MB | 20,9 мс | 37,3 мс |

Все viewports ниже target 500 мс и визуально интерактивны. Полные source-area bbox Калининского
коридора и Сосновки превышают 10 000 features и получают ожидаемый
`422 feature_limit_exceeded`; после одного zoom-in отображаются соответственно 4 172 и 743
segments в проверенном browser viewport.

Stage 2 измерил product route-preview payload на трёх реальных маршрутах: `131/27 KB`,
`294/72 KB` и `192/49 KB` raw/gzip. Playwright подтвердил, что route-builder сохраняет маршрут при
ошибке или отсутствии фоновой bbox-сети. Эти данные не дают основания вводить vector tiles.

## Reconsider when

- representative warmed p95 достигает 500 мс;
- обязательный viewport регулярно превышает feature limit;
- payload/rendering заметно блокирует целевое мобильное устройство;
- продукту требуется одновременная интерактивная city-wide `StreetSegment` network.

## Consequences

- Stage 2 не получает преждевременную tile infrastructure.
- Route overview и detailed network inspection остаются разными zoom states.
- Bbox/feature limits и отсутствие фоновой сети являются явными состояниями, а не silent partial
  delivery.
- Решение пересматривается по field measurements, а не как speculative optimization.

## Links

- [`ADR-0003`](0003-geo-playground-bbox-api.md)
- [`Stage 1 validation report`](../stage%201/validation-report.md)
- [`Stage 2 validation report`](../stage%202/validation-report.md)
- [`Machine-readable Stage 1 evidence`](../../data/validation/spb-stage1/report.json)
