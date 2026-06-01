# CLAUDE.md

## Стиль кода

**Не пиши комментарии в коде.** Никаких поясняющих комментариев к функциям, секционных заголовков (`// --- Foo ---`), описаний параметров и т.п. Код должен быть самодокументируемым через имена идентификаторов.

Исключение — только если без комментария читатель не поймёт *почему* (скрытый инвариант, обход бага, неочевидное ограничение API). В таких редких случаях — одна короткая строка.

## Документация

Документация (`README.md` и любые `.md` файлы) пишется **полностью на русском языке**. В оригинальном виде сохраняются только идентификаторы команд, флаги, ключи конфигов и shell-примеры.

## Назначение

`secrets` — обёртка, запускающая команду с секретами из `.kdbx`-хранилища, подставленными в окружение. Секреты не попадают в историю shell и не лежат на диске в plaintext. Чтение `.kdbx` — через внешнюю `keepassxc-cli` (своего крипто нет).

```
secrets [--config <path>] [--key-store <path>] [--secrets=name:env,...] -- <cmd> [args...]
secrets config [--config <path>]
```

## Структура

- `main.go` — разбор argv, диспетчеризация (`config`, `version`, run-режим).
- `args.go` — `splitArgs` (по `--`), `parseRunFlags`, `mergeSecretsFlag`.
- `config.go` — `Config`/`Section` (JSON), `loadConfig`/`saveConfig`, `resolve` (слияние `default` → секция тулзы → флаги).
- `keepass.go` — `checkEngine`, `runKP`, парсинг KeePass XML (`parseSecrets`), `lookupSecret`.
- `run.go` — `cmdRun`: резолв → пароль → `keepassxc-cli export -f xml` → инжект env → запуск дочерней команды, проброс кода возврата.
- `commands.go` — `cmdConfig` (интерактивная настройка `default`).
- `term.go` — ввод пароля через `/dev/tty` (`SECRETS_PASSWORD` для тестов), `readWithPrefill`.
- `paths.go` — `expandHome`, `defaultConfigPath` (`~/.config/secrets/default`).

## Сборка и тесты

Зависит от `keepassxc-cli` в PATH. Go 1.26.1.

- `just build` — собрать бинарь `./secrets` (версия из `versions.txt`).
- `just unit-test` — юнит-тесты (`go test ./...`).
- `just e2e-test` — e2e (тег `e2e`, реально создаёт/читает `.kdbx` через `keepassxc-cli`).
- `just test` — всё вместе.
- `just bump-version` — поднять `versions.txt`.

Релиз — push в `main`/`master`, тег `v<versions.txt>` собирается через `.github/workflows/release.yml` (см. конвенции `dimkarp93/install`).
