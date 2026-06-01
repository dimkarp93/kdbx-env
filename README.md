# secrets

`secrets` — обёртка для CLI-утилит, которая подставляет секреты из зашифрованного `.kdbx`-хранилища в переменные окружения дочерней команды и запускает её.

Зачем: многие утилиты требуют передачи токенов через env-переменные. Делать это вручную из shell неудобно и небезопасно — секрет попадает в историю команд и в plaintext-файлы. `secrets` решает это: значение секрета берётся из зашифрованного `.kdbx`, а сам вызов выглядит так:

```sh
secrets --config ~/.config/secrets/install_secrets -- install user/repo
```

В строке вызова секрета нет — история чистая. Секрет существует только в окружении дочернего процесса.

Это тот же паттерн, что у `op run -- cmd` (1Password CLI), `envchain`, `aws-vault exec`, `sops exec-env` — но self-hosted, на базе `keepassxc`.

## Требования

Нужна установленная `keepassxc-cli` (входит в KeePassXC):

```sh
# Debian/Ubuntu
sudo apt install keepassxc
# Arch Linux
sudo pacman -S keepassxc
# macOS
brew install keepassxc
```

## Установка

Через установщик [`dimkarp93/install`](https://github.com/dimkarp93/install):

```sh
github_install.sh dimkarp93/secrets
# или одной строкой:
curl -fsSL https://raw.githubusercontent.com/dimkarp93/install/master/install.sh | sh -s -- dimkarp93/secrets
```

## Использование

```
secrets [--config <path>] [--key-store <path>] [--secrets=name:env,...] -- <cmd> [args...]
secrets config [--config <path>]
secrets version | --version | -v
```

Всё после `--` — команда, которую надо запустить. Первое слово команды (`install` в примере) выбирает секцию конфига; если такой секции нет, используется `default`.

Флаги:

- `--config <path>` — путь до конфига. По умолчанию `~/.config/secrets/default`.
- `--key-store <path>` — полный путь до `.kdbx`-файла. Перетирает значение из конфига.
- `--secrets=name:env,...` — маппинг секретов на env-переменные. Сливается поверх конфига.

При запуске `secrets` спрашивает пароль от хранилища (ввод скрыт, читается с `/dev/tty`). Отмена — `Ctrl+C` / `Ctrl+D`.

### Пример

```sh
secrets --key-store ~/secrets/tokens.kdbx --secrets=GITHUB_TOKEN:GH_TOKEN -- gh repo list
```

`secrets` прочитает запись с заголовком `GITHUB_TOKEN` из `tokens.kdbx`, положит её пароль в переменную `GH_TOKEN` и запустит `gh repo list`.

## Конфиг

Конфиг — JSON. Набор секций по именам тулз плюс секция `default`. В каждой секции:

- `key-store` — полный путь до `.kdbx`-файла;
- `secrets` — маппинг `имя_в_хранилище: имя_env`.

```json
{
  "default": {
    "key-store": "~/.config/secrets/store.kdbx",
    "secrets": { "GITHUB_TOKEN": "GH_TOKEN" }
  },
  "install": {
    "secrets": { "NPM_TOKEN": "NPM_TOKEN" }
  }
}
```

**Слияние:** `default` — это база. Секция тулзы накладывается сверху: переопределяет `key-store` (если задан) и добавляет/переопределяет маппинги секретов. Для примера выше команда `install` получит общий `key-store` из `default` и оба секрета — `GITHUB_TOKEN` и `NPM_TOKEN`. Флаги `--key-store` / `--secrets` перетирают результат.

### Как ищется секрет

«Имя секрета» в конфиге — это **заголовок (Title)** записи в `.kdbx`, значением подставляется поле **Password** этой записи. Если в хранилище несколько записей с одинаковым Title, укажите полный путь `Группа/Подгруппа/Title`:

```json
"secrets": { "web/API_KEY": "API_KEY" }
```

## Команда `config`

`secrets config` интерактивно настраивает секцию `default` (путь до `key-store` и маппинг секретов) и записывает конфиг:

```sh
secrets config
# или в произвольный файл:
secrets config --config ~/.config/secrets/install_secrets
```

## Ограничения безопасности

`secrets` инжектит секреты в окружение дочернего процесса. Env-переменные процесса доступны через `/proc/<pid>/environ` тому же пользователю (и root). Это общий компромисс всего класса инструментов (`op run`, `envchain`, `aws-vault`) — но это радикально безопаснее, чем хранить секреты в истории shell или в plaintext-файлах. Если нужна защита от чтения окружения соседними процессами того же пользователя — этот подход (как и аналоги) не подходит.

## Разработка

```sh
just build       # собрать ./secrets
just unit-test   # юнит-тесты
just e2e-test    # e2e (нужна keepassxc-cli)
just test        # всё вместе
```
