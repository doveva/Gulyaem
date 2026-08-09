# Architecture Decision Records

ADR фиксирует одно значимое решение, контекст его принятия, рассмотренные альтернативы и
последствия. Принятый ADR не переписывается под новое состояние: изменение решения оформляется
новым ADR, который заменяет предыдущий.

## Именование

```text
NNNN-short-decision-name.md
```

Номер назначается последовательно. Статусы: `Proposed`, `Accepted`, `Deprecated`, `Superseded`.

Новый документ создаётся по [`adr-template.md`](adr-template.md).

Исходные решения по geo-модели собраны в
[`gulyaem-geo-exploration-adr.md`](../source_context/gulyaem-geo-exploration-adr.md). После
подтверждения на Stage 1 они должны быть разнесены на отдельные ADR в этом каталоге.
