# Verified bounded-worker slice

Observed 2026-10-04 UTC on Raj's Mac. The real paired check exercised source
`6bf89d3`. A subsequent unit/race-tested guard prevents partial passing evidence
from accepting canceled or cleanup-incomplete jobs; it does not alter the observed
paired outcomes below. The queue default is one worker, maximum two per batch,
with a maximum of eight queued jobs. There is no distributed scheduler.

[Recorded paired summary](evidence/milestone-4-concurrency-summary.json),
[worker walkthrough](milestone-4-concurrency.md), and [MVP blockers](mvp-readiness.md).

| Job | Lifecycle | Business verdict | Cleanup |
| --- | --- | --- | --- |
| job-1, fixed image | Canceled by its own inbox request | Inconclusive | Verified |
| job-2, fixed image | Completed | Passed | Verified |

Both node containers were confirmed live simultaneously before cancellation.
Run IDs were `7e49370a70f2ba4f` and `83e0cd5f34c42f32`; ports were 59488 and 59489.
Container IDs and journals were distinct. Canceling job-1 did not cancel job-2:
its peer completed all 120 measured recovery/regression requests and passed the
existing scoped resource checks. The batch exited nonzero because cancellation
is a real outcome, not an all-success result.

Both supervisors verified cleanup; explicit repeated reconciliation also passed
for each run. No node containers or generated kubeconfigs remained. A separately
labeled stopped sentinel kept its `created` state, then only its exact owned ID
was removed. Shared Docker stayed running with context `desktop-linux`.

## Validation

- Final uncached race suite, vet, formatting, shell syntax, and diff checks passed.
- Worker statement coverage: 93.0%; CLI main unit coverage is 0%. Real execution
  covered batch scheduling, targeted cancellation, dynamic ports, cleanup, and
  compatibility with recorded reconciliation.
- Unit checks verify active-count bound, slot reuse, default one worker, unknown/
  finished job refusal, parent cancellation, duplicate IDs, queued cancellation
  without execution, and inconclusive canceled/cleanup-incomplete partial passes.
- Existing early-crash ownership refusal is preserved. Recovery does not delete
  unrecorded/replacement nodes to make a queue result appear successful.
- Historical lifecycle/recovery check scripts now recognize assigned ports instead
  of assuming 18080; their full real suites were not rerun in this increment.
- No AI/provider calls, new credentials/permissions, paid CI, public publication,
  shared-daemon restart, or production/unrelated resource changes.

## Resource scope and remaining limits

Docker reported 2 CPUs and 3.827 GiB RAM with no active containers before the check.
Two startup node snapshots showed 914.3 MiB and 1023 MiB respectively. These are
whole-control-plane snapshots, not peak memory, application budgets, or a throughput
benchmark. Application limits and scoped measurement caveats remain unchanged.

No batch-parent SIGKILL, long soak, independent simultaneous batches, OS-wide
admission, queue restart persistence, fairness study, peak/throttling benchmark,
or host failure was tested. Concurrency is bounded within one batch; multiple
independent CLI processes are not a host-wide limit. The ephemeral-port reservation
has a bind race; a collision fails safely rather than probing an unrelated service.

Full ignored evidence: `evidence/batch-78d79fde5f4b2959/`, linking the two recorded
run directories. Journals, pinned images, raw observations, worker logs, cleanup,
recovery reports, and cancellation outcomes are preserved. Build cache/artifacts
are about 60 MiB, tools 9.5 MiB, total evidence 2.8 MiB, plus previous image/Go caches.
No active project workload remains.

## Before thin MCP / agents

Essential: bind candidate trials to fresh experiment baseline provenance; define
and test scoped typed tools/results; evaluate a bounded offline investigation/
proposal/test/revision loop; resolve model credentials only when live model work
starts. The current static variants are not an autonomous AI agent.

Deferred for the local demo: creation-to-snapshot crash closure, host-wide admission,
durable/distributed queues, outage/soak coverage, broader workloads, peak metrics,
commercial telemetry, production adapters, and public release. Keep documented
refusal/uncertainty instead of risky cleanup or guaranteed-safety claims.

Stop for parent review/pacing. Next narrow work is experiment/evidence binding and
the minimal deterministic tool contract, not more distributed scheduling or API use.
