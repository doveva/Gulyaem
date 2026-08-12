# ADR-0001: Основа импорта OSM для Stage 1

- **Status:** Accepted
- **Date:** 2026-08-09
- **Owners:** команда «ГуляЕм»
- **Related Stage:** Stage 1.2

## Context

Stage 1 должен воспроизводимо импортировать небольшой реальный OSM extract, создать собственную
`GeoDataVersion` и не допустить публикации частично обработанных данных. OSM остаётся upstream
форматом и не должен становиться persistent domain model. На этом подэтапе topology,
`StreetSegment` и walkability ещё не строятся.

## Decision

1. Первая fixture — `spb-dense-center`, bbox
   `30.3000,59.9300,30.3300,59.9450` в центре Санкт-Петербурга.
2. Snapshot хранится в репозитории как локальный `.osm.pbf`. Рядом хранятся manifest, SHA-256,
   bbox, source URL, timestamp получения и attribution. Сеть не требуется для обычного импорта.
   Contributor `user`, `uid` и `changeset` metadata удаляются при конвертации как ненужные для
   продукта; geometry, tags, object version и timestamp сохраняются.
3. PBF декодируется `github.com/paulmach/osm/osmpbf` через внутренний source adapter. Типы
   библиотеки не входят в application/domain contracts. Производительность измеряется до смены
   parser.
4. Успешный импорт идемпотентен по
   `(city_id, source_checksum, normalization_version)`. Повтор возвращает существующую `READY`
   version. Новая отличающаяся version атомарно переводит предыдущую `READY` в `SUPERSEDED`.
   `FAILED` попытка сохраняется и не мешает повторить импорт.
5. PostgreSQL хранит `City`, `GeoDataVersion`, import metadata, report и error. Raw OSM
   nodes/ways/relations остаются только в PBF и не зеркалируются в таблицы.
6. `City` и `GeoDataVersion` используют UUID. Санкт-Петербург выбирается стабильным code `spb`.
   Основной сценарий CLI — fixture name; явный file path остаётся escape hatch.

## Alternatives considered

- Загружать live OSM при каждом импорте: отклонено, потому что результат и доступность сети
  делают проверку невоспроизводимой.
- Начать с полного extract Санкт-Петербурга: отклонено до проверки pipeline на малой fixture.
- Использовать raw OSM IDs как domain identity: запрещено архитектурным контрактом.
- Сохранять raw OSM tables: отложено, пока topology implementation не докажет необходимость.
- Создавать новую `READY` version при каждом запуске одинакового файла: отклонено как
  неидемпотентное поведение без новой geo semantics.

## Consequences

- Import и тесты работают offline после clone.
- PBF является immutable source artifact, а checksum связывает его с version record.
- Изменение normalization rules требует нового `normalization_version`, даже при прежнем PBF.
- Для нового snapshot либо normalization version создаётся новая `GeoDataVersion`.
- Raw-source troubleshooting опирается на committed PBF и report, а не SQL-запросы к OSM mirror.

## Validation

- импорт fixture создаёт `READY` version с совпадающим SHA-256;
- второй импорт возвращает тот же version ID и outcome `already_ready`;
- повреждённый PBF оставляет попытку в `FAILED` и не меняет current `READY`;
- новый успешный уникальный import atomically supersedes прежнюю current version;
- parser performance пересматривается только после измерений на representative extracts.

## Links

- [`Stage 1 requirements`](../stage%201/stage-1-requirements.md)
- [`Stage 1 architecture contract`](../stage%201/architecture-contract.md)
- [`Stage 1 implementation plan`](../stage%201/implementation-plan.md)
