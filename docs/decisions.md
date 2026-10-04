# Initial decisions

1. **Private personal repository.** Verified account: `RajChodisetti`, GitHub user
   ID `110764922`, type `User`. Use `incident-trial-lab` as a neutral working name.
   Visibility changes require a public-release decision.
2. **Standard library first.** Go 1.26.2 is installed locally. No framework,
   Kubernetes client, agent SDK, or telemetry dependency until the relevant
   milestone needs it.
3. **Incomplete evidence is inconclusive.** An incomplete trial, empty checks, an
   unknown check, or malformed status must never appear as a pass. A complete set
   with a failed check fails; a complete set with only passing checks passes.
   This is a generic evaluator, not proof of recovery. The later trial layer must
   require baseline reproduction, recovery, regression, and budget checks.
4. **No cluster access in bootstrap.** Inspection covered client versions only.
   Docker daemon is unavailable, kind and Helm are absent, and no cluster API was
   contacted. Installation/runtime setup waits for milestone 2.
5. **Manual hosted CI until no-cost allowance is verified.** Existing GitHub auth
   can access the personal identity, but the billing-usage endpoint returned 404
   and requested additional `user` scope. No scope expansion was performed.
   Keep the workflow dispatch-only; run its identical checks locally. Do not claim
   GitHub CI passed when it was not run.

Environment observed on 2026-10-04 UTC: macOS arm64; Git 2.44.0;
Go 1.26.2; Docker client 29.6.2 (daemon stopped); kubectl client 1.36.1,
kustomize 5.8.1; kind/Helm absent. Initial sandboxed GitHub checks reported
connectivity/auth failure; a permitted network request verified the existing
login successfully. No credentials were generated or changed.


## Milestone 2 decisions

- Use a deliberately seeded discount-rounding defect and integer cents. The
  business formula and two control requests make reproduction falsifiable.
  The checker reports `baseline_reproduced`; it never labels the faulty service
  recovered merely because the reproduction command succeeds.
- Raj approved Docker startup and kind installation. Official kind v0.33.0 is
  checksum-pinned under ignored `.tools/`, without sudo or global PATH changes.
  Kubernetes 1.36.4 matches the installed kubectl minor and is digest-pinned.
- Package a static arm64 Go binary in scratch. Go 1.26.2, BuildKit's image digest,
  normalized image/layer timestamps, and the application manifest digest pin the
  local artifact. No registry push or external application image pull occurs.
  Timestamp normalization applies to artifacts only; Git dates are untouched.
- Use one project-specific disposable BuildKit builder because the existing
  Docker driver cannot export OCI. Do not switch the default builder or modify
  Docker settings. Delete the builder on exit.
- Require a fresh dedicated cluster and explicit kubeconfig/context on every
  kubectl operation. Preserve incomplete-run evidence, automatically clean up
  owned resources, and remove the temporary kubeconfig (which contains local
  cluster credentials) from the evidence directory after teardown.
- Containerd reports an imported OCI index as runtime image identity. Verify
  that this index references the pinned child manifest and its config matches
  the running container, instead of assuming those two digests are identical.
- No hosted CI, candidate fix, performance claim, AI integration, or commercial
  telemetry is part of this milestone.


## Milestone 3 decisions

- Freeze milestone 2's baseline code and image digest. Keep explicit candidate
  variants in a separate package and bake each into a separate binary/image.
- Include both a no-op proposal and an overfit proposal. Separate recovery from
  regression assertions so repairing one incident input cannot hide a regression.
- Start sequentially with a fresh full cluster per candidate. Concurrent workers,
  durable lifecycle/restart handling, and fault injection belong to milestone 4.
- Define workload and CPU/sampled-memory budgets before running candidates. Use
  application kubelet counters with stable pod UID, monotonic times, and a sampling
  window that brackets all requests; missing/reset/stale data is inconclusive.
- State resource coverage narrowly: cumulative CPU between snapshots and memory
  at two endpoints. Peak memory, throttling, sustained throughput, and statistical
  performance claims are untested. No claim of production safety follows.
- Remain deterministic until the trial system works. No API/model dependency,
  credential reuse decision, or hosted paid CI is needed for this milestone.


