# Crash reconciliation: second bounded lifecycle slice

This slice adds an explicit local recovery command, not a restart scheduler or
worker pool. See [verified evidence and limits](milestone-4-recovery-evidence.md).
A supervisor SIGKILL bypasses all defers: clean resources later from
persisted ownership without inventing the interrupted trial's business outcome.

```sh
./scripts/run-trial.sh -mode candidate -variant fixed -timeout 120s
# After an interrupted supervisor, use its printed run ID from the project root:
go build -o .build/trial-recover ./cmd/trial-recover
.build/trial-recover -run-id <recorded-run-id>
```

The supervisor atomically saves `record.json` before launch and updates it with
child process-group identity immediately after start. File and parent directory
are synced. A per-record OS lock is held by a live supervisor or reconciler; the
OS releases it on SIGKILL. Recovery refuses an active lock rather than infer
liveness from a stale PID file. A startup journal failure stops the child.

The child command line carries the run token. Recovery compares the recorded
start-time/command fingerprint before signaling that process group. A changed
identity is refused; an absent leader with an existing group is unresolved rather
than a reason to signal unrelated processes. This is cooperative local ownership,
not a security boundary against malicious processes sharing the same user account.

After kind creation, the child atomically records the node's full container ID,
name, and exact kind-cluster label before applying the application. Recovery
validates those snapshots under the requested run's directory, and refuses a
replaced, unrecorded, label-mismatched, or unexpectedly named container. It never
scans arbitrary jobs, guesses from age, or prunes unrelated Docker resources.
A freshly owned regular temporary kubeconfig is used for kind deletion; kind
cannot safely update a special-device target such as `/dev/null`.

Only verified process termination and verified resource removal allow local
kubeconfigs and empty run locks to be removed. Repeated recovery is idempotent.
Snapshots, process logs, the original record, and reconciliation evidence remain.
Unknown process/resource inventory remains incomplete, not successful cleanup.

An interrupted record without the supervisor's final result retains
`original_outcome: unknown` and `business_verdict: inconclusive`, even if cleanup
succeeds or partial child observations look promising. Recovery cleans resources;
it does not resume candidate execution or certify recovery of the application.

## Real check

```sh
go build -o .build/trial-run ./cmd/trial-run
go build -o .build/trial-recover ./cmd/trial-recover
python3 scripts/check-recovery.py
```

The harness checks active-run refusal, supervisor-only SIGKILL followed by recovery
of its still-owned process group, and supervisor plus owned-group SIGKILL leaving
an actual recorded kind container for restart cleanup. Both repeat reconciliation,
retain uncertainty, remove generated kubeconfigs, and preserve a separately labeled
stopped sentinel before its exact owned-ID teardown. Shared Docker is never stopped.

If the supervisor crashes during cluster creation **before a node snapshot exists**,
that container is unrecorded and automatic deletion is refused. The narrow journal
is not a complete write-ahead protocol for every Docker operation. Host power loss,
filesystem corruption, adversarial PID/label tampering, multiple worker recovery,
and daemon outage are not certified. Durable worker scheduling and broader startup
reconciliation remain later work. Do not introduce MCP/AI before that review.
