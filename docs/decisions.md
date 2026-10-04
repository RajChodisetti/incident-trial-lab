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
