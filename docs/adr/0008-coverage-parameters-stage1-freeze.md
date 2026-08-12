# ADR-0008: Начальные exploration coverage параметры для Stage 2

- **Status:** Accepted
- **Date:** 2026-08-10
- **Updated:** 2026-08-11
- **Owners:** команда «ГуляЕм»
- **Related Stage:** Stage 1.7
- **Supersedes:** экспериментальный статус profiles в ADR-0005

## Context

Coverage должно отражать исследование улицы в радиусе прогулки, не требуя обходить обе стороны
дороги и каждый двор. Stage 1.5 определил формулу и три profile, а Stage 1.7 сравнил их на пяти
реальных маршрутах в трёх типах городской среды.

## Decision

1. **Balanced** становится начальным default Stage 2:
   - radius `50 м`;
   - coverage ratio `0,6`;
   - minimum required `15 м`;
   - maximum required `80 м`.
2. Completion formula, exact PostGIS intersection и локальная grade signature из ADR-0005
   сохраняются. Grade compatibility не выводится из route-wide набора: каждый normalized fragment
   покрывает только совместимые segments внутри собственного buffer.
3. `PARTIAL` сохраняется как обязательный результат вычисления. Накопление между прогулками
   относится к будущему пользовательскому progress domain, а не к `StreetSegment` или matcher.
4. Strict `35 м / 0,8 / 20–120 м` и Generous `100 м / 0,4 / 10–50 м` остаются debug profiles для
   regression/tuning; они не являются пользовательскими режимами MVP.
5. `ROUTABLE_ONLY` может соединять matched route, но никогда не увеличивает exploration coverage.
6. Custom radius разрешён в диапазоне `5–200 м`, его начальное значение в playground — `50 м`.
   Analysis context фиксирован на `225 м`, чтобы максимальный custom radius не обрезался и сохранял
   25 м запаса.

## Evidence

| Route | Matched | Strict completed | Balanced completed | Generous completed |
|---|---:|---:|---:|---:|
| Nevsky | 100% | 16,2% | 22,9% | 43,7% |
| Admiralteyskaya | 99,1% | 10,8% | 17,0% | 33,0% |
| Konyushennaya (intentional unmatched) | 91,4% | 10,0% | 15,1% | 29,4% |
| Akademicheskaya | 100% | 13,7% | 18,4% | 40,7% |
| Sosnovka | 100% | 11,6% | 17,4% | 45,4% |

`completed` здесь означает долю explorable network в фиксированном 225 м analysis context, а не
долю длины route. Balanced стабильно находится между узким Strict и заметно более широким
Generous, сохраняя intentional unmatched-фрагмент и не меняя matching между profiles.

## Consequences

- Stage 2 получает один явный default без удаления измерительных profiles.
- Пользовательская прогулка сможет накопить partial progress позднее без изменения геометрической
  семантики Stage 1.
- Параметры пересматриваются на реальных GPS traces, если 50 м систематически покрывают другую
  grade/улицу или недостаточно компенсируют GPS noise и две стороны дороги.

## Links

- [`ADR-0005`](0005-sample-route-matching-and-radius-coverage.md)
- [`Stage 1 validation report`](../stage%201/validation-report.md)
- [`Machine-readable evidence`](../../data/validation/spb-stage1/report.json)

