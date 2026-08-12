# Stage 2 — Validation Report

**Дата:** 2026-08-12  
**Freeze status:** engineering complete; 98 / 100 acceptance criteria checked  
**Состояние сборки:** рабочее дерево Stage 2, Compose image `gulyaem/backend:stage2`  
**Valhalla:** `3.7.0`, pedestrian  
**GeoDataVersion:** `8b2897ce-4a9c-4361-98b1-d7a0fade201d`  
**Source checksum:** `05fa864bb753ffc4b2c632deae28e6b6ed80b8e677f65a9689d786596aa0e8ae`  
**Graph artifact:** `valhalla_tiles.tar`, SHA-256 recorded after Valhalla becomes healthy

Valhalla startup logs confirm `Tile extract successfully loaded`; this is the same tar artifact
whose SHA-256 the API verifies, not merely the neighboring source PBF.

## Результат

Вертикальный Stage 2 flow работает end-to-end:

```text
/map → ordered waypoints → Go API → Valhalla → reusable RouteAnalyzer → PostGIS → coverage preview
```

Preview остаётся stateless. В схему не добавлены `Route`, `Walk`, `User` или
`UserStreetProgress`. Product UI использует только понятие потенциального покрытия.

## Automated checks

- Go unit/contract/API suite: `CGO_ENABLED=0 go test ./...` — pass.
- Go static analysis: `CGO_ENABLED=0 go vet ./...` — pass.
- Real PostGIS grade-aware route-analysis integration test — pass.
- Real PostGIS version-switch regression: A resolved, B published, matching and coverage remain
  pinned to A — pass.
- Low-match decision regression: known ambiguous ratio `0.914470569` warns, `0.95` and normal
  99–100% ratios do not — pass.
- Frontend lint — pass.
- Frontend unit tests — 27 pass.
- Frontend production build — pass.
- Playwright full regression with Compose API — 6 pass:
  - existing `/debug/geo` flow and error/empty cases;
  - `/map` two-point preview and intermediate-point editing;
  - no-route recovery without losing waypoints;
  - mobile viewport two-point flow.
- Documentation index and docs-as-code check — pass.
- `docker compose config --quiet` — pass.
- Routing metadata contract verifies the real mounted graph artifact checksum — pass.
- Dataset mismatch readiness test (`GeoDataVersion` versus graph-bound metadata) — pass.
- Isolated stale-graph auto-invalidation regression — pass.
- Live mismatch drill: `/health/ready` returned 503 and `/route-previews` returned 409; after
  restoring the generated metadata, readiness returned 200 — pass.

## Runtime routes

Все маршруты построены реальным Valhalla и проанализированы текущим READY geo dataset.

| Environment | Representative points | Total | Routing | Analysis | Match | Raw payload | Gzip |
|---|---|---:|---:|---:|---:|---:|---:|
| Dense center | `59.935,30.305 → 59.940,30.325` | 0.90 s | 7 ms | 0.88 s | 100% | 131 KB | 27 KB |
| Regular urban | `60.010,30.390 → 60.035,30.415` | 1.34 s | 16 ms | 1.32 s | 100% | 294 KB | 72 KB |
| Park + residential | `60.012,30.335 → 60.030,30.360` | 0.98 s | 11 ms | 0.97 s | 100% | 192 KB | 49 KB |

Product payload omits `NOT_COVERED` segment geometries while retaining context metrics; it keeps
`COMPLETED`, `PARTIAL` and diagnostic connector geometries. HTTP responses negotiate gzip.

## Warmed latency

Thirty sequential warmed requests used the dense-center route:

| Metric | p50 | p95 | min | max |
|---|---:|---:|---:|---:|
| Total preview | 1105 ms | 1273 ms | 832 ms | 1581 ms |
| Valhalla | 2 ms | 4 ms | — | — |
| Route analysis | 1102 ms | 1268 ms | — | — |

The current p95 meets the Stage 2 engineering target of 1–2 seconds. PostGIS analysis, not
Valhalla, is the dominant cost and is emitted separately in structured logs. Exact coordinates are
not logged.

## Interaction and responsive validation

- Desktop route builder: start/destination, explicit intermediate point, delete, clear and error
  recovery verified in Playwright.
- Reorder and stale-response acceptance are covered by reducer/unit tests; production requests use
  both `AbortController` and a monotonic sequence check.
- Marker movement updates React state only on MapLibre `dragend`; routing is not called during
  pointer movement.
- Mobile viewport `390 × 844`: two-point flow, summary sheet, actions and marker visibility pass.
- Existing `/debug/geo` regression remains green.

## Known limitations and follow-up

- AC-84 interaction review на физическом мобильном устройстве и AC-86 unaided comprehension review
  остаются product-validation activities; automated viewport validation завершена. Эти два пункта
  явно не отмечены в [`acceptance-criteria.md`](acceptance-criteria.md).
- The initial JavaScript bundle is about 1.18 MB raw / 317 KB gzip because MapLibre and the product
  and debug maps share one entry chunk. This does not block Stage 2, but route-level code splitting
  is the first frontend performance follow-up if field measurements show slow startup.
- The three representative payloads are acceptable after gzip; bbox + GeoJSON remains the chosen
  background-network delivery. There is no evidence requiring vector tiles.

## Documentation freeze

- ADR-0006…0008 восстановлены в каноническом каталоге `docs/adr/` как принятые решения.
- ADR-0009 принят на основании Stage 1 bbox и Stage 2 payload/interaction evidence.
- Acceptance checklist синхронизирован с validation report: 98 из 100 подтверждены, два field
  validation пункта оставлены открытыми явно.
- Прежние открытые формулировки преобразованы в decision log; Stage 3 materialization boundary
  отмечена как осознанно deferred, а не как незакрытое Stage 2 решение.

## Stage 3 readiness

The route-preview contract, geo-version reference and potential coverage output are usable as
inputs for a future Stage 3 materialization decision. Stage 3 must add personal progress semantics
outside this preview service rather than changing Stage 2 coverage meaning.

The routing compatibility guard is graph-bound rather than environment-bound. Valhalla startup
produces `routing-dataset.json` only after a healthy graph exists; the API verifies the artifact
SHA-256 and compares its recorded source checksum with the current READY `GeoDataVersion`.
`/health/ready` therefore requires DB + Valhalla + compatible routing dataset, and cannot report
ready while every route preview would fail with a dataset mismatch.
