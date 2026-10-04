# Milestone 4: lifecycle slice verified evidence

Observed 2026-10-04 UTC on Raj's Mac. Tested code revision:
`374543e4de84ea7a442e33e3636788fa98b95c0b`. Final documentation/ignore updates do
not change this tested implementation. This is the first bounded lifecycle slice;
remaining lifecycle work has not been silently marked complete.

[Recorded real-check summary](evidence/milestone-4-lifecycle-summary.json) and
[implementation walkthrough](milestone-4.md).

| Real isolated-cluster check | Outcome | Child exit | Cleanup | Whole-run duration |
| --- | --- | ---: | --- | ---: |
| 8-second deadline during baseline cluster creation | Timed out | 130 | Complete | 9.088 s |
| SIGTERM with candidate port-forward active | Canceled | 130 | Complete | 38.958 s |
| Injected failure after candidate manifest apply | Failed | 70 | Complete | 40.288 s |
| Successful fixed-candidate trial | Completed | 0 | Complete | 47.330 s |

Durations include setup and cleanup; the SIGTERM row is **not cancellation latency**.
All four real cases terminated gracefully; none required the supervisor's forced
KILL escalation. Forced KILL was verified by the local process-group unit fixture.

After every case, the harness verified the owned process group no longer existed,
no owned kind node container remained, generated kubeconfigs were gone, and an
independently labeled stopped sentinel container kept the same ID and `created`
state. The sentinel was never started. The harness then removed only that exact
owned sentinel ID after verifying its ownership label. Final Docker inspection
found no validation clusters or sentinel remaining. Docker context stayed
`desktop-linux`, and no default kubeconfig or unrelated resource was modified.

The normal candidate still passed recovery, all regressions, and the existing
scoped CPU/two-memory-sample budget across 120 measured requests. Deadline,
cancellation, and injected failure never became successful candidate completion.
The creation-timeout log confirms a partially created node was deleted.

## Local tests and checks

- Race-enabled uncached suite, formatting, and `go vet` passed after implementation.
- Lifecycle statement coverage: 86.7%. CLI main unit coverage remains 0%; the new
  supervisor CLI was exercised by the four real cases above.
- Tests cover normal/failing/start-failing children; canceled-before-start and
  canceled-running work; independent cleanup deadlines; forced group termination;
  failed-parent orphan cleanup; ownership mismatch; unavailable inventory; failed
  deletion; and preservation of unrelated local state/symlinks.
- An initial failed-parent test exposed descendants holding inherited output pipes
  open. Bounding `exec.Cmd.WaitDelay` corrected it before final passing validation.
- Shell syntax, Python compile check, and `git diff --check` passed.
- Hosted GitHub CI was not run; no external AI API, paid service, permission
  expansion, public publication, or production activity occurred.

## Limits and next step

Real cleanup completed via the shell traps, followed by the supervisor's independent
ownership/removal verification. Cleanup-command failure and ownership mismatch were
injected with local fake-command tests, **not by disrupting the real Docker daemon**.
No real daemon outage, host sleep/crash, supervisor SIGKILL, forced real-cluster KILL,
full supervised three-candidate matrix, worker-pool load, or long soak was tested.

The supervisor runs one local process group. It has no persistent job store,
restart reconciliation, durable leases, distributed controller, or concurrency
scheduler. Cooperative exact-name/label ownership is not a defense against a
malicious peer with Docker access. Abrupt supervisor death can leave resources;
restart/cleanup reconciliation is the next lifecycle increment before agent work.

Ignored local bundle: `evidence/lifecycle-validation-20261004T171508Z/`, with paths
to four `evidence/lifecycle-<run-id>/` directories. These preserve supervisor results,
process logs, child trial evidence, cleanup checks, and source identity context.
Reusable build artifacts are about 52 MiB, tools 9.5 MiB, and total evidence 1.9 MiB,
plus existing Go/kind/BuildKit caches. No active project test workload remains.

Stop here for parent review/pacing. Next: the remaining milestone 4 restart
reconciliation and bounded worker-concurrency slice; no model/MCP integration yet.
