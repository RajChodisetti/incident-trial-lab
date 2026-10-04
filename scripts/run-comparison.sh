#!/bin/sh
set -eu
cd "$(dirname "$0")/.."
mkdir -p .build
lock=.build/comparison${TRIAL_RUN_ID:+-$TRIAL_RUN_ID}.lock
mkdir "$lock" || { echo 'Comparison already running' >&2; exit 1; }
trap 'rmdir "$lock"' EXIT
trap 'exit 130' INT TERM
bundle=${TRIAL_EVIDENCE_PARENT:-evidence}/comparison-$(date -u +%Y%m%dT%H%M%SZ)-$$
mkdir -p "$bundle"
echo "Comparison evidence: $bundle"
# Existing baseline runner rejects reuse and tears down its own fresh cluster.
./scripts/run-baseline.sh > "$bundle/baseline-run.log" 2>&1
baseline_dir=$(awk '/^Evidence:/ {gsub(/;/,"",$2);print $2}' "$bundle/baseline-run.log")
cp "$baseline_dir/check.json" "$bundle/baseline.json"
printf '%s\n' "$baseline_dir" > "$bundle/baseline-evidence-path.txt"
for variant in ineffective misleading fixed; do
    ./scripts/run-candidate.sh "$variant" "$bundle/baseline.json" > "$bundle/$variant-run.log" 2>&1
    trial_dir=$(awk '/^Evidence:/ {gsub(/;/,"",$2);print $2}' "$bundle/$variant-run.log")
    cp "$trial_dir/result.json" "$bundle/$variant.json"
    printf '%s\n' "$trial_dir" > "$bundle/$variant-evidence-path.txt"
    jq --arg variant "$variant" '{candidate:$variant,verdict,recovery:.behavior.recovery,regressions:.behavior.regressions,budget:.resource_budget.status,requests:.behavior.requests}' "$bundle/$variant.json"
done
jq -n --slurpfile baseline "$bundle/baseline.json" --slurpfile ineffective "$bundle/ineffective.json" --slurpfile misleading "$bundle/misleading.json" --slurpfile fixed "$bundle/fixed.json" '{baseline_reproduced:$baseline[0].baseline_reproduced,ineffective:$ineffective[0],misleading:$misleading[0],fixed:$fixed[0]}' > "$bundle/comparison.json"
jq -e '.baseline_reproduced and .ineffective.verdict=="failed" and .ineffective.behavior.recovery=="failed" and .misleading.verdict=="failed" and .misleading.behavior.recovery=="passed" and .misleading.behavior.regressions=="failed" and .fixed.verdict=="passed"' "$bundle/comparison.json" >/dev/null
printf 'Expected comparison matrix verified. Evidence: %s\n' "$bundle"
