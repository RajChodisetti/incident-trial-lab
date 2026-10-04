# Lifecycle robustness: first bounded increment

This increment supervises a **single local execution**. It adds cancellation,
process-group termination, independent bounded cleanup, and explicit ownership
checks. It does not add a durable scheduler, worker pool, restart recovery, or
multi-host orchestration. See [verified checks and gaps](milestone-4-evidence.md).

```sh
./scripts/check.sh
./scripts/run-trial.sh -mode comparison -timeout 10m
# Or a single existing candidate trial:
./scripts/run-trial.sh -mode candidate -variant fixed -timeout 90s
```

The existing pinned baseline/candidate OCI archives must exist. This supervisor
supports macOS/Linux process groups; Windows is unsupported. Compilation by the
wrapper occurs before the supervised execution deadline starts.

`cmd/trial-run` creates a random resource identifier (not a credential), writes a
private evidence directory, and launches the shell runner in a separate process
group. The runner receives unique run-derived cluster names, evidence paths, and
lock names. An existing cluster is refused. The default kubeconfig is never used.

SIGINT/SIGTERM or deadline expiry sends TERM to the entire owned group, then KILL
if it has not exited within the configured grace period. The supervisor waits for
the group leader, kills remaining descendants in that group, and bounds output
pipe waits with `exec.Cmd.WaitDelay`. A child failure or canceled/timed-out run is
never classified as successful completion.

Cleanup uses a fresh context, independently of the canceled trial context. It
inventories only the four allowlisted cluster names for this invocation, verifies
each found container's exact kind ownership label before deletion, and checks
that no labeled containers remain. Unavailable Docker inventory, label mismatch,
failed deletion, or surviving resources is incomplete cleanup, not success. Each
command and the total cleanup phase have deadlines. If cleanup is incomplete,
local credentials/locks are retained under ignored evidence for diagnosis.
After verified cleanup, remove only empty run-derived locks and regular kubeconfig
files inside this invocation's evidence root; never follow symlinks.

The ownership model assumes cooperative local users. Random run names plus exact
labels prevent accidental cross-project cleanup; they are not a security boundary
against a malicious peer with Docker access. No broad prune or arbitrary container
ID is accepted. The supervisor itself stays outside the child process group.

Defaults: execution 10m, termination grace 20s, cleanup 45s, individual cleanup
command 15s. A nonresponsive host/kernel may still prevent timely process exit;
SIGKILL or crash of the supervisor itself can bypass cleanup. Those cases require
future restart reconciliation rather than a claim of guaranteed teardown.

## Reproducible checks

```sh
GOCACHE=/tmp/incident-trial-lab-go-cache go build -o .build/trial-run ./cmd/trial-run
python3 scripts/check-lifecycle.py
```

The real-check harness creates one labeled **stopped** sentinel container, tests
cluster creation timeout, TERM while port-forwarding, injected failure after
apply, and successful candidate completion. It checks process-group disappearance,
resource removal, credential removal, and sentinel preservation after every case.
Finally it removes only its own exact sentinel ID after verifying its ownership
label. The test-only `TRIAL_TEST_FAIL_AFTER_APPLY=1` hook exits after an owned local
manifest was applied; it is off by default.

Unit/race tests cover graceful and forced termination, descendant-held output
pipes, failed-parent orphan cleanup, independent cleanup deadlines, inventory/
ownership/deletion failures, and preservation of unrelated local state/symlinks.

Next is the remaining lifecycle increment: restart reconciliation, bounded worker
concurrency, and stronger cleanup-failure recovery. No agent/model/MCP integration
or production safety claim follows from this slice.
