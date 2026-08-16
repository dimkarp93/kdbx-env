#!/bin/sh
set -e
V=$(tr -d '[:space:]' < versions.txt)
CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -X main.version=$V" -o kdbx-env .
echo "Built: ./kdbx-env (v$V)"
