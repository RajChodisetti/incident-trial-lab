# Bounded local workers

This increment adds a single-use, in-memory candidate queue. Default concurrency
is **one**, maximum concurrency **two per batch**, and at most eight queued jobs.
It is not a distributed scheduler or a host-wide admission controller.
See [verified results and gaps](milestone-4-concurrency-evidence.md).

```sh
./scripts/run-batch.sh -variants fixed,misleading
./scripts/run-batch.sh -workers 2 -variants fixed,fixed
```

The existing pinned candidate archives must exist. Each job starts its own journaled
supervisor, run ID, process group, kind cluster, kubeconfig, evidence directory,
locks, and loopback port. The baked candidate image/digest is shared read-only;
trial state is not shared. Worker slots remain occupied until supervisor cleanup
returns. Results preserve input order even when completion order differs.

The CLI prints a unique batch evidence directory. To request cancellation of a
single registered job, create a regular file at `<batch-directory>/cancel/job-1`.
The monitor rejects symlinks and calls cancellation only inside that pool's job
registry. Unknown/finished IDs cannot cancel another job. Parent SIGINT/SIGTERM
cancels active/queued jobs; queued cancellation starts no supervisor/resources.
The executor signals only its own live supervisor PID, allowing that supervisor's
normal process-group/owned-resource cleanup. If supervisor termination or terminal
evidence is unavailable, report `needs_recovery` rather than success; use its run ID
with the existing explicit reconciler. Batch-wide SIGKILL is not certified here.

Lifecycle status and business verdict are separate. A completed worker can carry
a failed candidate verdict; canceled or missing business evidence is inconclusive.
One canceled job makes the batch CLI exit nonzero even when its peer succeeds.
This is an experimental outcome, not a reason to hide successful peer evidence.

Supervised ports are chosen with a local ephemeral-port probe and recorded in the
run journal. The reservation is released before kubectl binds, so a competing bind
can still occur. Own port-forward readiness is required before probing; collision
fails the trial and cleans up. Legacy direct shell runners default to port 18080.
Existing early-crash refusal remains: no node snapshot means no guessed deletion.

## Verification

```sh
go build -o .build/trial-run ./cmd/trial-run
go build -o .build/trial-batch ./cmd/trial-batch
go build -o .build/trial-recover ./cmd/trial-recover
python3 scripts/check-concurrency.py
```

The real check requires two nodes to be simultaneously live with distinct run IDs,
container IDs, and ports. It cancels job-1 via its own batch inbox while job-2 must
complete and pass its business assertions. Both must verify cleanup and support
idempotent recorded reconciliation. A separately labeled stopped sentinel must
remain unchanged before its own exact-ID teardown. No daemon restart is used.

Unit/race tests check the maximum active count, slot reuse, default one worker,
unknown/finished cancellation, parent cancellation, duplicate jobs, and cancellation
before launch. Remaining limits include multiple independent batches, batch-parent
crash recovery, long soaks, fairness/backpressure beyond eight jobs, and host failures.
