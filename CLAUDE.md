# CLAUDE.md

## Стиль кода

**Не пиши комментарии в коде.** Никаких поясняющих комментариев к функциям, секционных заголовков (`// --- Foo ---`), описаний параметров и т.п. Код должен быть самодокументируемым через имена идентификаторов.

Исключение — только если без комментария читатель не поймёт *почему* (скрытый инвариант, обход бага, неочевидное ограничение API). В таких редких случаях — одна короткая строка.

## Документация

Документация (`README.md` и любые `.md` файлы) пишется **полностью на русском языке**. В оригинальном виде сохраняются только идентификаторы команд, флаги, ключи конфигов и shell-примеры.

## Назначение

`secrets` — обёртка, запускающая команду с секретами из `.kdbx`-хранилища, подставленными в окружение. Секреты не попадают в историю shell и не лежат на диске в plaintext. Чтение `.kdbx` — через внешнюю `keepassxc-cli` (своего крипто нет).

```
secrets [--config <path>] [--key-store <path>] [--secrets=name:env,...] [--dry-run] -- <cmd> [args...]
secrets config  [--config <path>] [-y]
secrets check   [--config <path>] [-y]
secrets show    [--config <path>]
secrets forget  [--config <path>]
```

`--dry-run` печатает разрешённый план (конфиг, секция, маппинги, итоговая команда с плейсхолдерами `<secret from Title>`) и завершается, не читая хранилище и не запрашивая пароль.

`config` после сохранения и `check` сверяют наличие `Title` в `.kdbx` (через `keepassxc-cli export`) и предлагают создать отсутствующий файл/записи (`-y` — без подтверждений). `check` агрегирует требуемые `Title` по файлам через `resolve` по всем секциям.

## Конфиг и кэш

Схема конфига — struct `Config{ Sections map[string]Section json:"sections"; Cache *CacheConfig json:"cached,omitempty" }`. `config.Load` читает только эту схему (программа в разработке, обратной совместимости со старыми форматами нет).

Кэш пароля (`internal/keyring`): opt-in только через `cached`-секцию (`enabled`, `ttl`); никаких флагов/env. Хранит master-пароль `.kdbx` в OS-keyring через `github.com/zalando/go-keyring`. `domain.UnlockExport` — единая точка «достать пароль (кэш→prompt) + export»; `keyring.New(cfg.Cache)` строит политику. `secrets forget` (`cmdForget`) чистит keyring для всех `.kdbx` конфига. keyring-обёртки (`keyringSet/Get/Delete`) — var'ы, стабятся в тестах. Деградирует без Secret Service (miss + warning, не падает).

## Структура

Код разбит на пакеты под `internal/` (без циклов: `config`/`keepass`/`term` — листья; `keyring → config`; `domain → config,keepass,keyring,term`; `cmd → domain,config,keepass,keyring,term`; корневой `main → cmd`).

- `main.go` (package `main`) — только `var version` (через `-X main.version`) и вызов `cmd.Execute(version)`.
- `internal/config` — `Config{Sections,Cache}`/`Section`/`CacheConfig` (JSON), `Load`/`Save`; `ExpandHome`, `DefaultPath` (`~/.config/secrets/default`).
- `internal/keepass` — `CheckEngine`, `Run` (вызов `keepassxc-cli`), парсинг KeePass XML (`ParseSecrets`, тип `Entry`), `LookupSecret`; операции записи `CreateStore` (`db-create`), `AddEmptySecret` (`mkdir`+`add`).
- `internal/keyring` — `Cache`, `New(cfg.Cache)`, методы `Get/Remember/Forget`, подменяемые `keyringSet/Get/Delete` (go-keyring / Secret Service).
- `internal/term` — терминальный ввод через `/dev/tty` (`SECRETS_PASSWORD` для тестов): `ReadPassword`, `ReadWithPrefill` (fallback-ввод), `Confirm` (Y/N, учитывает `-y`), `IsInteractive`.
- `internal/domain` — логика приложения: `Resolve` (слияние `default` → секция тулзы → флаги, тип `Resolved`), `Mapping`/`MappingsFromMap`/`MappingsToMap`, `UnlockExport` (кэш→prompt→export), `AggregateStores`/`AggregateStoreMappings`, `gatherMissing`, `ReconcileStores` (отчёт + создание недостающего), `BuildStoreViews`/`StoreView`.
- `internal/cmd` — CLI-слой:
  - `execute.go` — `Execute(version)`: разбор argv, диспетчеризация (`config`/`check`/`show`/`forget`/`version`/run-режим), `usage`, `parseConfigArgs`.
  - `args.go` — `splitArgs` (по `--`), `parseRunFlags`, `mergeSecretsFlag`, тип `runFlags`.
  - `run.go` — `cmdRun`: резолв → (если `--dry-run` → `printPlan`) → пароль → export → инжект env → запуск дочерней команды, проброс кода возврата.
  - `plan.go` — `printPlan` и хелперы dry-run (`describeSection`, `renderCommand`, `shellQuote`).
  - `commands.go` — `cmdConfig` (TTY → TUI, иначе `configFallback`; после сохранения — `ReconcileStores` для default), `cmdCheck`, `cmdForget`.
  - `tui.go` — Bubble Tea-модель настройки `default`: поле `key-store` с автодополнением пути (`refreshPathSuggestions`, `deleteLastPathSegment` на `alt+backspace`, раскрытие `~` на `tab`), построчный редактор маппингов с хоткеями (`a`/`e`/`d`/`↑↓`/`tab`/`ctrl+s`/`esc`) и легендой.
  - `show.go` — `cmdShow`: read-only TUI (`showTUI`) со списком `.kdbx` (навигация `↑/↓`), `Enter` → открыть в GUI через `openInKeePassXC` (var, стабится в тестах); не-TTY → печать.

## Сборка и тесты

Зависит от `keepassxc-cli` в PATH. Go 1.26.1. TUI — на Bubble Tea (`charmbracelet/bubbletea`, `bubbles`, `lipgloss`); кэш пароля — `zalando/go-keyring` (godbus, без cgo). Сборка статическая (`CGO_ENABLED=0`).

- `just build` — собрать бинарь `./secrets` (версия из `versions.txt`).
- `just unit-test` — юнит-тесты (`go test ./...`).
- `just e2e-test` — e2e (тег `e2e`, реально создаёт/читает `.kdbx` через `keepassxc-cli`).
- `just test` — всё вместе.
- `just bump-version` — поднять `versions.txt`.

Релиз — push в `main`/`master`, тег `v<versions.txt>` (см. конвенции `dimkarp93/install`). Две площадки, одинаковые артефакты:

- GitHub Actions — `.github/workflows/release.yml`.
- Gitea Actions — `.gitea/workflows/release.yml`: без внешних actions (checkout, установка Go и публикация — шаги `run:` на shell), релиз создаётся через Gitea API.