## Milestone 4: first lifecycle slice

- Supervise the existing deterministic runners rather than building a scheduler
  or distributed state machine in one increment. Run one child process group with
  explicit execution, termination-grace, and independent cleanup deadlines.
- Unique run-derived cluster/lock/evidence names plus exact kind-label checks define
  cooperative local ownership. Do not claim protection against a malicious peer
  sharing Docker access. Refuse broad selectors and preserve mismatched resources.
- Cancellation/timeout are outcomes, not successful candidate verdicts. Cleanup
  failure makes normal completion fail; canceled contexts never prevent a fresh
  cleanup attempt. Preserve inaccessible/mismatched/remaining-resource evidence.
- Bound descendant-held stdout/stderr pipe waits with `exec.Cmd.WaitDelay`; an
  initial failure test exposed why waiting only for a shell was insufficient.
- Real interruption validation uses an owned stopped sentinel to verify unrelated
  resource preservation. No global daemon restart, broad prune, or security change.
- SIGKILL/crash recovery, durable ownership reconciliation, worker concurrency,
  whole-host failures, and long-running soak tests remain outside this increment.
  Complete further lifecycle work before introducing model/API integration.


## Milestone 4: explicit crash reconciliation

- Persist a small atomic/synced run record before launch, then its child identity;
  hold a local OS lock so recovery refuses a live supervisor even when PID files
  are stale. Never automatically resume an interrupted candidate.
- Record exact node IDs/names/labels after creation. On restart, refuse deletion
  of unrecorded or replacement nodes; retain incomplete cleanup if proof is absent.
- Compare child start-time/command fingerprints including the run token before
  signaling. Refuse changed identity and unresolved orphan groups.
- A real hard-crash test exposed kind's kubeconfig mutation failure with a special
  device. Use a fresh owned regular temporary file, preserving command errors and
  verification instead of treating disappearance alone as a successful command.
- Cleanup success cannot reconstruct the interrupted trial outcome: unknown and
  inconclusive remain explicit. Repeating reconciliation must preserve that result.
- Do not simulate shared-daemon outage, add concurrency, or start model/API work in
  this slice. Creation-to-snapshot gaps and host/filesystem failure remain limits.


## Milestone 4: bounded local workers

- Default one worker, hard maximum two per in-memory batch, maximum eight jobs.
  Preserve ordered results and occupy a slot through verified supervisor cleanup.
- Per-job contexts and live supervisor PID signaling isolate cancellation. Parent
  cancellation also marks queued work without launching new resources.
- Use separate recorded loopback ports/run identities; require own forward readiness
  before workload probes. A competing bind fails safely rather than sharing a port.
- Separate lifecycle completion from business verdict; canceled/cleanup-incomplete
  partial passes remain inconclusive. Failed candidate evidence is not a failed
  worker process by itself, and canceled jobs are not hidden in aggregate success.
- Record a real paired cancellation/peer-success check with independent node IDs and
  recovery compatibility. Do not equate max-two per batch with host-wide admission.
- Prioritize fresh experiment provenance and a narrow deterministic tool contract
  before thin MCP/agent work; defer broad distributed/outage hardening for the demo.


## Thin local MCP surface

- Pin the official Go SDK; use stdio with typed input/output schemas and no MCP
  listener, model calls or account configuration.
- Accept only the supported scenario, pinned candidate presets and server-owned
  IDs. Never expose shell, file-path, arbitrary image or apply parameters.
- Return asynchronous jobs promptly; cap eight total session jobs and one/two
  active supervisors, retaining a slot through cleanup and provenance binding.
- Keep lifecycle, historical business verdict, cleanup and current proof eligibility
  distinct. Return evidence as untrusted data, with named bounded sections.
- Explicit owned-job cancellation and process shutdown use the existing supervisor;
  request cancellation alone does not undo already-admitted work. List all owned
  jobs to recover IDs after response loss. Clients need a graceful shutdown allowance.
- Preserve the local trust/restart/host-wide limitations. Next evaluate an offline
  investigation loop; preset choice alone does not establish AI engineering depth.
