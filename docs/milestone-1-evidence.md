# Milestone 1 evidence

Recorded 2026-10-04 UTC. This document describes local checks; remote SHA and
workflow status are verified separately after pushing the final commit.

- Repository: https://github.com/RajChodisetti/incident-trial-lab (created private).
- Local path: `/Users/rajchodisetti/Documents/Codex/2026-10-03/task/incident-trial-lab`.
- `go test -race -count=1 -cover ./...`: passed, 100% statement coverage for the
  small `internal/evaluation` package; 12 table-driven cases. Coverage is not a
  claim of trial-engine correctness or performance.
- `go vet ./...`: passed.
- Formatting check through `./scripts/check.sh`: passed.
- `git diff --check`: passed.
- Hosted CI: not run. Workflow is manual-only because included Actions allowance
  cannot be verified with current permissions. No additional scope was requested
  or granted; no paid resources were started.
- No Kubernetes integration, benchmark, incident reproduction, or recovery trial
  was run; those capabilities are not implemented.

To repeat the local checks, run `./scripts/check.sh` from the repository. In the
restricted Codex sandbox, use `GOCACHE=/tmp/incident-trial-lab-go-cache` if the
default Go cache path is outside writable roots.

Next milestone: specify and reproduce one pinned synthetic baseline on a dedicated
local kind cluster. Prerequisites: Docker Desktop running, kind installed, and
explicit continuation. Public release, AI credentials, and hosted CI spending
remain decisions for later; they do not prevent local development.
