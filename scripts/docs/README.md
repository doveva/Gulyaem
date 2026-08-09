# Инструменты документации

Каталог содержит локальные проверки docs-as-code. Скрипты используют только стандартную
библиотеку Python и Git, поэтому те же команды можно позже перенести в GitLab CI.

## Команды

```bash
python3 scripts/docs/docs.py index
python3 scripts/docs/docs.py index --check
python3 scripts/docs/docs.py check
python3 scripts/docs/docs.py check --base main
```

- `index` обновляет автоматически формируемые блоки в корневом `README.md` и
  `docs/README.md`.
- `index --check` завершается с ошибкой, если индекс устарел.
- `check` проверяет незакоммиченный и staged diff.
- `check --base <ref>` дополнительно проверяет накопленный diff относительно Git ref.

Политика расширений и исключений находится в [`docs/docs-config.json`](../../docs/docs-config.json).
