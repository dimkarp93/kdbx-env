# CLAUDE.md

## Code style

**Do not write comments in the code.** No explanatory comments on functions, no section headers (`// --- Foo ---`), no parameter descriptions and so on. The code must be self-documenting through identifier names.

The only exception is when without a comment the reader would not understand *why* (a hidden invariant, a bug workaround, a non-obvious API limitation). In such rare cases — one short line.

## Language

All code, scripts, CLI messages and comments in CI configs are **English only**.

Documentation is maintained in two languages, as file pairs:

- `<name>.ru.md` — the Russian version (the source of truth);
- `<name>.en.md` — the English version;
- `<name>.md` — a symlink to `<name>.en.md` (English is the default).

Changing a document requires editing **both** language versions. Only command identifiers, flags, config keys and shell examples are kept in their original form.

## Purpose

`kdbx-env` is a wrapper that runs a command with secrets from a `.kdbx` store injected into the environment. Secrets do not end up in the shell history and are not stored on disk in plaintext. Reading the `.kdbx` goes through the external `keepassxc-cli` (there is no in-house crypto).

```
kdbx-env [--config <path>] [--key-store <path>] [--secrets=name:env,...] [--dry-run] -- <cmd> [args...]
kdbx-env config  [--config <path>] [-y]
kdbx-env check   [--config <path>] [-y]
kdbx-env show    [--config <path>]
kdbx-env forget  [--config <path>]
```

`--dry-run` prints the resolved plan (config, section, mappings, the resulting command with `<secret from Title>` placeholders) and exits without reading the store or asking for a password.

`config` after saving, and `check`, verify that the `Title`s are present in the `.kdbx` (via `keepassxc-cli export`) and offer to create the missing file/entries (`-y` — without confirmations). `check` aggregates the required `Title`s per file via `resolve` over all sections.

## Config and cache

The config schema is the struct `Config{ Sections map[string]Section json:"sections"; Cache *CacheConfig json:"cached,omitempty" }`. `config.Load` reads only this schema (the program is under development, there is no backward compatibility with old formats).

Password caching (`internal/keyring`) is opt-in only through the `cached` section (`enabled`, `ttl`); no flags or env variables. It stores the `.kdbx` master password in the OS keyring via `github.com/zalando/go-keyring`. `domain.UnlockExport` is the single entry point for "get the password (cache→prompt) + export"; `keyring.New(cfg.Cache)` builds the policy. `kdbx-env forget` (`cmdForget`) clears the keyring for all `.kdbx` files in the config. The keyring wrappers (`keyringSet/Get/Delete`) are vars, stubbed in tests. It degrades gracefully without Secret Service (a miss plus a warning, no crash).

## Structure

The code is split into packages under `internal/` (no cycles: `config`/`keepass`/`term` are leaves; `keyring → config`; `domain → config,keepass,keyring,term`; `cmd → domain,config,keepass,keyring,term`; the root `main → cmd`).

- `main.go` (package `main`) — only `var version` (set via `-X main.version`) and the `cmd.Execute(version)` call.
- `internal/config` — `Config{Sections,Cache}`/`Section`/`CacheConfig` (JSON), `Load`/`Save`; `ExpandHome`, `DefaultPath` (`~/.config/kdbx-env/default`).
- `internal/keepass` — `CheckEngine`, `Run` (invoking `keepassxc-cli`), KeePass XML parsing (`ParseSecrets`, the `Entry` type), `LookupSecret`; write operations `CreateStore` (`db-create`), `AddEmptySecret` (`mkdir`+`add`).
- `internal/keyring` — `Cache`, `New(cfg.Cache)`, the `Get/Remember/Forget` methods, the swappable `keyringSet/Get/Delete` (go-keyring / Secret Service).
- `internal/term` — terminal input through `/dev/tty` (`KDBX_ENV_PASSWORD` for tests): `ReadPassword`, `ReadWithPrefill` (fallback input), `Confirm` (Y/N, honours `-y`), `IsInteractive`.
- `internal/domain` — application logic: `Resolve` (merging `default` → tool section → flags, the `Resolved` type), `Mapping`/`MappingsFromMap`/`MappingsToMap`, `UnlockExport` (cache→prompt→export), `AggregateStores`/`AggregateStoreMappings`, `gatherMissing`, `ReconcileStores` (report + creation of what is missing), `BuildStoreViews`/`StoreView`.
- `internal/cmd` — the CLI layer:
  - `execute.go` — `Execute(version)`: argv parsing, dispatch (`config`/`check`/`show`/`forget`/`version`/run mode), `usage`, `parseConfigArgs`.
  - `args.go` — `splitArgs` (on `--`), `parseRunFlags`, `mergeSecretsFlag`, the `runFlags` type.
  - `run.go` — `cmdRun`: resolve → (if `--dry-run` → `printPlan`) → password → export → env injection → launching the child command, propagating the exit code.
  - `plan.go` — `printPlan` and the dry-run helpers (`describeSection`, `renderCommand`, `shellQuote`).
  - `commands.go` — `cmdConfig` (TTY → TUI, otherwise `configFallback`; after saving — `ReconcileStores` for default), `cmdCheck`, `cmdForget`.
  - `tui.go` — the Bubble Tea model for configuring `default`: the `key-store` field with path completion (`refreshPathSuggestions`, `deleteLastPathSegment` on `alt+backspace`, `~` expansion on `tab`), a line-by-line mapping editor with hotkeys (`a`/`e`/`d`/`↑↓`/`tab`/`ctrl+s`/`esc`) and a legend.
  - `show.go` — `cmdShow`: a read-only TUI (`showTUI`) with a list of `.kdbx` files (navigation with `↑/↓`), `Enter` → open in the GUI via `openInKeePassXC` (a var, stubbed in tests); non-TTY → plain printing.

## Building and tests

Depends on `keepassxc-cli` being in PATH. Go 1.26.1. The TUI is built on Bubble Tea (`charmbracelet/bubbletea`, `bubbles`, `lipgloss`); password caching uses `zalando/go-keyring` (godbus, no cgo). The build is static (`CGO_ENABLED=0`).

- `just build` — build the `./kdbx-env` binary (version taken from `versions.txt`).
- `just unit-test` — unit tests (`go test ./...`).
- `just e2e-test` — e2e (the `e2e` tag, really creates/reads a `.kdbx` through `keepassxc-cli`).
- `just test` — everything at once.
- `just bump-version` — bump `versions.txt`.

Releasing — a push to `main`/`master`, tag `v<versions.txt>` (see the `dimkarp93/install` conventions). Two platforms, identical artifacts:

- GitHub Actions — `.github/workflows/release.yml`.
- Gitea Actions — `.gitea/workflows/release.yml`: no external actions (checkout, Go installation and publishing are `run:` shell steps), the release is created through the Gitea API.
