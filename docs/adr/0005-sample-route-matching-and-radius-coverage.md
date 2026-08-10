# ADR-0005: Последовательный map matching и радиусное exploration coverage

- **Status:** Accepted
- **Date:** 2026-08-10
- **Owners:** команда «ГуляЕм»
- **Related Stage:** Stage 1.5

## Context

После визуальной проверки `StreetSegment` нужно измерить, насколько реальная прогулка исследует
окружающую сеть. Точное следование GPS-линии одной стороне улицы не должно требовать повторного
прохода по другой стороне или захода в каждый двор. При этом прототип не должен создавать
production `Walk`, `Route` или пользовательский прогресс.

## Decision

1. Пять immutable GeoJSON routes проверяют три среды: плотный центр, жилой Калининский коридор от
   Академической до Гражданского проспекта и Сосновку с жилыми краями. Исходный OSM хранится только
   в объединённом PBF. Fixture pin-ит checksum и normalization version, но mismatch даёт warning,
   а не отказ анализа.
2. Matching выполняется последовательно по samples через 5 м. PostGIS/GiST выбирает кандидатов в
   12 м, Go оценивает distance, ненаправленный direction (допуск 55°) и continuity. `EXPLORE` —
   только tie-breaker; `ROUTABLE_ONLY` разрешён как connector; `IGNORE` не участвует.
3. Неуверенные участки не притягиваются принудительно. API возвращает
   `UNMATCHED_NO_CANDIDATE`, `UNMATCHED_DIRECTION`, `UNMATCHED_DISCONTINUITY` или
   `UNMATCHED_AMBIGUOUS`, а matched fragments содержат компоненты confidence.
4. Exploration buffer строится вокруг успешно нормализованной route geometry. PostGIS точно
   пересекает каждый `EXPLORE` segment с объединённым buffer; direct intervals предварительно
   объединяются по measure сегмента. Bridge/tunnel/indoor/level signatures не покрывают другой
   grade. `ROUTABLE_ONLY` никогда не увеличивает coverage.
5. Completion threshold:

   ```text
   required_m = min(segment_length_m,
     clamp(segment_length_m * coverage_ratio, min_required_m, max_required_m))
   ```

   Нулевое покрытие — `NOT_COVERED`, ненулевое ниже threshold — `PARTIAL`, достигшее threshold —
   `COMPLETED`. Provenance отдельно показывает `DIRECT`, `RADIUS` и `DIRECT_AND_RADIUS`.
6. Экспериментальные profiles: Strict `10 м / 0.8 / 20–120 м`, Balanced
   `20 м / 0.6 / 15–80 м`, Generous `35 м / 0.4 / 10–50 м`; custom radius ограничен 5–50 м.
   Analysis context независимо фиксирован на 75 м.
7. Метрики различают geometric covered length, completed network length, explorable context,
   completed network ratio, matched ratio и unmatched route length. Результаты вычисляются на
   запрос и не накапливаются между routes.

## Consequences

- Можно сравнивать profiles на одной route без изменения данных и миграций.
- Радиус соответствует продуктовой семантике исследования, а partial сохраняет информацию до
  появления пользовательского cumulative progress.
- Sequential matcher остаётся измеримым прототипом; если реальные GPS traces покажут проблемы,
  его можно заменить без изменения fixtures и response semantics.
- Routing engines остаются отдельным Stage 1.6.

## Validation

- все пять routes анализируются на одной current `GeoDataVersion`;
- normal routes дают высокий matched ratio, deliberate courtyard fixture сохраняет unmatched;
- strict/balanced/generous меняют coverage, не меняя matching;
- underground/indoor grade не покрывается surface route;
- API возвращает использованные параметры и version metadata;
- frontend различает source, normalized, unmatched, connector и три coverage statuses.

## Links

- [`Stage 1 requirements`](../stage%201/stage-1-requirements.md)
- [`Stage 1 implementation plan`](../stage%201/implementation-plan.md)
- [`Stage 1 validation fixture`](../../data/test-areas/spb-stage1-validation/README.md)
- [`Sample routes`](../../data/sample-routes/spb-stage1/README.md)
