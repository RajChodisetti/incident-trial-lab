# Local baseline provenance gate

A verified baseline is now an experiment prerequisite, rather than a caller's
`baseline_verified` boolean. This is an additive gate over the existing local
supervisor and candidate scripts; the legacy standalone paths produce **unbound**
evidence and must not be used as an agent-facing acceptance shortcut.

Run `./scripts/run-bound-comparison.sh` from the project root after the pinned
baseline/candidate archives have been built. It starts a fresh supervised baseline,
registers the completed run, reserves one attempt per candidate, runs each candidate
in a separate new cluster using the registered baseline proof, and binds receipts.
The script expects ineffective/overfit candidates to fail and the general fix to
pass. It also verifies candidate and baseline replay refusal. Interrupting the
wrapper forwards termination to the active supervisor and preserves evidence.

The underlying `cmd/experiment` has five ID-scoped operations:

- `register -baseline-run <id>` accepts one completed, clean supervised baseline
  whose finish time is at most 10 minutes old and not in the future.
- `inspect -experiment <id>` revalidates the specification and baseline evidence.
- `reserve -experiment <id> -variant fixed|ineffective|misleading` records a new
  attempt and returns the exact baseline proof path for the existing supervisor.
- `verify -experiment <id> -attempt <id>` rechecks a live stored receipt against
  its unchanged evidence and binding markers; expired receipts remain historical
  files, but cannot serve as current experiment proof.
- `bind -experiment <id> -attempt <id> -candidate-run <id>` verifies a completed,
  clean, newly started candidate run and publishes its receipt once.

Experiment IDs and attempt IDs are random resource identifiers, not credentials.
An experiment expires 30 minutes after registration. Every load/bind checks the
same fixture bytes, workload bytes, assertion definitions, protocol/kind config,
deployment specification and candidate image lock. Archived applied manifests
must match that specification, with only the candidate image replacement allowed. Baseline cases must be included
unchanged in the comparison assertions. A changed scenario needs a fresh experiment.

Both baseline and candidate image manifest bytes must hash to their pinned digests.
The pod's requested image, runtime config digest and runtime image/index content
must agree with that manifest. The resource snapshots must belong to that pod.
Baseline responses must reproduce the expected fault and pass the controls.
Candidate warmup and all 120 measured responses are rechecked against the exact
case identities, roles and values; aggregate verdicts and resource budgets are
recomputed. Behavior timestamps must lie inside the candidate lifecycle interval.
The candidate must have started after its reservation, and its copied baseline
proof must exactly match the registered baseline evidence. Receipts hash the
lifecycle, workload, response, image, pod and resource evidence and identify the
experiment, baseline, candidate, attempt and image pins.

Completed runs with valid unknown observations can bind an **inconclusive** receipt;
known assertion failures remain **failed**. Canceled, timed-out, failed or unclean
lifecycles cannot bind an accepted receipt. Missing/malformed/mismatched evidence
is refused; its original evidence remains for diagnosis. Refusal does not rewrite
the original trial outcome or invent successful recovery.

OS locks serialize receipt consumption; concurrent conflicts return a refusal.
Exclusive markers prevent reusing a baseline, candidate run or attempt. Marker
files and their directories are fsynced. A crash between the marker and receipt
writes can leave a consumed but unreceipted attempt: fail closed, preserve files
for review, and run a fresh experiment. Automated repair of that gap is deferred.

This is consistency checking within a trusted local owner's workflow, not signed
attestation. A person able to rewrite all project files, journals and clocks can
forge evidence. Adversarial tampering, host-wide admission, clock rollback,
durable distributed scheduling and production safety are outside this increment.
Resource coverage remains CPU interval plus two memory endpoints, not peak memory;
latencies are descriptive, with no improvement claim. No AI API or apply action is
implemented.

Next paced increment: define typed local MCP contracts for the single registered
scenario, experiment creation, bounded candidate start/list/read/cancel by owned
IDs, with evidence treated as untrusted data. Expose lifecycle, business verdict,
cleanup and measurement coverage separately. No arbitrary shell, external apply,
new credentials or live-model work is needed for that contract.
