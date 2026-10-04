#!/bin/sh
set -eu
cd "$(dirname "$0")/.."
manifest=${1:-deploy/local/baseline.json}
jq -e '
  [.items[] | select(.kind == "Deployment") | .spec.template.spec.containers[].image] as $images |
  ($images | length == 1) and all($images[]; test("^incident-trial-lab[.]invalid/synthetic-service@sha256:[0-9a-f]{64}$"))
' "$manifest" >/dev/null || { echo 'Require one digest-pinned local baseline image' >&2; exit 1; }
