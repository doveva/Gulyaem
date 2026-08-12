# Stage 3 documentation integration

Этот архив является overlay для репозитория, на основе которого он был сформирован.

Из корня репозитория распакуйте архив с сохранением путей/заменой файлов.

Он:

- заменяет root `AGENTS.md` и `README.md` текущим Stage 3 context;
- добавляет `docs/stage 3/`;
- добавляет proposed ADR-0010…0013;
- обновляет `docs/stages/README.md`, `docs/architecture/README.md`;
- обновляет сгенерированный documentation index в root/docs README.

После распаковки рекомендуется выполнить:

```bash
python3 scripts/docs/docs.py index --check
python3 scripts/docs/docs.py check
```

`docs.py check` проверяет только изменённые code files, поэтому на чистом documentation-only diff
он должен сообщить, что documentation-sensitive code changes отсутствуют.
