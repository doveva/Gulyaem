# ADR-0009: Bbox + GeoJSON как начальная map delivery Stage 2

- **Status:** Proposed — ожидает финального reviewer sign-off Stage 1.7
- **Date:** 2026-08-10
- **Owners:** команда «ГуляЕм»
- **Related Stage:** Stage 1.7

## Context

Stage 1 использует bounded GeoJSON вместо city-wide graph и vector tiles. До Stage 2 нужно
подтвердить, что representative street-level viewports укладываются в target `p95 < 500 мс`, а
защитные ограничения не скрывают частичные данные.

## Proposed decision

1. Сохранить bbox + GeoJSON для Stage 2 с текущими ограничениями `25 км²`, `10 000 features` и
   zoom `>= 13`.
2. Превышение limit продолжает возвращать явный `422`, а не truncated FeatureCollection.
3. Overview route может показывать route/coverage без всей фоновой StreetSegment network; для
   инспекции сети пользователь приближает карту на один уровень.
4. Не вводить vector tiles до измеренного нарушения target или появления обязательного city-wide
   overview use case.

## Evidence

30 warmed requests на каждый representative viewport:

| Viewport | Features | Raw GeoJSON | Warm p50 | Warm p95 |
|---|---:|---:|---:|---:|
| Dense center | 6 558 | 2,76 MB | 52,2 мс | 65,2 мс |
| Akademicheskaya | 5 462 | 2,30 MB | 46,1 мс | 86,5 мс |
| Sosnovka | 1 676 | 0,71 MB | 20,9 мс | 37,3 мс |

Все viewports ниже target 500 мс и визуально интерактивны. Полные source-area bbox Калининского
коридора и Сосновки превышают 10 000 features и получают ожидаемый
`422 feature_limit_exceeded`; после одного zoom-in отображаются соответственно 4 172 и 743
segments в проверенном browser viewport. При ошибке UI очищает прежние geometry/statistics, чтобы
не показывать stale данные.

## Reconsider when

- representative warmed p95 достигает 500 мс;
- обязательный viewport регулярно превышает feature limit;
- payload/rendering заметно блокирует целевое мобильное устройство;
- продукту требуется одновременная интерактивная city-wide StreetSegment network.

## Consequences

- Stage 2 не получает преждевременную tile infrastructure.
- Route overview и detailed network inspection остаются разными zoom states.
- Ограничение полного corridor/park overview явно принимается либо этот ADR пересматривается до
  завершения Stage 1.7.

## Links

- [`ADR-0003`](0003-geo-playground-bbox-api.md)
- [`Stage 1 validation report`](../stage%201/validation-report.md)
- [`Machine-readable evidence`](../../data/validation/spb-stage1/report.json)

