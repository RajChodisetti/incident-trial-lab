# Observed incident context and offline selection records

This slice adds a bounded business/incident projection and a reconstructable
offline selection journal. It does not add a provider adapter, live model test,
candidate trial, install, credential access or public publication. The current
scripted loop and consumed Responses probe remain intact.

## What the selector can see

The scenario package embeds the actual baseline `contract.md` and `workload.json`.
It verifies the business sentence, loads the validated baseline workload, and
requires all recorded controls to pass and exactly one recorded assertion to fail
with a verified numeric HTTP response. Only the business rule, contract/workload
hashes, the observed incident request, required assertion value, observed value
and HTTP status enter `View.business_incident`. Names, prose outside the rule,
raw responses, known-fault markers and comparison workloads are excluded.

The visible assertion is **required 283 cents, observed 282** for an order of
three 105-cent units discounted by 10%. This is an already observed baseline
assertion, not the expected verdict or answer for an untried candidate. The general
business rule is task context a human investigator would need. Future candidate
recovery/regression answers and held-out evaluation cases remain outside input.
Existing neutral aliases and source/snippet hashes accompany untried code facts;
prior feedback represents observed outcomes only.

`HistoricalIncidentView` rehashes both captured baseline evidence sections,
checks trial/scenario linkage and verifies the baseline workload hash against
the captured experiment. Saved eligibility does not authorize a current trial.
The projection is optional in the local library and exercised by the offline
record command; the default scripted trial CLI is not rewired in this slice.
The Responses first-test validator rejects this added context, preserving its
original request scope and reservation.

## Exact request and owner audit

`NewOfflineSelectionJournal(fake)` wraps a fake selector with the neutral-ID layer.
Each record stores the exact pre-selection request (view and choices), its SHA-256,
the normalized decision/status, actual creation time and a full-record SHA-256.
**Owner alias mappings stay outside the selector-facing request.** Never send the
whole owner record as a provider prompt. Input snapshots and accessor results are
deep copies, so selector/reviewer mutations cannot rewrite the recorded request.

The in-memory journal permits at most three records. It retains selected,
abstained, invalid, error and canceled categories; raw invalid answers and error
bodies are omitted. It does not authorize provider spending, retry decisions or
create a durable worker. A pending/unfinished record cannot pass replay validation.

`DecodeSelectionRecord` rejects unknown fields, oversize records, malformed
aliases/evidence/history, status mismatches, changed hashes, unverified scenario
context and code facts that differ from the currently compiled projection.
Reconstruction performs **no selector call, trial or apply action**. Records are
local Go JSON snapshots with version 1, not a public API contract or signed
attestation. Source/contract drift requires an explicit compatible snapshot/version
strategy; silently accepting an old record under new code is not supported.

## Actual offline evidence

From project root, with the saved synthetic capture available:

```sh
go run ./cmd/selection-record -record
go run ./cmd/selection-record -replay
```

The record command uses the fixed owned historical capture path and refuses to
overwrite `evidence/selection-context-offline.json`. It has no model transport or
trial caller. This slice created one record at **2026-10-06T02:55:16.364510Z** using
an always-abstaining stub and validated it by replay. Source evidence is the earlier
actual scripted baseline, trial `d2ddae27ffc3792c`; no fresh incident ran here.

Request SHA-256:
`16c827b16afa0e59f5af2689c9bf773404d6e47609c3811413955b53c2c90f14`.
Record SHA-256:
`088224f3bf058ad4fe049a626022c60afd28761056aa335359a7b09820a3f9cf`.
The committed [offline record](evidence/selection-context-offline.json) contains
sanitized synthetic context and owner audit mapping, with mode
`offline_fake_selection` and status `abstained`. This proves record plumbing and
reconstruction, not model reasoning, abstention calibration or recovery.

Tests cover verified observed assertions, rejection of malformed/injected responses,
missing controls and wrong capture/fixture linkage; neutral input without expected
candidate outcomes; mutation isolation; exact JSON reconstruction; tampering,
unknown fields, invalid answers and selector errors; canceled inputs and journal
bounds. Local CLI recording/replay is exercised separately from unit coverage.

Local verification on 2026-10-06: `./scripts/check.sh` passed formatting, full
package race tests with coverage and `go vet`; `git diff --check` passed. Focused
investigation race tests/vet also verify the committed artifact and a canceled
record after adding those cases. Scenario-package coverage was 90.0%; CLI unit
coverage remains zero, with its actual offline record/replay paths exercised above.
Coverage and reconstruction do not prove model quality or runtime image provenance.

## Remaining prerequisites

- Held-out scenarios/cases, balanced option permutations, evaluation metrics and
  a simple comparator remain before selection-quality claims. The published seeded
  case and candidate snippets are not a held-out benchmark.
- The subsequent [source/artifact binding gate](source-artifact-binding.md) is
  fixture-tested offline. Existing images/receipts lack its proof and remain
  historical; a reviewed rebuild, pin update and fresh runtime replay are required.
- This journal is in-memory plus an explicitly saved offline artifact, not durable
  restart orchestration or a live-provider response attestation scheme.
- Verified Decisions API contracts/access/pricing and a new approved budget still
  gate provider integration. No endpoint, schema or access is assumed here.
- Readable evidence presentation, reviewed fresh deterministic demo replay,
  actual screenshot/caption preparation and local Prometheus remain open. Keep
  scripted, fake-selector and live proposal-wiring results visibly distinct.
