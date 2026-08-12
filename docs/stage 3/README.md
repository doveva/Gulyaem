# Stage 3 — Exploration Core

- **Status:** Draft
- **Goal:** превратить stateless route preview в persistent Walk и персональное exploration state
- **Started:** 2026-08-12
- **Completed:** —

## Входные гарантии

Stage 3 опирается на замороженные результаты Stage 1/2:

- versioned internal `StreetSegment`;
- `EXPLORE / ROUTABLE_ONLY / IGNORE`;
- topology-first segmentation;
- grade-aware Balanced coverage;
- Valhalla;
- routing graph provenance;
- request-pinned `GeoDataVersion`;
- stateless `/api/v1/route-previews`;
- mobile route-builder flow;
- bbox + GeoJSON.

## Документы этапа

| Документ | Назначение |
|---|---|
| [`stage-3-requirements.md`](stage-3-requirements.md) | Scope, сценарии, FR/NFR и Definition of product behavior |
| [`architecture-contract.md`](architecture-contract.md) | Fixed boundaries и consistency invariants |
| [`domain-model.md`](domain-model.md) | Route, Walk, progress, delta и lifecycle semantics |
| [`persistence-model.md`](persistence-model.md) | SQL-oriented schema proposal и indexes |
| [`api-contract.md`](api-contract.md) | HTTP contract и errors |
| [`frontend-flow.md`](frontend-flow.md) | `/map` flow от preview до summary |
| [`implementation-plan.md`](implementation-plan.md) | Последовательность Stage 3.1–3.9 |
| [`acceptance-criteria.md`](acceptance-criteria.md) | Проверяемый DoD |
| [`open-decisions.md`](open-decisions.md) | Evidence-driven decisions |
| [`validation-plan.md`](validation-plan.md) | Automated / integration / E2E / mobile validation |

## Proposed ADR

- [`ADR-0010`](../adr/0010-stage3-route-materialization.md) — server-side materialization from preview.
- [`ADR-0011`](../adr/0011-stage3-walk-completion-transaction.md) — atomic/idempotent completion.
- [`ADR-0012`](../adr/0012-stage3-exploration-read-model.md) — progress read model and rebuild.
- [`ADR-0013`](../adr/0013-stage3-development-actor-context.md) — actor ownership before authentication.

ADR имеют `Proposed` status до реализации и validation.

## Stage result

После завершения сюда добавляются:

- фактические schema/API deviations;
- принятые ADR;
- performance measurements;
- validation report;
- known limitations;
- readiness decision для Stage 4.
