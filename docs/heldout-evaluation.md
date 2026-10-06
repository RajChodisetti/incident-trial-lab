# Offline held-out checkout evaluation

This is a bounded harness for the existing discount-rounding incident. It calls
injected offline selectors and exercises actual Go candidate HTTP handlers in
process. It makes no model request, starts no cluster and verifies no running
image, resource budget or trial receipt. Repository visibility remains private.

## Exact split

Selector input comes from the validated saved
[offline context record](evidence/selection-context-offline.json): the business
rule, observed `(105, 3, 10)` request with required 283 and observed 282 cents,
historical evidence references, baseline signal, no candidate history, one allowed
attempt and three source-linked arithmetic snippets. The two baseline controls
remain in the baseline fixture; they are not added as selector examples. The
existing six-case comparison workload is also excluded from selector input.

The evaluator-only [v1 dataset](../internal/investigation/testdata/heldout-discount-v1.json)
contains the following 18 requests. Columns are unit cents, quantity, discount
percent and expected response. Tests enforce unique names/requests, no request
overlap with either existing workload, legal-input results against an independent
arbitrary-precision contract calculation, and rejection status at all six limits.

| Case | Unit | Quantity | Discount | Expected |
| --- | ---: | ---: | ---: | --- |
| minimum-fraction | 1 | 2 | 1 | 200 / 1 cent |
| near-full-minimum | 1 | 100 | 99 | 200 / 1 |
| below-cent-boundary | 99 | 2 | 1 | 200 / 196 |
| above-cent-boundary | 101 | 2 | 1 | 200 / 199 |
| exact-bulk-cent | 101 | 99 | 99 | 200 / 99 |
| half-cent-bulk | 199 | 3 | 50 | 200 / 298 |
| full-discount-boundary | 103 | 99 | 100 | 200 / 0 |
| maximum-discounted-subtotal | 1000000 | 100 | 99 | 200 / 1000000 |
| near-maximum-rounding | 999999 | 100 | 1 | 200 / 98999901 |
| single-unit-rounding | 73 | 1 | 33 | 200 / 48 |
| minimum-no-discount | 1 | 100 | 0 | 200 / 100 |
| accumulated-small-fractions | 101 | 100 | 99 | 200 / 101 |
| unit-below-range | 0 | 1 | 0 | 400 |
| unit-above-range | 1000001 | 1 | 0 | 400 |
| quantity-below-range | 1 | 0 | 0 | 400 |
| quantity-above-range | 1 | 101 | 0 | 400 |
| discount-below-range | 1 | 1 | -1 | 400 |
| discount-above-range | 1 | 1 | 101 | 400 |

These cases and expected responses never enter `View` or `Selector.Select`.
The harness rebuilds neutral candidate facts through the existing wrapper, trying
all six assignments of three candidates to `option-01..03`. Every candidate occurs
twice at each position. Each call is independent, with no held-out feedback passed
to subsequent calls. Exact request hashes and owner orders are evaluator audit
output, outside selector input. Mutation-isolation tests protect later inputs.

"Held-out" means separated from this selector input. The fixtures are authored
with knowledge of this synthetic fault, committed and inspectable, and may overlap
earlier developer test inputs. They are not a secret benchmark, a prospective model
test, a training-data guarantee or cross-incident generalization evidence. No tuning
or threshold search was performed. A later evaluated provider must not receive
the evaluator dataset, outcome report or owner mapping as prompt context.

## Deterministic controls and actual results

From project root:

```sh
go run ./cmd/selection-eval
```

This command accepts no arguments and includes only offline controls. Its actual
2026-10-06 output is committed as
[held-out control evidence](evidence/heldout-controls-offline.json).
Dataset hash is SHA-256 of compact JSON:
`fad303ec893241c282bffc52481502a7ac47de49c06fa34cc58927a845d57f07`.
The seed record hash is included in each report; old runtime evidence remains
historical and cannot satisfy the new source/artifact gate.

| Control | Selected | All 18 pass | Abstained | Invalid | Selector errors |
| --- | ---: | ---: | ---: | ---: | ---: |
| source oracle: fixed | 6 | 6 | 0 | 0 | 0 |
| source oracle: ineffective | 6 | 0 | 0 | 0 | 0 |
| source oracle: misleading | 6 | 0 | 0 | 0 | 0 |
| always first option | 6 | 2 | 0 | 0 | 0 |
| explicit abstention | 0 | 0 | 6 | 0 | 0 |
| unknown option | 0 | 0 | 0 | 6 | 0 |
| injected selector error | 0 | 0 | 0 | 0 | 6 |

The source-oracle controls deliberately identify a predetermined variant by its
known logic hash; they do not infer a fix. Both faulty variants fail the same six
rounding cases and pass the other 12 checks per selection. First-option selection
passes all cases in two of six balanced assignments. This is a deterministic
position comparator, not model accuracy or a probabilistic quality estimate.

Abstention, invalid and selector-error outcomes run no case evaluation and are
never counted as successful selections. Tests separately cover empty/semantic/
unknown choices, contradictory candidate-plus-abstention, nil selectors, invalid
seed records, cancellation before/during selection and input mutation. Cancellation
interrupts the sweep and preserves partial accounting. Selectors must cooperate
with context cancellation; this is not an isolated process supervisor. At most six
calls and 108 handler checks occur per control. No raw selector error is retained.

## Verification and next slice

On 2026-10-06, focused investigation race tests and `./scripts/check.sh` passed
formatting, all package race tests with coverage and `go vet`; `git diff --check`
passed. Investigation coverage was 81.3%. Two actual CLI runs produced identical
output: 42 offline selector calls and 432 handler checks across the seven controls.
The CLI is exercised offline separately; its unit coverage remains zero. Coverage
measures tested branches, not semantic reasoning or runtime safety. No hosted CI,
credential access, installation, Docker build or live cluster was used.

Next remains a reviewed candidate rebuild/pin update and fresh owned bound replay,
then a readable evidence demo and actual screenshots/captions. Keep scripted,
offline control and live wiring results distinct. Provider documentation/access/
pricing and a new live-test budget remain independent unresolved gates.

The subsequent [fresh bound runtime replay](source-bound-runtime.md) now verifies
the current pins through the deterministic MCP workflow. It runs the existing
six-case comparison workload, not this evaluator-only dataset. Readable evidence
presentation and actual screenshot preparation are the next bounded work.
