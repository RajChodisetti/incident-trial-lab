# Deterministic candidate comparison

Milestone 3 adds a small sequential local comparison, not an agent or a durable
worker/controller. See [verified evidence](milestone-3-evidence.md).
The [protocol](../scenarios/discount-rounding/comparison-contract.md)
was written before running candidates; it defines assertions, cycle count, resource
budgets, and explicit measurement limits.

```sh
./scripts/check.sh
./scripts/build-candidates.sh
./scripts/run-comparison.sh
```

Use the same Go 1.26.2/macOS arm64 and local prerequisites as milestone 2. The
milestone-2 baseline archive must exist (recreate it with `./scripts/build-baseline.sh`
if needed); its code and locked manifest are unchanged. Each candidate is a separate
binary/image with its variant baked in, rather than a mutable runtime flag. The
candidate builder checks its output against `deploy/local/candidates.json` on rebuild.
The variant functions are explicit synthetic examples, not generated agent patches.

Read `internal/candidates/service.go` to explain each proposal:

- `ineffective` leaves per-unit rounding unchanged.
- `misleading` repairs only the original incident tuple, hiding overfitting from
  anyone who reruns just that one assertion.
- `fixed` applies the discount once to the complete order subtotal.

The unchanged baseline must reproduce its known defect first. Each candidate gets
an entirely fresh kind cluster, namespace, pod UID, kubeconfig, and image import.
The runner rejects an existing cluster or image digest mismatch, verifies runtime
image/config identity, and stops/waits its port forward and deletes its owned
cluster at exit. It does not mutate the default kubeconfig, reuse a previous
candidate's state, or apply changes outside the dedicated local trial.

The checker runs one warmup cycle plus 20 measured cycles of six orders. Recovery
and regression verdicts are separate; every measured request preserves its raw
response, status, expected/observed total, duration, and error. Unavailable or
malformed responses are unknown. The general evaluator only passes a candidate
when verified baseline, behavior, and scoped budget evidence all pass.

Resources come from the application container in kubelet `stats/summary`, not
Docker's entire control plane. Snapshot timestamps must bracket the entire warmup
and measured workload, pod UID must match, and cumulative CPU cannot reset. CPU
budget is <= 5 CPU-seconds over that snapshot interval; both endpoint working-set
samples must be <= 32 MiB. This is **not peak-memory coverage**. Startup/idle time
inside the sampling interval contributes to CPU; p95/median request durations are
descriptive and are not a statistically rigorous throughput/latency comparison.

Ignored `evidence/comparison-*` bundles link the baseline and individual trial
bundles. Every candidate has raw observations, before/after kubelet stats, fixture,
image/runtime identities, pod JSON, cleanup evidence, and a scoped result. A failed
candidate is valid experimental evidence; the comparison command succeeds only
when the expected fixed/ineffective/misleading matrix is observed.

Milestone 4 adds [supervised execution](milestone-4.md); prefer
`./scripts/run-trial.sh -mode comparison -timeout 10m` to direct shell execution.

No model/API integration, credentials, hosted CI, commercial telemetry, external
apply, production access, or unrelated Docker cleanup is needed. Keep the repo
private. Next is milestone 4 lifecycle robustness: bounded workers, deadlines,
cancellation, failure injection, and restart/cleanup behavior. Current shell traps
are cleanup conveniences, not a tested durable cancellation/recovery mechanism.
