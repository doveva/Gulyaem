# Деплой и эксплуатация

Операционная точка входа для локального запуска Stage 1. Приложение поддерживает запуск
зависимостей в Docker с API/frontend на хосте и сборку всего стека через Docker Compose.

## Требования

- Go 1.25+;
- Node.js 24+ и npm;
- Docker с Compose.

## Полный стек в Docker Compose

```bash
cp .env.example .env
docker compose up --build -d
docker compose ps
```

Compose собирает нативный AMD64/ARM64 PostGIS-образ поверх официального PostgreSQL 17, выполняет
миграции один раз, затем запускает API и статический frontend. Playground доступен на
`http://localhost:3000/debug/geo`, readiness API — на `http://localhost:8080/health/ready`.
Для нового пустого named volume один раз опубликуйте оба локальных набора:

```bash
docker compose run --rm geo-import
docker compose run --rm district-import
```

Повтор обоих импортов идемпотентен.

Остановить контейнеры:

```bash
docker compose down
```

Named volume с PostgreSQL сохраняется. Для удаления данных требуется отдельное явное действие;
обычный `down` их не удаляет.

## Локальная разработка

Установить зависимости:

```bash
make bootstrap
cp .env.example .env
```

Поднять PostGIS и применить миграции:

```bash
make db-up
make migrate
make geo-import
make district-import
```

В первом терминале запустить API:

```bash
make api
```

Во втором терминале запустить Vite:

```bash
make frontend
```

Playground будет доступен на `http://localhost:5173/debug/geo`.

`make geo-import` читает committed `spb-dense-center.osm.pbf`. Повторный запуск является
идемпотентным и не требует сети. Импорт строит `StreetSegment` в памяти и публикует их вместе с
новой `GeoDataVersion` одной транзакцией. Container-вариант той же операции:

```bash
docker compose run --rm geo-import
docker compose run --rm district-import
```

## Конфигурация

Пример всех параметров находится в `.env.example`. Пароли в нём предназначены только для
локальной разработки. Реальные секреты не должны попадать в Git.

Если проектные `5532`, `8080` или `3000` заняты, измените соответствующие `*_PORT` в `.env`.
При смене frontend-порта добавьте новый origin в `CORS_ALLOWED_ORIGINS`; при смене API-порта
обновите browser-visible `VITE_API_URL`.

`Makefile` загружает корневой `.env`. Для host-mode `make api` строит `DATABASE_URL` из
`POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_PORT` и `POSTGRES_DB`, а адрес прослушивания — из
`API_PORT`. Поэтому изменение `POSTGRES_PORT` одинаково применяется к `make db-up` и `make api`.
Значения, переданные как аргументы Make, имеют приоритет, например:

```bash
make api POSTGRES_PORT=55432 API_PORT=18080
```

Для нестандартного или внешнего PostgreSQL можно передать готовый URL:

```bash
make api DATABASE_URL='postgres://user:password@db.example/gulyaem?sslmode=require'
```

Frontend-переменные `VITE_*` встраиваются во время сборки образа. После их изменения frontend
нужно пересобрать. `VITE_MAP_STYLE_URL` по умолчанию указывает на публичный OpenFreeMap Liberty;
стиль не является доменным контрактом и заменяется конфигурацией. `VITE_CITY_ID` выбирает город
для инженерного playground; default совпадает с seeded UUID Санкт-Петербурга.

`GEO_TEST_AREA` по умолчанию равен `spb-dense-center`. `NORMALIZATION_VERSION` входит в identity
версии вместе с checksum: после изменения normalization rules его значение обязательно меняется.
Stage 1.3 использует `stage1-segments-v1`. Если локальный `.env` был создан на Stage 1.2 и всё ещё
содержит `stage1-v1`, обновите его либо выполните:

```bash
make geo-import NORMALIZATION_VERSION=stage1-segments-v1
```

`MAX_SEGMENT_LENGTH_M=0` отключает artificial length splitting. Положительное значение оставлено
только для контролируемых экспериментов.

`DISTRICT_TEST_AREA=spb-administrative-districts` выбирает committed GeoJSON с 18 районами.
`DISTRICT_NORMALIZATION_VERSION=stage1-districts-v1` входит в identity независимой
`DistrictDataVersion`. Оба импорта работают без runtime-доступа к сети.

## Миграции

Миграции находятся в `backend/migrations` и выполняются образом `golang-migrate`:

```bash
docker compose run --rm migrate
```

При полном запуске Compose сервис `api` стартует только после успешной миграции. Первая миграция
включает PostGIS; readiness проверяет вызов `PostGIS_Version()`.

## Health checks и диагностика

- `/health/live` проверяет процесс API;
- `/health/ready` проверяет соединение с PostgreSQL и доступность PostGIS;
- `/health` внутри frontend-контейнера проверяет nginx.

Структурированные JSON-логи API доступны через:

```bash
docker compose logs -f api
```

## Проверка

```bash
make check
```

Команда запускает Go tests/vet, frontend lint/tests/build и проверки документации.

## Ограничения Stage 1.4

Production deployment, backup/restore и rollback application-релиза пока не определены. Данные
этого этапа локальные и воспроизводимые. Районы доступны как независимый слой; генерация маршрутов
и coverage относятся к Stage 1.5.
