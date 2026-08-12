# ADR-0007: Freeze topology и WalkabilityProfile после Stage 1

- **Status:** Accepted
- **Date:** 2026-08-10
- **Owners:** команда «ГуляЕм»
- **Related Stage:** Stage 1.7
- **Supersedes:** экспериментальный статус параметров в ADR-0002

## Context

Правила Stage 1.3 были намеренно предварительными до проверки плотного центра, регулярной жилой
сети и парка. Stage 1.4–1.7 добавили визуализацию, районы, реальные маршруты, coverage, routing
comparison и воспроизводимый validation report. Перед Stage 2 нужно решить, требуется ли менять
identity/fragmentation/classification `StreetSegment`.

## Decision

1. Topology segmentation из ADR-0002 принимается как основа Stage 2: значимые graph nodes и смена
   semantics создают split; произвольная нарезка по длине не используется.
2. `max_segment_length_m=0` остаётся default. Пороги `< 5 м` и `> 500 м` остаются только
   диагностикой, а не причиной split или reject.
3. `StreetSegment` остаётся ненаправленной domain identity. Directional traversal принадлежит
   routing graph.
4. WalkabilityProfile v1 принимается без изменения. В частности:
   - публичные `footway/path/track`, streets и steps — `EXPLORE`;
   - `highway=service + service=alley/track` остаются `EXPLORE` с reason
     `public_service_alley/public_service_track`;
   - остальные service access и crossings/connectors — `ROUTABLE_ONLY`;
   - prohibited/private, unsupported highway и indoor/corridor — `IGNORE`.
5. `street_id` остаётся nullable; автоматическая нормализация названий улиц не добавляется до
   появления Stage 2 use case.
6. Raw OSM identity остаётся debug provenance и не становится product identity.

## Evidence

Полный import создал 28 957 положительных segments: 14 896 `EXPLORE`, 10 035
`ROUTABLE_ONLY`, 4 026 `IGNORE`, без invalid и zero-length geometry. Диагностика нашла 4 362
segments короче 5 м и 13 длиннее 500 м; визуальная проверка не показала fragment explosion или
основания автоматически резать длинные park/linear paths.

| Среда | Segments | EXPLORE | ROUTABLE_ONLY | IGNORE | Median / P95 |
|---|---:|---:|---:|---:|---:|
| Dense center | 6 558 | 2 649 | 2 338 | 1 571 | 11,5 / 86,0 м |
| Akademicheskaya | 5 462 | 2 780 | 2 162 | 520 | 18,4 / 96,3 м |
| Sosnovka | 1 676 | 1 344 | 225 | 107 | 33,0 / 157,7 м |

В Сосновке отдельно присутствуют публичные `track` segments; park paths и residential edges
образуют ожидаемую сеть. В центре ранее замечены отдельные случаи, где маршрут может задевать
подземный переход или вход метро. Для Stage 2 это принимается как известное ограничение: оно не
оправдывает изменение общей topology/classification до появления реальных GPS traces.

## Consequences

- Stage 2 использует стабильную `StreetSegment` identity и текущую normalization version
  `stage1-segments-v1`.
- Длинные и короткие segments остаются наблюдаемыми, но не исправляются без конкретной ошибки.
- Отдельное моделирование метро, indoor и сложных grade-separated переходов остаётся техническим
  долгом.
- Изменение правил требует новой normalization version и ADR с migration/reconciliation plan.

## Links

- [`ADR-0002`](0002-street-segment-topology-and-walkability.md)
- [`Stage 1 validation report`](../stage%201/validation-report.md)
- [`Machine-readable evidence`](../../data/validation/spb-stage1/report.json)

