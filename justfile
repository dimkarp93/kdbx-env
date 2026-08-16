tests_dir := "tests"
version_file := "versions.txt"

_default:
    @just --list

build:
    @./build.sh

unit-test mask="":
    go test {{ if mask != "" { "-run " + mask } else { "" } }} ./...

e2e-test mask="": build
    #!/usr/bin/env sh
    set -e
    mkdir -p {{tests_dir}}/_root
    KDBX_ENV_E2E_ROOT="$(pwd)/{{tests_dir}}/_root" \
    KDBX_ENV_E2E_KEEP=0 \
    go test -tags=e2e {{ if mask != "" { "-run " + mask } else { "" } }} ./...; \
    status=$?; \
    rm -rf {{tests_dir}}/_root; \
    exit $status

test: unit-test e2e-test

clear-tests:
    @rm -rf {{tests_dir}}/*
    @touch {{tests_dir}}/.gitkeep
    @echo "Cleared {{tests_dir}}/"

bump-version:
    #!/usr/bin/env sh
    set -e
    if [ ! -f {{version_file}} ]; then echo "{{version_file}} not found"; exit 1; fi
    cur=$(tr -d '[:space:]' < {{version_file}})
    if ! echo "$cur" | grep -Eq '^[0-9]+\.[0-9]+\.[0-9]+$'; then
      echo "{{version_file}} must contain semver MAJOR.MINOR.PATCH (got: '$cur')"; exit 1
    fi
    M=${cur%%.*}; rest=${cur#*.}; m=${rest%%.*}; p=${rest#*.}
    echo "Current version: $cur"
    while true; do
      printf "Bump which? b/major, m/minor, s/patch: "
      read kind
      case "$kind" in
        b|major) M=$((M+1)); m=0; p=0; break ;;
        m|minor) m=$((m+1)); p=0; break ;;
        s|patch) p=$((p+1)); break ;;
        "")      echo "  (no input, try again)" ;;
        *)       echo "  invalid: $kind — expected b/m/s or major/minor/patch" ;;
      esac
    done
    new="$M.$m.$p"
    echo "$new" > {{version_file}}
    echo "Updated {{version_file}}: $cur -> $new"
