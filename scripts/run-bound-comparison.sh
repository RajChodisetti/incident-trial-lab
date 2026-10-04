#!/bin/sh
# Additive provenance gate; legacy trial/batch/cancellation paths remain available.
set -eu
cd "$(dirname "$0")/.."
mkdir -p .build evidence
go build -o .build/trial-run ./cmd/trial-run
go build -o .build/experiment ./cmd/experiment
logdir=$(mktemp -d "$PWD/evidence/bound-comparison-XXXXXXXX")
child=
cancel() {
    trap - INT TERM
    if [ -n "$child" ]; then kill -TERM "$child" 2>/dev/null || true; wait "$child" || true; fi
    printf 'Canceled; preserved evidence: %s\n' "$logdir" >&2
    exit 130
}
trap cancel INT TERM
run() {
    output=$1; shift
    .build/trial-run "$@" > "$output" 2>&1 &
    child=$!
    if wait "$child"; then child=; else child=; cat "$output" >&2; exit 1; fi
}
run "$logdir/baseline.log" -mode baseline
baseline_run=$(sed '1d' "$logdir/baseline.log" | jq -er '.run_id')
.build/experiment -action register -baseline-run "$baseline_run" > "$logdir/experiment.json"
experiment_id=$(jq -er '.experiment_id' "$logdir/experiment.json")
for variant in ineffective misleading fixed; do
    .build/experiment -action reserve -experiment "$experiment_id" -variant "$variant" > "$logdir/$variant-attempt.json"
    proof=$(jq -er '.baseline_proof' "$logdir/$variant-attempt.json")
    attempt_id=$(jq -er '.attempt_id' "$logdir/$variant-attempt.json")
    run "$logdir/$variant.log" -mode candidate -variant "$variant" -baseline "$proof"
    candidate_run=$(sed '1d' "$logdir/$variant.log" | jq -er '.run_id')
    .build/experiment -action bind -experiment "$experiment_id" -attempt "$attempt_id" -candidate-run "$candidate_run" > "$logdir/$variant-receipt.json"
    .build/experiment -action verify -experiment "$experiment_id" -attempt "$attempt_id" > "$logdir/$variant-verified.json"
    expected=failed; if [ "$variant" = fixed ]; then expected=passed; fi
    jq -e --arg expected "$expected" '.verdict == $expected' "$logdir/$variant-receipt.json" >/dev/null
    # Exercise replay refusal against actual completed evidence.
    if .build/experiment -action bind -experiment "$experiment_id" -attempt "$attempt_id" -candidate-run "$candidate_run" > "$logdir/$variant-replay.stdout" 2> "$logdir/$variant-replay.stderr"; then
        echo 'Replayed candidate evidence accepted' >&2; exit 1
    fi
    printf '%s: %s; replay refused\n' "$variant" "$expected"
done
if .build/experiment -action register -baseline-run "$baseline_run" > "$logdir/baseline-replay.stdout" 2> "$logdir/baseline-replay.stderr"; then
    echo 'Replayed baseline accepted' >&2; exit 1
fi
printf 'Experiment %s; evidence: %s\n' "$experiment_id" "$logdir"
