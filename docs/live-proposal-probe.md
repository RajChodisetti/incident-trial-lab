# First live proposal probe

This increment adds one OpenAI Responses request over a historical synthetic
baseline projection. The owner authorized reuse of the user-created key in ignored
`.env.local` and a US$2 total first-test cap. No tool, trial, patch or apply action
is exposed to the model. The offline loop remains scripted.

The CLI uses only `evidence/offline-loop-eaa167c837aeb1c5/captures.json`, verifies
the two baseline payload hashes, then projects typed counters and reference
metadata. It sends no raw observations, errors, logs, arbitrary paths or employer
data. Historical eligibility fields do not grant current trial authority. Fixed
preset descriptions explain the general formula in the prompt; choosing `fixed`
therefore checks wiring rather than independent diagnosis or repair quality.

## Request and budget

The pinned snapshot is `gpt-5.4-mini-2026-03-17`, with standard service tier,
`store:false`, reasoning effort `none`, no tools, no redirects, no proxy override,
45-second HTTP timeout, 1,536 output-token cap and 16 KiB request ceiling.
The schema allows only `fixed`, `ineffective` or `misleading`, a bounded rationale,
and exactly two reference IDs. Local decoding additionally rejects unknown fields,
trailing JSON, duplicate/unknown references and missing latest behavior/provenance.
Structural validity does not establish semantic correctness.

Official pricing checked **2026-10-05**: input US$0.75, cached input US$0.075,
output US$4.50 per million tokens. Published context is 400,000 tokens. Sources:
[model limits and snapshot](https://developers.openai.com/api/docs/models/gpt-5.4-mini),
[standard pricing](https://developers.openai.com/api/docs/pricing), and
[Structured Outputs](https://developers.openai.com/api/docs/guides/structured-outputs).

Before sending, an exclusive, synced local marker permanently consumes the one
authorized attempt. The reservation charges the entire published context at the
uncached rate plus the output cap, doubled: **US$0.613824**, below US$2. Failures,
refusals, timeouts and process interruption consume the attempt; there are no
automatic retries or refunds of unknown usage. Concurrent invocations cannot
reserve a second attempt. The ledger is local, not a provider billing limit;
deleting it does not authorize a repeat. This cap covers this controlled request's
standard token charges, not unrelated account activity or taxes. Recheck pricing
and authorization before any later request.

On a completed response, estimate from reported uncached/cached input and output
tokens. Reasoning tokens are included in output and not double-counted. Missing or
invalid usage stays unknown, covered by the reservation. Raw API errors and
credential values are never included in reports; quota/auth failures stop for owner
review, with no billing changes or credit purchase.

## Running and inspecting

Only run after explicit authorization; the first run consumes the attempt:

```sh
go run ./cmd/proposal-probe -authorized-first-test
```

The CLI requires `.env.local` to remain ignored and untracked. It reads the key in
memory for the normal HTTPS Authorization header; it never exports or copies it.
Local `evidence/live-first-test/attempt-reserved.json` and `result.json` record the
attempt, status, usage, cost estimate and validated proposal. No raw response is
persisted. Tests use a distinct fake credential and fake HTTP transport.

The next decision is review of this first result. Connecting a live proposer to
fresh MCP trials, more requests, model comparisons and diagnosis benchmarks all
remain separate increments requiring an updated bounded test plan.
