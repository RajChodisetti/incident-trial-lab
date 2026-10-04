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
