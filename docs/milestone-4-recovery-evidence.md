# Verified SIGKILL and restart reconciliation

Observed 2026-10-04 UTC on Raj's Mac. Tested implementation revision:
`e1a91be9c2b9acc598b8f4a885b08c6c0c4450dd`. Later changes are documentation only.
Repository default branch is `main`; visibility remains private.

[Recorded summary](evidence/milestone-4-recovery-summary.json) and
[recovery walkthrough](milestone-4-recovery.md).

| Real check | Restart behavior | Original trial outcome | Cleanup |
| --- | --- | --- | --- |
| Supervisor SIGKILL with port-forward active | Verified recorded child identity; terminated its owned group | Unknown / inconclusive | Complete |
| Supervisor and owned process-group SIGKILL | Deleted the actual leftover node using its recorded full ID/name/label | Unknown / inconclusive | Complete |

Both checks refused recovery while the original supervisor still held the run
lock. After the crash, no normal `lifecycle.json` result existed; recovery did not
infer success from partial child evidence. Both then repeated recovery successfully
and retained the original unknown/inconclusive outcome. Final journal states were
`reconciled`, process termination was verified, and generated kubeconfigs were gone.

The second check's verified leftover node was
`b9adeec0fb207dc56c6cbd4461f18bb16614e0d51f2a3477423728b9151877ea`, in cluster
`itl-candidate-fixed-4fa24d4cbad1b63b`. The reconciler found that exact recorded ID,
verified its name and kind label, deleted it, and independently confirmed removal.
This exercised real restart cleanup, not just a fake-command unit test.

A separately labeled stopped sentinel kept its same `created` state after each
case, then the harness removed only its exact owned ID after verifying its label.
Final inspection found no validation cluster or sentinel containers remaining.
Docker stayed running throughout; its context stayed `desktop-linux`. No shared
daemon restart, broad prune, default kubeconfig change, or unrelated resource change.

## Checks and correction evidence

- Uncached race suite, formatting, `go vet`, shell syntax, and diff checks passed.
- Lifecycle statement coverage: 77.7%; CLI main unit coverage remains 0%. Both new
  recovery/supervisor CLI paths were exercised by the real crash/restart checks.
- Tests cover atomic journal round-trip, active-lock exclusion/release, malformed
  records, ID mismatch, symlink refusal, startup journal failure, recorded-ID
  mismatch/replacement refusal, exact-label verification, and idempotent absent
  resources, in addition to previous cancellation/deadline/process-group tests.
- An initial hard-crash check exposed kind's mutation failure with `/dev/null` as
  kubeconfig target. The node disappeared, but the command failed, so cleanup was
  correctly incomplete. The fix uses an owned regular temporary file; final real
  checks passed. The retained incomplete record was explicitly retried and is now
  reconciled. Initial failure logs remain in ignored local evidence.
- Hosted CI remains unrun. No public publication, paid action, permission expansion,
  external API/model dependency, or production access was introduced.

## Limits

The supported case is a recorded local run with intact intent, process identity,
and node ownership snapshots. Missing/corrupt records or unrecorded/replacement
nodes are refused rather than deleted. A crash during creation before the node
snapshot leaves a known gap: automatic cleanup cannot certify ownership there.
Pre-journal runs are not automatically migrated or guessed from container age.

Host power loss, filesystem corruption, daemon outage, adversarial PID/label
manipulation, long soaks, multiple workers, and distributed leases were not tested.
No worker concurrency, automatic execution resume, MCP, or AI was added. Process
fingerprints and local locks assume a cooperative host; this is not a security
boundary against a malicious peer sharing the user/Docker account.

Full ignored evidence: `evidence/recovery-validation-20261004T183727Z/`, linking
run directories `lifecycle-698a8a1fe2699045` and `lifecycle-4fa24d4cbad1b63b`.
They preserve synced records, ownership snapshots, process logs, active-run refusal,
reconciliation results, and idempotence checks. Previous failed-attempt evidence
is under `evidence/recovery-validation-20261004T183314Z/`; its incomplete record
`lifecycle-07c113e2d2413b90` was subsequently reconciled explicitly.

Reusable build artifacts are about 56 MiB, tools 9.5 MiB, and total evidence 2.4 MiB,
plus existing Go/kind/BuildKit caches. No active project trial workload remains.

Stop for parent review/pacing. Next: address creation-to-snapshot reconciliation
and bounded worker concurrency in another narrow lifecycle increment, before MCP
or model integration. Do not describe this as a complete distributed job engine.
