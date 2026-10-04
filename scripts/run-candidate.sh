#!/bin/sh
set -eu
cd "$(dirname "$0")/.."
variant=${1:?candidate variant required}
baseline_proof=${2:?baseline report required}
case "$variant" in fixed|ineffective|misleading) ;; *) echo 'Unknown candidate' >&2; exit 1;; esac
kind=.tools/kind-v0.33.0/kind
cluster=incident-trial-lab-candidate-$variant${TRIAL_RUN_ID:+-$TRIAL_RUN_ID}
if [ -n "${TRIAL_RUN_ID:-}" ]; then cluster=itl-candidate-$variant-$TRIAL_RUN_ID; fi
context=kind-$cluster
node=$cluster-control-plane
digest=$(jq -er --arg variant "$variant" '.[$variant] | select(test("^sha256:[0-9a-f]{64}$"))' deploy/local/candidates.json)
image=incident-trial-lab.invalid/candidate-$variant@$digest
archive=.build/candidates/$variant/image.oci.tar
actual=$(tar -xOf "$archive" index.json | jq -er '.manifests | if length==1 then .[0].digest else error("single platform required") end')
test "$actual" = "$digest" || { echo 'Candidate archive differs from lock' >&2; exit 1; }
jq -e '.baseline_reproduced == true' "$baseline_proof" >/dev/null || { echo 'Baseline proof required' >&2; exit 1; }
# Never reuse an existing cluster or merge the user\047s default kubeconfig.
if "$kind" get clusters | awk -v name="$cluster" '$0==name {found=1} END {exit !found}'; then
    echo "Refusing to reuse existing cluster $cluster" >&2; exit 1
fi
lock=.build/candidate-$variant${TRIAL_RUN_ID:+-$TRIAL_RUN_ID}.lock
mkdir "$lock" || { echo 'Another baseline run is active' >&2; exit 1; }
evidence=${TRIAL_EVIDENCE_PARENT:-evidence}/candidate-$variant-$(date -u +%Y%m%dT%H%M%SZ)-$$
mkdir -p "$evidence"
kubeconfig="$PWD/$evidence/kubeconfig"
created=0
forward_pid=
cleanup() {
    result=$?
    trap - EXIT
    if [ -n "$forward_pid" ]; then kill "$forward_pid" 2>/dev/null || true; wait "$forward_pid" 2>/dev/null || true; fi
    if [ "$created" = 1 ]; then
        if ! "$kind" delete cluster --name "$cluster" --kubeconfig "$kubeconfig" > "$evidence/cleanup.txt" 2>&1; then result=1; fi
        if [ -n "$(docker ps -aq --filter label=io.x-k8s.kind.cluster="$cluster")" ]; then echo 'Cluster containers remain; cleanup incomplete' >> "$evidence/cleanup.txt"; result=1; fi
    fi
    printf '%s\n' "$result" > "$evidence/exit-status.txt"
    rm -f "$kubeconfig"
    rmdir "$lock"
    printf 'Evidence: %s; exit status: %s\n' "$evidence" "$result"
    exit "$result"
}
trap cleanup EXIT
trap 'exit 130' INT TERM
created=1
"$kind" create cluster --name "$cluster" --kubeconfig "$kubeconfig" --config deploy/local/kind.yaml --wait 120s > "$evidence/cluster-create.txt" 2>&1
k() { kubectl --request-timeout=5s --kubeconfig "$kubeconfig" --context "$context" -n incident-trial-lab "$@"; }
# Import only into the node container created above; retain the content digest.
docker exec -i "$node" ctr -n k8s.io images import --digests --base-name incident-trial-lab.invalid/candidate-$variant - < "$archive" > "$evidence/image-import.txt"
docker exec "$node" ctr -n k8s.io images ls > "$evidence/node-images.txt"
rg -F "$image" "$evidence/node-images.txt" >/dev/null || { echo 'Pinned image missing from node' >&2; exit 1; }
jq --arg image "$image" '(.items[] | select(.kind=="Deployment") | .spec.template.spec.containers[0].image) = $image' deploy/local/baseline.json > "$evidence/manifest.json"
k apply -f "$evidence/manifest.json" > "$evidence/apply.txt"
if [ "${TRIAL_TEST_FAIL_AFTER_APPLY:-0}" = 1 ]; then echo 'Injected failure after apply' >&2; exit 70; fi
k rollout status deployment/synthetic-service --timeout=60s > "$evidence/rollout.txt"
k get pods -o json > "$evidence/pods.json"
k get nodes -o json > "$evidence/nodes.json"
docker exec "$node" ctr -n k8s.io content get "$digest" > "$evidence/image-manifest.json"
config_digest=$(jq -r '.config.digest' "$evidence/image-manifest.json")
observed_config=$(jq -er '.items[0].status.containerStatuses[0].image' "$evidence/pods.json")
test "$config_digest" = "$observed_config" || { echo 'Runtime config identity does not match pinned manifest' >&2; exit 1; }
runtime_digest=$(jq -er '.items[0].status.containerStatuses[0].imageID | split("@")[1]' "$evidence/pods.json")
docker exec "$node" ctr -n k8s.io content get "$runtime_digest" > "$evidence/runtime-image.json"
# containerd may report its imported OCI index rather than the child manifest.
if [ "$runtime_digest" != "$digest" ]; then
    jq -e --arg digest "$digest" 'any(.manifests[]; .digest == $digest)' "$evidence/runtime-image.json" >/dev/null || { echo 'Runtime image index does not reference pinned manifest' >&2; exit 1; }
