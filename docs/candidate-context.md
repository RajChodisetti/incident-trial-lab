# Bounded candidate code context

The neutral selector now supplies small technical code facts for its untried
options. `candidates.Changes()` embeds the actual `internal/candidates/service.go`
at build time and parses its quote-arithmetic branch. It specializes the dispatch
predicate for each registered candidate, removes semantic IDs and comments, and
projects the initial assignment and candidate arithmetic into bounded Go snippets.
No service behavior, image pin, fixture assertion or expected verdict is changed.

## Input and provenance boundary

Each selector-facing option has its neutral alias, the initial arithmetic, the
specialized arithmetic, the SHA-256 of the complete embedded source, and the
SHA-256 of the normalized specialized snippet. Each snippet is at most 1,024 bytes;
source input is capped at 16 KiB, and the catalog still has three candidates.
Only quote fields, integer literals and the supported arithmetic/boolean operators
survive projection. Calls, arbitrary identifiers, raw strings and comments are
rejected. Ambiguous/unsupported arithmetic shapes or extra total updates fail
closed rather than silently presenting stale hand-written descriptions.

`NewNeutralSelector` fixes these source facts alongside the random alias mapping
for one evaluation loop. It reconstructs `View.candidate_changes` from compiled
facts on each call, ignores caller-provided descriptions, and supplies only the
currently offered untried aliases. History and code-context slices are independent
copies; an inner selector cannot rewrite subsequent inputs through mutation.
Aliases, source hashes and code hashes remain stable through observed revisions.

No fixture expected values, future candidate verdicts, image filenames, semantic
preset names, private paths or raw logs/responses are supplied as candidate context.
Numeric conditions in the code remain visible because they are actual implementation
facts. Prior feedback remains observed recovery/regression/counts and aliased trial
history; it is not a privileged expected outcome for an untried candidate.

The source hash identifies the compiled projection's actual source bytes. It does
**not** attest that an existing pinned image was built from that source. Images
were not rebuilt or replayed in this offline slice. Existing behavior/provenance
references and fresh trial/receipt/comparison gates continue independently; a
selector's choice or code impression cannot grant a recovery verdict. The mapping
and context are local immutable inputs, not a durable request/selection journal.

The first Responses probe continues to accept only its original baseline projection;
it rejects added candidate context. Its request budget/ledger and deliberately
descriptive wiring prompt remain untouched. No provider transport is introduced.

## Verification and remaining work

Local tests compare projected arithmetic against the actual candidate HTTP handlers
on **192 synthetic inputs** spanning quantity/discount boundaries and both conditional
and general cases. This checks sampled projection equivalence, not universal service
correctness. Source/snippet hashes and bounds are checked; unsupported calls, labels,
extra total updates, changed output expressions and malformed source are rejected.

Fake-selector tests check informative context without expected-outcome/name leakage,
ignoring caller-injected descriptions, stable filtered aliases/code hashes through
no-improvement/regression/recovery revisions, mutation isolation and explicit
abstention. Existing independent-verdict and Responses fake-HTTP tests remain part
of the local regression suite. No semantic model reasoning is evaluated by these
fakes; no actual provider, fresh cluster or hosted CI is exercised.

Recorded 2026-10-06: focused candidate/investigation/Responses fake-HTTP race tests
passed; `./scripts/check.sh` passed formatting, all local race tests with coverage
and `go vet`; `git diff --check` passed. Candidate package coverage was 89.2% and
investigation coverage 78.9%, not guarantees of full correctness or model quality.
No actual credential was read and the consumed first-test reservation was unchanged.

Next prerequisites for meaningful selection evaluation:

- The subsequent [business/incident projection and offline record slice](business-context-records.md)
  now supplies the actual rule and observed failed assertion, with exact-input
  reconstruction. Keep held-out inputs/selection expectations separate from it.
- Define balanced/permuted fixtures, abstention/invalid/error metrics and a simple
  comparison baseline before claims about model quality. Check other leakage paths.
- The offline journal preserves exact context, owner alias mapping and decision.
  The subsequent [source/artifact gate](source-artifact-binding.md) is tested with
  fixtures; fresh bound images/replay and live-provider journaling remain before
  a live repair claim.
- Resolve Decisions API documentation, pricing/retention/limits and selected-project
  access, plus a newly approved request budget, before adding any provider adapter.

For the portfolio demo, prepare a readable evidence presentation, replay the owned
deterministic demo from the reviewed checkout, then capture actual screenshots and
captions. Label scripted, simulated and proposal-wiring results separately. Local
Prometheus presentation and the later authorized live-agent evaluation remain open.
