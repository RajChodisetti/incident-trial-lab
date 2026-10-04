# Milestone 3 comparison protocol

Define assertions and budgets before candidate runs. This deterministic local
exercise has no model/API calls. Reproduce the unchanged milestone-2 baseline
first. Every candidate then gets its own new kind cluster; never reuse a trial.

Candidates: `ineffective` preserves the faulty per-unit truncation; `misleading`
fixes only the exact original incident input; `fixed` applies the business formula
to every valid order. A broader regression input must reject the special-case fix.
These are explicit synthetic variants, not independently discovered agent patches.

Workload: one unmeasured warmup cycle, followed by 20 measured sequential cycles of
all six cases in `comparison-workload.json` (120 requests). All six assertions use
an independent order-subtotal formula. Classify the original incident assertion
as recovery and the other five as regressions. Report every failed/unknown check.
Missing responses are unknown, never evidence of recovery. No retries hide failures.

Resource acceptance is deliberately scoped: cumulative application-container CPU
between before/after kubelet snapshots must be <= 5 CPU-seconds; both sampled
working-set observations must be <= 32 MiB. Missing/reset counters, changed pod UID,
or absent samples are unknown. This measures an interval and endpoint memory,
NOT peak memory, CPU throttling, sustained throughput, or control-plane resources.
The pod is limited to 250m CPU and 64Mi memory. Passing these checks provides no
production-safety or universal performance guarantee.

Preserve per-request durations and report median/p95 as descriptive observations
only. One fresh cluster per candidate is not a statistically rigorous performance
comparison; no speedup claim is allowed. Broader workloads/peak sampling come later.
A candidate can pass only after the baseline was verified, all measured behavioral
checks pass, and the defined sampled resource checks pass. Incomplete evidence is
inconclusive. Human approval remains required before any optional external apply.
