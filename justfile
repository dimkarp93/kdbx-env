tests_dir := "tests"
version_file := "versions.txt"

_default:
    @just --list

build:
    #!/usr/bin/env sh
    set -eu
    v=$(tr -d '[:space:]' < {{version_file}})
    u=$(git remote get-url origin 2>/dev/null || true)
    case "$u" in
        "")    o=local ;;
        *://*) h=${u#*://}; h=${h#*@}; o="https://${h%.git}" ;;
        *:*)   h=${u#*@};   o="https://$(printf '%s' "${h%.git}" | tr ':' '/')" ;;
        *)     o=local ;;
    esac
    if [ -f upstream.txt ]; then up=$(tr -d '[:space:]' < upstream.txt); else up="$o"; fi
    c=$(git rev-parse --short HEAD 2>/dev/null || true)
    CGO_ENABLED=0 go build -trimpath \
        -ldflags="-s -w -X main.version=$v -X main.origin=$o -X main.upstream=$up -X main.commit=$c -X main.channel=local" \
        -o kdbx-env .
    echo "Built: ./kdbx-env (v$v)"

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

bump-patch:
    #!/usr/bin/env sh
    set -eu
    v=$(tr -d '[:space:]' < {{version_file}})
    IFS=. read -r MAJ MIN PAT <<EOF
    $v
    EOF
    printf '%s.%s.%s\n' "$MAJ" "$MIN" "$((PAT + 1))" > {{version_file}}
    cat {{version_file}}

bump-minor:
    #!/usr/bin/env sh
    set -eu
    v=$(tr -d '[:space:]' < {{version_file}})
    IFS=. read -r MAJ MIN PAT <<EOF
    $v
    EOF
    printf '%s.%s.0\n' "$MAJ" "$((MIN + 1))" > {{version_file}}
    cat {{version_file}}

bump-major:
    #!/usr/bin/env sh
    set -eu
    v=$(tr -d '[:space:]' < {{version_file}})
    IFS=. read -r MAJ MIN PAT <<EOF
    $v
    EOF
    printf '%s.0.0\n' "$((MAJ + 1))" > {{version_file}}
    cat {{version_file}}
