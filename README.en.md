# kdbx-env

`kdbx-env` is a wrapper for CLI tools that injects secrets from an encrypted `.kdbx` store into the environment variables of a child command and runs it.

Why: many tools require tokens to be passed via env variables. Doing that by hand from the shell is inconvenient and unsafe — the secret ends up in the command history and in plaintext files. `kdbx-env` solves this: the secret value is taken from an encrypted `.kdbx`, and the invocation looks like this:

```sh
kdbx-env --config ~/.config/kdbx-env/install_secrets -- install user/repo
```

There is no secret on the command line — the history stays clean. The secret exists only in the environment of the child process.

This is the same pattern as `op run -- cmd` (1Password CLI), `envchain`, `aws-vault exec`, `sops exec-env` — but self-hosted, on top of `keepassxc`.

## Requirements

`keepassxc-cli` must be installed (it ships with KeePassXC):

```sh
# Debian/Ubuntu
sudo apt install keepassxc
# Arch Linux
sudo pacman -S keepassxc
# macOS
brew install keepassxc
```

## Installation

Via the [`dimkarp93/install`](https://github.com/dimkarp93/install) installer:

```sh
github_install.sh dimkarp93/kdbx-env
# or as a one-liner:
curl -fsSL https://raw.githubusercontent.com/dimkarp93/install/master/install.sh | sh -s -- dimkarp93/kdbx-env
```

## Usage

```
kdbx-env [--config <path>] [--key-store <path>] [--secrets=name:env,...] [--dry-run] -- <cmd> [args...]
kdbx-env config [--config <path>]
kdbx-env version | --version | -v
```

Everything after `--` is the command to run. The first word of the command (`install` in the example) selects the config section; if there is no such section, `default` is used. Flags must come **before** `--`.

Flags:

- `--config <path>` — path to the config. Defaults to `~/.config/kdbx-env/default`.
- `--key-store <path>` — full path to the `.kdbx` file. Overrides the value from the config.
- `--secrets=name:env,...` — mapping of secrets to env variables. Merged on top of the config.
- `--dry-run` — do not run the command and do not touch the store: print the resolved plan (see below).

On startup `kdbx-env` asks for the store password (input is hidden, read from `/dev/tty`). Cancel with `Ctrl+C` / `Ctrl+D`.

### Example

```sh
kdbx-env --key-store ~/secrets/tokens.kdbx --secrets=GITHUB_TOKEN:GH_TOKEN -- gh repo list
```

`kdbx-env` reads the entry titled `GITHUB_TOKEN` from `tokens.kdbx`, puts its password into the `GH_TOKEN` variable and runs `gh repo list`.

### Preview: `--dry-run`

With the `--dry-run` flag the command is not executed, no password is requested and the `.kdbx` is not read. The resolved plan is printed instead: which config is used, which section is applied, the active mappings and the resulting command (secret values are replaced with the `<secret from Title>` placeholder):

```sh
$ kdbx-env --dry-run -- install user/repo
Dry run — the command will NOT be executed.

Config file:  /home/user/.config/kdbx-env/default
Tool:         install
Section:      "install" (merged over "default")
Key-store:    ~/.config/kdbx-env/store.kdbx

Mappings (env ← secret):
  GH_TOKEN  ← GITHUB_TOKEN
  NPM_TOKEN ← NPM_TOKEN

Command:
  GH_TOKEN=<secret from GITHUB_TOKEN> NPM_TOKEN=<secret from NPM_TOKEN> install user/repo
```

## Config

The config is a JSON object with the fields `sections` (a set of sections named after tools plus `default`) and an optional `cached` (see [Password caching](#password-caching)). Every section has:

- `key-store` — full path to the `.kdbx` file;
- `secrets` — mapping `store_entry_name: env_name`.

```json
{
  "sections": {
    "default": {
      "key-store": "~/.config/kdbx-env/store.kdbx",
      "secrets": { "GITHUB_TOKEN": "GH_TOKEN" }
    },
    "install": {
      "secrets": { "NPM_TOKEN": "NPM_TOKEN" }
    }
  },
  "cached": { "enabled": true, "ttl": "10m" }
}
```

**Merging:** `default` is the base. The tool section is layered on top: it overrides `key-store` (if set) and adds/overrides secret mappings. For the example above, the `install` command gets the shared `key-store` from `default` and both secrets — `GITHUB_TOKEN` and `NPM_TOKEN`. The `--key-store` / `--secrets` flags override the result.

### How a secret is looked up

The "secret name" in the config is the **Title** of an entry in the `.kdbx`, and the value substituted is the **Password** field of that entry. If the store has several entries with the same Title, specify the full path `Group/Subgroup/Title`:

```json
"secrets": { "web/API_KEY": "API_KEY" }
```

## The `config` command

`kdbx-env config` interactively configures the `default` section (the `key-store` path and the secret mappings) and writes the config:

```sh
kdbx-env config
# or into an arbitrary file:
kdbx-env config --config ~/.config/kdbx-env/install_secrets
```

A TUI opens in the terminal:

- if the config already exists, the fields are **prefilled** with the current values;
- the `key-store` field supports **path completion**: `Tab` — complete (and expand `~` into the full path), `↑/↓` — cycle through the options in the current directory, `Alt+Backspace` — delete the last path segment (up to `/`);
- secret mappings are shown **line by line** and edited with hotkeys: `a` — add, `e` — edit, `d` — delete, `↑/↓` — select, `Tab` — switch between the path field and the list, `Ctrl+S` — save, `Esc` — cancel. The hotkey legend is always visible at the bottom.

If stdin/stdout is not a terminal (a pipe, a script), `config` falls back to simple line-by-line input without the TUI.

After saving, `config` checks the store of the `default` section:

- if the `.kdbx` file does not exist, it offers to create it (asking for a new password);
- then it verifies that every specified `Title` is present in the store; missing ones are listed and it offers to add them as empty entries.

With the `-y` flag everything missing (the file, the secrets) is created without questions.

## The `check` command

`kdbx-env check` verifies that all `.kdbx` files referenced by the config (across all sections, taking merging with `default` into account) contain entries with every required `Title`. Missing ones are printed **grouped by file**, after which it offers to add them as empty entries.

```sh
kdbx-env check
kdbx-env check --config ~/.config/kdbx-env/install_secrets
kdbx-env check -y          # add all missing entries without confirmation
```

Sample output:

```
Missing secrets:
  /home/user/.config/kdbx-env/store.kdbx
    - NPM_TOKEN
  /home/user/work/deploy.kdbx  (key-store does not exist)
    - AWS_KEY
```

A password is requested for every `.kdbx` (it is needed both to read and to add entries).

## The `show` command

`kdbx-env show` displays the current config: the file path and the list of `.kdbx` stores with their mappings (`name → env`). You can navigate the stores with the `↑/↓` arrows, and `Enter` opens the selected file in the **KeePassXC GUI** (the `keepassxc` binary).

```sh
kdbx-env show
kdbx-env show --config ~/.config/kdbx-env/install_secrets
```

- Files missing on disk are marked `(missing)`; they cannot be opened.
- `q` / `Esc` — quit.
- If stdin/stdout is not a terminal, `show` simply prints the config without navigation.

## Password caching

Within a single `kdbx-env` invocation the database password is asked once. To avoid retyping it **between** invocations during a session, caching can be enabled in the config:

```json
"cached": { "enabled": true, "ttl": "10m" }
```

- `enabled` — enable the cache (disabled by default).
- `ttl` — entry lifetime (Go `time.ParseDuration` format: `30s`, `10m`, `2h`; defaults to `10m`).

The password is stored in the **OS keyring** via Secret Service (gnome-keyring / KWallet) and is controlled only through the config (there are no CLI flags or env variables). To drop the cache: `kdbx-env forget` (clears the entries for every `.kdbx` from the config).

**What this implies and the risks:**

- What is cached is the **master password** of the `.kdbx` — it opens the **whole** database, not just the injected variables. That is a more valuable target than the environment of a child process.
- In the keyring the password is encrypted on disk and decrypted in the daemon's memory for the duration of the login session; you can inspect/revoke it in **seahorse** ("Passwords and Keys").
- The basic Secret Service has **no** per-application separation: any process running as the same user can read the entry (the same trust boundary as `/proc/<pid>/environ`).
- If Secret Service is unavailable (headless, SSH, no session bus), the cache simply does not work and the password is requested as usual (the command does not fail).

## Security limitations

`kdbx-env` injects secrets into the environment of a child process. A process's env variables are readable via `/proc/<pid>/environ` by the same user (and root). This is the common tradeoff of this whole class of tools (`op run`, `envchain`, `aws-vault`) — but it is radically safer than keeping secrets in the shell history or in plaintext files. If you need protection from neighbouring processes of the same user reading the environment, this approach (like its analogues) is not suitable.

## Development

```sh
just build       # build ./kdbx-env
just unit-test   # unit tests
just e2e-test    # e2e (requires keepassxc-cli)
just test        # everything at once
```
