# Local MCP verification evidence

Verified on 2026-10-04 UTC in the working tree based on `4a56586`. The service images,
scenario assertions, supervisor, provenance gate and legacy worker paths were
unchanged. The new surface uses the official Go SDK v1.8.0 with Go module checksums.
No AI API, account access, paid service, hosted CI or public publication was used.

`./scripts/check.sh` passed formatting, all package race tests and `go vet`.
Toolserver unit coverage is 55.5%; the real subprocess/backend path is exercised by
the actual stdio comparison, not fully covered by unit instrumentation. CLI entry
points have no unit coverage. `git diff --check` passed.

The offline SDK client exercised real protocol discovery and all eight typed tools
using **simulated backend results** on protocol revisions 2026-07-28 and 2025-11-25.
Inputs/outputs were independently validated against the advertised JSON Schemas.
Tests refused extra properties, path/command arguments, unsupported IDs and variants,
unowned job cancellation, invalid worker counts and excess submissions. They tested
max-two execution, queued cancellation, peer completion, shutdown, evidence section
selection and changed-proof eligibility separate from historical outcome. Backend
file checks refused symlinks, oversized files and malformed/trailing JSON.

`.build/mcp-check` also passed discovery, schema validation and invalid-call refusal
against the actual stdio executable without creating any trial resources. After
adding all-owned-job listing for response-loss recovery, the final offline suite and
this protocol-only check passed again; the trial runner/binding path was unchanged.

`.build/mcp-check -real` negotiated protocol **2026-07-28** and produced actual kind
evidence through tool calls:

| Operation | Owned ID | Result |
| --- | --- | --- |
| Baseline | trial `75a1d33196ccb128`, run `d42f8ca9a50699ca` | Fault reproduced; controls passed; registered fresh experiment `00590544eb3f9b5a` |
| Fixed candidate | trial `201eaafc7aedfbce`, run `a09f4040ba978a0a` | Completed, cleanup verified, provenance bound/current, verdict passed |
| Queued misleading candidate | trial `4c459f1fe8b87986` | Canceled before start; no run ID, supervisor log, or cluster; verdict inconclusive |

The fixed image passed recovery, all regressions and 120 measured requests after
warmup. Its sampled CPU delta was 11,472,000 nanoseconds; maximum of the two memory
samples was 4,882,432 bytes, within the predefined budgets. These are descriptive
scoped observations, not peak metrics or measured improvements. `get_evidence`
returned verified baseline observations, candidate behavior, lifecycle and provenance
sections. `compare_trials` preserved both the passed fixed job and canceled job;
`list_trials` returned all three jobs.

Both actual lifecycle journals reported complete cleanup. Independent read-only
Docker checks of their exact recorded cluster labels found **zero remaining
containers**. No unrelated containers were inspected or removed. Raw evidence is
under `evidence/mcp-session-ee0e958311374394`, the two `evidence/lifecycle-<run-id>`
directories and `evidence/experiment-00590544eb3f9b5a`. Compact tool/client, receipt,
resource and cleanup evidence is committed in
[mcp-real-summary.json](evidence/mcp-real-summary.json). The server exited after the
client closed. Remote SHA and CI status are verified separately after pushing.

This slice did not run a new actual max-two MCP comparison, active-cancellation
matrix, hard-kill/disconnect cleanup matrix, durable queue or host-wide admission
test. Those boundaries remain explicit; prior engine tests passed unchanged.
No live LLM, autonomous patch generation or investigation loop is implemented.

Next paced increment: offline agent-loop contract/evaluation with evidence-grounded
proposals, failed/overfit/unknown preservation, bounded retries and reviewable traces.
Review this thin tool surface before live-model credentials or additional scope.