fi
k logs deployment/synthetic-service > "$evidence/service.log"
docker stats --no-stream --format '{{json .}}' "$node" > "$evidence/node-resources.json"
k port-forward --address 127.0.0.1 deployment/synthetic-service 18080:8080 > "$evidence/port-forward.log" 2>&1 &
forward_pid=$!
ready=0
for attempt in 1 2 3 4 5 6 7 8 9 10; do
    kill -0 "$forward_pid" 2>/dev/null || { echo 'Port forward failed; see evidence' >&2; exit 1; }
    if rg -q 'Forwarding from 127.0.0.1:18080' "$evidence/port-forward.log"; then ready=1; break; fi
    sleep 1
done
test "$ready" = 1 || { echo 'Port forward deadline exceeded' >&2; exit 1; }
cp scenarios/discount-rounding/comparison-workload.json "$evidence/workload.json"
cp "$baseline_proof" "$evidence/baseline-proof.json"
printf '%s\n' "$digest" > "$evidence/image-digest.txt"
git rev-parse HEAD > "$evidence/source-commit.txt"
shasum -a 256 .build/candidates/$variant/synthetic-service > "$evidence/binary.sha256"
k get --raw "/api/v1/nodes/$node/proxy/stats/summary" > "$evidence/resources-before.json"
go run ./cmd/trial-check > "$evidence/behavior.json"
finished=$(jq -er '.finished_at | sub("\\.[0-9]+Z$";"Z") | fromdateiso8601' "$evidence/behavior.json")
ready=0
for attempt in 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16 17 18 19 20; do
    k get --raw "/api/v1/nodes/$node/proxy/stats/summary" > "$evidence/resources-after.json"
    if jq -e --argjson finished "$finished" '
      [.pods[] | select(.podRef.namespace=="incident-trial-lab") | .containers[] | select(.name=="service") | .cpu.time,.memory.time] as $times |
      ($times|length==2) and all($times[]; (sub("\\.[0-9]+Z$";"Z") | fromdateiso8601) > $finished)
    ' "$evidence/resources-after.json" >/dev/null; then ready=1; break; fi
    sleep 2
done
# A sampling timeout is preserved as unknown budget evidence, never inferred pass.
printf '%s\n' "$ready" > "$evidence/resource-samples-fresh.txt"
go run ./cmd/trial-check -behavior "$evidence/behavior.json" -before "$evidence/resources-before.json" -after "$evidence/resources-after.json" -baseline "$baseline_proof" > "$evidence/result.json"
jq '{verdict,baseline_verified,resource_budget,behavior:{recovery:.behavior.recovery,regressions:.behavior.regressions,requests:.behavior.requests,median_micros:.behavior.median_micros,p95_micros:.behavior.p95_micros}}' "$evidence/result.json"
