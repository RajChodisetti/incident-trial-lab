#!/bin/sh
set -eu

cd "$(dirname "$0")/.."
unformatted=$(gofmt -l .)
if [ -n "$unformatted" ]; then
    printf 'Go files need formatting:\n%s\n' "$unformatted" >&2
    exit 1
fi
go test -race -count=1 -cover ./...
go vet ./...
