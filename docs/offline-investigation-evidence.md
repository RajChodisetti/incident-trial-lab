# Offline loop verification evidence

Verified 2026-10-05 UTC in the working tree based on `8394a8e`. Existing service image
pins, assertions, MCP tool allowlists, provenance and supervisor/worker code were
unchanged. No live model, API credential, persistent grant, paid service or hosted
CI was used.

`./scripts/check.sh` passed formatting, all package race tests and `go vet`.
Investigation-package unit coverage is 72.4%; actual MCP adapter/subprocess behavior
is exercised by the real run, not fully covered by unit instrumentation. CLI unit
coverage is zero. `git diff --check` passed.

**Simulated tool/proposer tests** verify the failed/no-improvement → regression →
verified-recovery sequence, attempt exhaustion, duplicate proposals, unsupported
command-like variants, missing/unknown/stale evidence references, proposer/tool
errors, response loss after admission, stale provenance and inconclusive evidence.
They also verify cancellation/deadline cleanup, explicit unverified-cleanup reporting,
nonempty-session refusal and a noncooperative proposer deadline. Injected instructions
in observations are preserved in captured data and withheld from the proposer view;
they do not add commands, paths, tools or grants. These fakes do not verify real
image/runtime provenance or AI reasoning.

**Actual execution** was `.build/offline-loop -real` through a dedicated stdio MCP
server. Loop `eaa167c837aeb1c5` ran from `2026-10-05T00:03:35.604059Z` to
`2026-10-05T00:06:17.521026Z`, within its eight-minute decision budget. Fresh
experiment: `18032624273d2878`; baseline run: `b74685da0787e41f`.

| Scripted proposal | Actual run | Recovery/regressions | Recorded outcome |
| --- | --- | --- | --- |
| ineffective | `51136a6039f9d98a` | failed / failed; 40 measured failures | no improvement |
| misleading | `fe2f5f7fab3bdaa1` | passed / failed; 20 measured failures | regression; revise |
| fixed | `106c42c61da46b66` | passed / passed; zero failures | verified recovery; stop |

Each candidate made 120 measured requests after warmup in a fresh isolated cluster.
All three passed the predefined sampled resource budget; observations are retained
in the compact evidence, with no improvement/peak-memory claim. The loop captured
and cited eight behavior/provenance responses. All eight SHA-256 hashes were
independently recomputed from the saved Go-serialized captures and matched. Final
`compare_trials` confirmed the fixed receipt remained current. The trace records
651 status polls, eight evidence reads, three candidate submissions and one final
comparison; no unsuccessful attempt was discarded.

All four lifecycle journals reported completed outcomes and verified cleanup.
Independent Docker queries of their exact recorded owned labels found zero
remaining containers. No unrelated containers were inspected or removed. Raw data
is under `evidence/offline-loop-eaa167c837aeb1c5`,
`evidence/mcp-session-265deb7252c7050d`, the four lifecycle directories, and the
experiment directory. Compact trace, hashes, sampled budgets and cleanup results
are committed in [offline-loop-real-summary.json](evidence/offline-loop-real-summary.json).
The final regression suite also verified audit preservation after a lost submission
response; the successful trial path was unchanged. Remote SHA/CI status are
verified separately after pushing.

The actual proposer was intentionally scripted. No actual cancellation/tool-failure
matrix, live model evaluation, autonomous diagnosis or patch generation was run in
this slice. Those paths are simulated control-flow checks plus the existing engine
regression suite. Review this contract before any live-model request; provider,
credential reuse/new decision and an explicit request/spend cap remain required.
See [the smallest dry-run plan](offline-investigation.md#decision-before-live-ai).
