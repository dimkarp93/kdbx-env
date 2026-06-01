#!/bin/sh
set -e
V=$(tr -d '[:space:]' < versions.txt)
go build -trimpath -ldflags="-X main.version=$V" -o secrets .
echo "Built: ./secrets (v$V)"
