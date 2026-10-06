# Neutral candidate IDs for offline selector evaluation

`NewNeutralSelector(inner)` wraps the existing provider-neutral `Selector` and can
be injected into `SelectorProposer`. Create one wrapper per evaluation loop. The
inner selector sees `option-01`, `option-02`, `option-03` instead of answer-revealing
`fixed`, `ineffective`, `misleading` names. An unbiased random permutation assigns
aliases; semantic catalog ordering is not preserved. Aliases stay stable across
revisions in that loop, including prior feedback and the untried choice list.

Only the owner-side wrapper maps an accepted alias back to the real candidate ID.
Final loop traces retain real IDs for review and independent trial evidence.
Unknown or unavailable aliases, semantic IDs, empty decisions and ambiguous
abstention are rejected. Explicit abstention passes through to the existing
inconclusive stop. Input history is copied; selector mutations cannot rewrite
earlier evidence. Cancellation and selector errors retain the no-fallback behavior.
No CLI default, image pin, trial outcome or existing Responses prompt is changed.

## Offline evaluation contract

Acceptance criteria: no semantic IDs in the inner view/choices; a bijective
mapping across all six candidate permutations;
stable filtered aliases through revisions; invalid decisions starting no candidate
trial; abstention preserving baseline evidence and an inconclusive trace; and
unchanged independent no-improvement/regression/recovery outcomes.

Tests enumerate all six permutations and all three choices, verify input/history
isolation, exercise unknown/semantic/empty/unavailable/ambiguous/abstaining answers,
and simulate the three-step revision sequence. The random constructor is checked
for a complete bijection without relying on a particular random assignment.
Cancellation and generic error handling are also tested. These are fake-selector
and fake-tool control-flow checks, not a live model or cluster evaluation.

Verified locally on 2026-10-06: focused investigation race tests passed;
`./scripts/check.sh` passed formatting, full package race tests with coverage and
`go vet`; `git diff --check` passed. Investigation package coverage was 78.6%,
which is not a model-quality or full-runtime correctness claim. No provider,
fresh cluster replay or hosted CI was run, and the consumed request ledger was
not reset. Earlier actual evidence remains explicitly historical.

## Limits

This addresses **label and ordering leakage only**. It does not remove answer hints
from candidate descriptions, code, filenames, prompts or unrelated context. The
prior Responses wiring probe deliberately describes the presets; it remains a
wiring result and is not upgraded into a reasoning benchmark. No paid retest runs.

The subsequent [candidate-context slice](candidate-context.md) supplies bounded,
source-linked arithmetic snippets under these aliases. A meaningful evaluation
still needs reviewed business/incident context and held-out expectations, balanced
permutations and selection/abstention/error metrics before model-quality claims.
Anonymous IDs and fake selections do not demonstrate diagnosis or intelligent choice.
No Decisions API endpoint, schema, SDK or access is assumed by this wrapper.

The wrapper is an opt-in library boundary with offline tests; no live selector is
wired into the existing CLI. Its small local maps are not a durable multi-process
evaluation journal. Each instance should be confined to one loop, and access/budget
review is still required before any future provider call.
