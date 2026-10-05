# Offline finite-selector adapter

`investigation.SelectorProposer` now connects a provider-neutral typed selection
to the existing `Proposer` loop. This increment supplies **no provider adapter**;
all new selections and trial responses are fakes in local tests. The existing
scripted CLI and standalone Responses probe remain available. No paid request,
credential read/change, model install or live cluster ran for this slice.

## Contract and walkthrough

1. The owner injects a `Selector` implementing
   `Select(context.Context, View, []string) (Decision, error)`. `Decision` is a
   local domain type with a candidate ID or explicit abstention; it is not an
   assumed Decisions API response schema.
2. Before selecting, require structurally valid latest behavior/provenance
   references sharing a loop and trial, a bounded attempt allowance and valid
   prior candidate history. Offer only untried IDs from the shared finite catalog.
   Exhausted choices stop without invoking the selector.
3. Pass independent choice/history slices to the selector. It has no MCP `Caller`,
   shell, path, apply or verdict capability. Cancellation is checked before and
   after selection; the existing loop also bounds a noncooperative proposer.
4. Reject empty/unknown/repeated candidates and ambiguous candidate-plus-abstention
   results. Selector errors stop with a generic error and no fallback. Explicit
   abstention records `selector_abstained`, leaves the trace verdict inconclusive,
   starts no further candidate trial, and retains already captured evidence.
5. Map an allowed ID to a proposal with the trusted view's latest two references
   and **deterministic audit text**, not a claimed model explanation. Shared
   proposal validation rejects malformed latest references, duplicate/unknown
   citations and missing current behavior/provenance citations.
6. The existing loop remains responsible for a fresh owned trial, independent
   evidence, recovery/regression/warmup/resource checks, provenance, final current
   comparison, deadlines and cleanup. Choosing `fixed` cannot grant a passed
   verdict; abstention does not erase previous failures or create recovery.

The shared `candidates.IDs`/`ValidID` policy is used by the selector, proposal
validator, Responses output enum, candidate handler guard, and MCP catalog,
schema and admission guard. Returned ID slices cannot mutate that policy.
The allowed strings and current Responses request shape are unchanged. No live
Responses retest was performed or authorized in this increment.

## Local verification

Tests exercise all three allowed IDs, history filtering, input mutation isolation,
unknown/empty/repeated answers, explicit and ambiguous abstention, exhausted
choices, malformed/mismatched references, cancellation and selector failure.
Simulated loop tests verify rejected selections start no candidate trial; an
abstention after a failed trial preserves its evidence; and the revision sequence
retains no-improvement, regression and verified-recovery outcomes. Independent
recovery, regression, resource, uncertainty, stale provenance and final-comparison
failures prevent a passed trace even when the selector chooses `fixed`.

`./scripts/check.sh` is the acceptance check: formatting, full local race tests
with coverage, and `go vet`. HTTP/tool fakes and local test servers do not make
provider requests or create clusters. No hosted CI is dispatched.

This slice passed that check on 2026-10-05. The investigation package also passed
focused race tests and vet after the added duplicate-citation rejection case.
Existing Responses fake-HTTP tests passed without reading the approved key or
repeating its consumed live request. CLI unit coverage remains zero; no actual
trial CLI or provider endpoint was exercised in this slice.

## Limits and next dependency

Reference shape/hash checks establish structural linkage, not current eligibility,
payload authenticity or rationale correctness. Existing trial provenance gates
remain necessary. The finite presets do not generate patches; fake sequences
measure control flow, not AI diagnosis or model quality. No production-controller,
optional-apply or screenshot publication capability is added.

Decisions API remains the intended selector, pending the [documented migration
gates](milestones.md#agreed-candidate-selector-direction--2026-10-05): verified
contract/SDK support, account/project access, pricing/limits/retention, and a newly
approved live-test scope and budget. Neither ordinary Luna access nor the consumed
Responses test grants Decisions access. A future provider adapter must explicitly
map the verified response to this local `Decision` type, define uncertainty/error
handling, and record the correct execution mode before any live loop evaluation.
