# Milestone 3 verified comparison

Observed 2026-10-04 UTC on Raj's macOS arm64 machine. Tested source revision:
`e0fd34dc56d3bed561eb1302dde5e70a8e9d4280`. The final commit adds documentation
only. The milestone-2 baseline source/locked manifest remained unchanged.

[Recorded summary](evidence/milestone-3-summary.json) and
[predefined protocol](../scenarios/discount-rounding/comparison-contract.md).

| Candidate | Recovery | Regressions | Measured failures / requests | Scoped budget | Verdict |
| --- | --- | --- | ---: | --- | --- |
| Ineffective | Failed | Failed | 40 / 120 | Passed | Failed |
| Misleading special-case fix | Passed | Failed | 20 / 120 | Passed | Failed |
| General subtotal fix | Passed | Passed | 0 / 120 | Passed | Passed |

The baseline first reproduced the known 283-versus-282-cent business failure,
with both control orders correct. Each candidate then ran in a separately created
and removed kind cluster with its own pinned image, kubeconfig, node, and pod.
Distinct pod UIDs were checked across all three candidates. Runtime image/config
identity was verified in every trial. No prior candidate's live state was reused.

Each trial used one six-request warmup cycle, then 20 measured cycles of six
assertions: recovery, two original controls, a different rounding input, full
discount, and maximum subtotal. No unknown observations occurred in the real
comparison. The misleading fix recovered only the original incident input but
returned 368 instead of 370 cents for the new bulk-order regression on every cycle.
The command verified the expected matrix and exited zero; failed candidate
verdicts remain failure evidence, not application recovery.

## Resource observations and limits

| Candidate | Application CPU delta | Maximum of two working-set samples |
| --- | ---: | ---: |
| Ineffective | 15.597 ms CPU | 4,874,240 bytes |
| Misleading | 11.348 ms CPU | 4,870,144 bytes |
| Fixed | 11.255 ms CPU | 4,890,624 bytes |

Predefined acceptance: <= 5 CPU-seconds between kubelet snapshots; <= 32 MiB at
both memory endpoints. All three passed these scoped checks. Every sampling window
bracketed the warmup and measured workload, with stable pod UID and monotonic CPU
counters. CPU intervals also include idle/startup time present between snapshots.

**Peak memory, CPU throttling, sustained throughput, load concurrency, statistical
latency improvement, and broad production regressions remain untested.** Endpoint
memory sampling cannot detect every transient spike. One fresh trial per candidate
and 20 sequential workload cycles do not justify a performance speedup claim.
Median/p95 durations are preserved only as descriptive observations. Passing this
limited local fixture does not certify arbitrary fixes or production safety.

## Validation

- Local formatting, race-enabled tests, `go vet`, shell syntax, and diff checks passed.
- Unit cases verify the three behavioral outcomes; missing/malformed/canceled
  evidence; invalid fixtures; baseline gating; absent/stale/reset/changed-pod
  resource counters; exceeded CPU/memory budgets; and sampling-window coverage.
- Statement coverage: trial 94.6%, candidates 73.7%, baseline 96.4%, synthetic
  baseline 100%, evaluator 100%. CLI main packages have 0% unit statement coverage;
  both service CLIs and both checker CLIs were exercised in the real local runs.
- All three candidate images were rebuilt successfully against their locked digests.
- Hosted GitHub CI remains manual and was **not run**. No external AI provider,
  API credential, permission expansion, production access, or paid resource used.

## Evidence and cleanup

Full ignored local comparison bundle:
`evidence/comparison-20261004T155311Z-46950/`. It contains comparison results and
paths to the original baseline and each candidate trial. Trial bundles preserve
all per-request responses/durations, workload bytes/hash, before/after kubelet
summaries, runtime identities, pod/node JSON, logs, source SHA, and cleanup status.

All four owned baseline/candidate clusters were removed. Every candidate runner
exited zero after cleanup and removed its generated kubeconfig. Its port-forward
child was stopped/waited. Verification found no owned cluster container, temporary
milestone-3 builder container, or builder volume left. Docker remains running;
its default context remains `desktop-linux`. No unrelated resources were changed.

Reusable local artifacts remain: project build directory about 48 MiB, tools
9.5 MiB, and evidence about 1.4 MiB, plus existing Go cache and pinned kind/BuildKit
image caches reported in milestone 2. No active project trial workload remains.
No broad prune or default kubeconfig change was performed.

Next increment is milestone 4 lifecycle robustness, pending parent/user
continuation. Concurrent scheduling, restart persistence, cancellation/cleanup
failure injection, and durable worker/controller behavior are not implemented.
