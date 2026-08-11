# Stage 1.7 — Validation report

- **Date:** 2026-08-11
- **Fixture:** `spb-stage1-validation`
- **Automated status:** Passed
- **Visual engineering review:** Passed with known limitations
- **Final reviewer sign-off:** Pending

## Reproduction

```bash
make stage1-e2e
make stage1-validate
```

Playwright проверяет загрузку реального playground, layer toggle, filter-driven reload, segment
selection/inspector с typed normalization, смену viewport, Balanced analysis, routing overlay, API
error и empty state.
Validation runner выполняет 30 warmed bbox requests в каждой среде, проверяет import invariants и
прогоняет пять routes через три coverage profiles.

## Automated results

- current version `READY`, checksum fixture совпадает, normalization `stage1-segments-v1`;
- 28 957 segments, `invalid_geometries=0`, `zero_length_segments=0`;
- 3/3 Playwright scenarios passed;
- все representative bbox p95 ниже target 500 мс;
- все пять routes и 15 profile analyses выполнены;
- четыре обычных routes дают 99,1–100% matching, intentional ambiguous route — 91,4%.
- synthetic mixed-grade PostGIS regression подтверждает: parallel tunnel около surface остаётся
  `NOT_COVERED`, а локальный tunnel fragment покрывает tunnel segment без двойного подсчёта buffers.

| Среда | Features | GeoJSON | p95 | Наблюдение |
|---|---:|---:|---:|---|
| Dense center | 6 558 | 2,76 MB | 92,8 мс | Плотная сеть читаема; известны отдельные metro/underpass artifacts |
| Akademicheskaya | 5 462 | 2,30 MB | 51,8 мс | Жилые кварталы, дворы и connectors визуально различимы |
| Sosnovka | 1 676 | 0,71 MB | 36,9 мс | Park paths, public tracks и residential edge образуют связную сеть |

## Visual review

Dense center подтверждает ранее принятую topology: crossings/connectors отделены от explorable
network, дворы не превращены целиком в обязательное покрытие. Редкие подземные/metro artifacts
остаются известным ограничением.

В районе Академической после приближения отображались 4 172 segments; явного fragment explosion,
неестественных пропусков или причины менять service semantics не найдено. Полный corridor overview
защищён лимитом 10 000 features.

В Сосновке после приближения отображались 743 segments. Park network выглядит связной, а
Balanced analysis дал 100% matched, 17,4% completed context network и 0 м unmatched. Полный park
area overview также требует zoom-in.

## Freeze candidates

- topology, ненаправленная identity, `max_segment_length_m=0` и WalkabilityProfile v1:
  [`ADR-0007`](../adr/0007-street-segment-stage1-freeze.md);
- Balanced `50 м / 0,6 / 15–80 м`, profiles `35/50/100 м`, custom `5–200 м` и сохранение `PARTIAL`:
  [`ADR-0008`](../adr/0008-coverage-parameters-stage1-freeze.md);
- bbox + GeoJSON с явным overview limitation:
  [`ADR-0009`](../adr/0009-bbox-geojson-stage2.md), ожидает reviewer sign-off.

## Known limitations

- метро, indoor и сложные grade-separated переходы требуют реальных GPS traces перед tuning;
- полный Калининский corridor и полный bbox Сосновки превышают feature limit;
- результаты latency получены локально на Docker Desktop arm64 и не являются production SLA;
- расширенный 225 м analysis context дал 0,8–2,3 с на route analysis и требует performance review
  перед production sizing;
- cumulative user progress, production Walk/GPS и reconciliation версий остаются Stage 2+.

## Exit decision

Автоматические критерии и engineering review позволяют перейти к Stage 2 после подтверждения
reviewer двух пунктов: допустимости zoom-in для полного corridor/park overview и общего визуального
принятия трёх зон.
