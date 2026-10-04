#!/bin/sh
set -eu
cd "$(dirname "$0")/.."
mkdir -p .build
go build -o .build/trial-run ./cmd/trial-run
go build -o .build/trial-batch ./cmd/trial-batch
exec .build/trial-batch "$@"
