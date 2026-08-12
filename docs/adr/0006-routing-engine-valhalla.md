# ADR-0006: Valhalla как routing engine для Stage 2

- **Status:** Accepted
- **Date:** 2026-08-10
- **Updated:** 2026-08-12
- **Owners:** команда «ГуляЕм»
- **Related Stage:** Stage 1.6, Stage 2

## Context

Продукту нужен self-hosted pedestrian routing engine, но внутренняя модель исследования города
остаётся за `StreetSegment`. Чтобы не выбирать движок по общему впечатлению, Valhalla 3.7.0,
GraphHopper 11.0 и OSRM 26.7.3 сравнивались на одном PBF и пяти маршрутах Stage 1.5. Критерии:
качество pedestrian route — 40%, map matching — 20%, эксплуатация — 20%, ресурсы — 15%,
сложность интеграции — 5%.

## Decision

1. Для интеграции в Stage 2 выбирается **Valhalla 3.7.0** с базовым pedestrian profile.
2. OSRM остаётся воспроизводимым performance baseline и возможным fallback, но не основным
   движком. GraphHopper исключается из активного shortlist до появления нового требования или
   профиля, который изменит результаты.
3. Идентификаторы рёбер routing engine считаются диагностическими. Маршрут после routing или map
   matching всё равно сопоставляется с конкретной версией собственных `StreetSegment`.
4. Routing graph и `GeoDataVersion` обязаны происходить из одного source dataset. Метаданные graph
   создаются после успешного запуска Valhalla, содержат checksum исходного PBF и checksum реального
   graph artifact; API читает этот файл, проверяет artifact и сравнивает source checksum с текущей
   READY geo version.
5. Тюнинг pedestrian profile, production sizing и GPS-noise параметры не входят в это решение и
   должны проверяться отдельными экспериментами на более крупных данных.

## Evidence

Все измерения Stage 1 получены одним runner на checksum-pinned `spb-stage1-validation`, с 30
прогретыми запросами на каждый из пяти маршрутов. Геометрическое сходство считается симметрично в
коридоре 20 м; `StreetSegment match` использует matcher Stage 1.5.

| Метрика | Valhalla | GraphHopper | OSRM |
|---|---:|---:|---:|
| Успешные routes | 5/5 | 5/5 | 5/5 |
| Candidate внутри reference corridor | 76,2% | 75,2% | 75,3% |
| Reference внутри candidate corridor | 70,2% | 68,6% | 68,6% |
| Средний `StreetSegment match` | 100% | 100% | 99,96% |
| Median warm p50 | 2,06 ms | 1,52 ms | 0,46 ms |
| Cold ready time | 7,76 s | 23,96 s | 5,18 s |
| Peak memory | 58,5 MiB | 250,9 MiB | 76,2 MiB |
| Idle memory | 58,3 MiB | 250,9 MiB | 9,9 MiB |
| Graph size | 13,8 MiB | 6,0 MiB | 16,1 MiB |

Map matching проверен на обычной и намеренно неоднозначной трассе:

| Сценарий | Valhalla | GraphHopper | OSRM |
|---|---|---|---|
| Admiralteyskaya | успешно, 100% / 98,1% corridor | успешно, 100% / 100% | успешно, 100% / 100% |
| Konyushennaya | успешно, 100% / 98,7% corridor | HTTP 400: broken sequence | успешно, но 78,8% / 98,7% из-за лишнего обхода |

Stage 2 дополнительно подтвердил production-shaped adapter, работу трёх реальных pedestrian
маршрутов и graph-bound compatibility guard. При намеренной подмене source checksum readiness
возвращает `503`, а preview — `409`; после восстановления сгенерированных graph metadata readiness
снова возвращает `200`.

## Rationale

OSRM быстрее и экономнее после запуска, но согласованный приоритет качества и будущего GPS map
matching выше разницы в миллисекундах локального spike. Valhalla дала лучший средний corridor
overlap и единственная сохранила ожидаемую геометрию на обоих map-matching сценариях. У
GraphHopper самый дорогой cold start и один из двух обязательных match-сценариев не выполнен.

## Consequences

- Stage 2 использует Valhalla HTTP API через engine-neutral backend port и не связывает domain
  identity с tile/edge IDs движка.
- Routing readiness требует DB, Valhalla и совместимости проверенного graph artifact с READY
  `GeoDataVersion`.
- При изменении source checksum старые tiles автоматически инвалидируются перед новой сборкой.
- OSRM compose profile и adapter сохраняются только для regression comparison.
- Измерения на Docker Desktop arm64 являются относительными; они не задают production limits.
- Перед production решением нужны полный городской extract, конкурентная нагрузка и реальные
  шумные GPS traces.

## Reproduction

```bash
make routing-spike
make routing-prepare
make routing-up
```

## Links

- [`Routing spike fixture`](../../data/routing-spike/spb-stage1/README.md)
- [`Machine-readable comparison`](../../frontend/public/routing-spike/comparison.json)
- [`Stage 2 validation report`](../stage%202/validation-report.md)
- [Valhalla API](https://valhalla.github.io/valhalla/api/)

