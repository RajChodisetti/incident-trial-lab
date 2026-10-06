# Decisions beta HTTP selector — offline contract implementation

Contract checked against the [official Decisions guide](https://developers.openai.com/api/docs/guides/decisions)
and [October 6 changelog](https://developers.openai.com/api/docs/changelog) on
2026-10-06. Selected-project access has not been tested. No credential was read
and no provider request was made for this increment.

`internal/investigation/decisions.go` implements one text-only choice question
using `POST /v1/decisions`, model `gpt-6-luna`, and the documented
`model`, `input`, `questions` request fields. It parses the documented `answers`
envelope and choice answer fields: `type`, `name`, `choice`, `probabilities`,
`confidence`. It does not guess an SDK method, usage envelope, request token
limit parameter, or retention control. Unknown envelope metadata is ignored;
unknown answer fields cause a conservative failure, which may need review if
the beta contract changes.

The existing neutral wrapper supplies only untried `option-01` through `option-03`
and bounded, source-linked synthetic context. The adapter locally checks the
current compiled candidate source and business fixture before sending anything.
An explicit `abstain` choice stops proposal generation. Unknown values never
fall back to a candidate. Each answer must name the requested question and supply
a complete unique distribution plus finite probabilities/confidence in [0,1].
A sum tolerance of 1e-6 is our policy, not a provider calibration guarantee.
The selected choice need not be the largest probability, and confidence need
not equal its probability. Neither field decides recovery or trial eligibility.
Only the existing independent trial/held-out evaluators can establish outcomes.

A request is at most 24 KiB of serialized JSON (the projected selection context
remains capped at 16 KiB); responses are capped at 64 KiB and JSON depth 32.
Duplicate JSON keys, trailing JSON, multiple/missing answers, bad values and
unknown choices fail closed. HTTP uses a 15-second timeout, caller cancellation,
no retries and no redirects. Provider/transport error details are masked.
`Choose` returns the validated distribution, confidence and exact request SHA
for a future audit recorder; it does not itself persist a live receipt.

The constructor accepts a caller-supplied credential; it never loads environment
variables or files. There is no live CLI, spend ledger or automatic provider
wiring in this slice. Mock transports exercise the actual HTTP request contract,
validation, neutral-to-domain proposal mapping, abstention, cancellation and
status handling. These are adapter tests, not provider quality measurements.

## Proposed live test, still pending approval

1. Obtain explicit approval for up to 20 requests and an aggregate US$0.25 ceiling,
   synthetic incident/code only, reusing the previously chosen key/balance.
   Do not reset or reuse the consumed first Responses-test ledger.
2. Before dispatch, establish a defensible billed-input upper bound and implement
   an atomic, persistent reservation ledger that includes failed/timed-out calls.
   Recheck pricing. Byte bounds alone are not verified token/billing bounds.
   Stop before a reservation would exceed the approved ceiling; no retries.
3. One bounded proposal-only access/contract check. On denied access or contract
   mismatch, preserve the refusal and stop; do not change account permissions.
4. If successful and funded, run 18 fixed inputs: six alias permutations repeated
   three times, recording exact request/answer, probabilities, confidence, timing
   and independent held-out outcomes. Allow at most one additional explicit
   abstention/insufficient-evidence check. Total attempts, including the initial
   check and failures, must remain <=20. Define the insufficient-evidence fixture
   before running it; the current adapter intentionally requires valid observed
   context, so a fixture policy change would need its own reviewed increment.
5. Report raw correct/incorrect/abstain/invalid counts and permutation effects.
   No calibrated-confidence, speed, recovery or superiority claim follows from
   this tiny experiment. A fresh model-driven MCP trial loop is a later increment.

The guide documents input-only Decisions billing at US$0.10/million tokens,
with regional/long-context premiums and no output or cache charges. The
[model page](https://developers.openai.com/api/docs/models/gpt-6-luna) documents
1,050,000 context tokens, a 2x input multiplier above 272K, and a 10% regional
premium. Reserving the entire published model context, conservatively including
both premiums, gives **US$0.231 per call**, or **US$4.62 for 20 calls**. Under that
coarse bound, a US$0.25 budget permits only one attempt. The model context is not
an independently verified Decisions-specific billing ceiling.

For these small text requests, a provisional 32,768-input-token allowance would
estimate US$0.0032768/call; 20 would be US$0.065536 base, US$0.0720896 with the
regional premium, or US$0.1441792 with both premiums conservatively included.
These are conditional estimates, **not an enforceable worst-case spend bound**.
The guide does not establish how to count all billed Decisions question/input
framing locally. Until that is verified, do not promise all 20 calls fit the cap
or dispatch the proposed batch. Account availability, applicable retention and
actual usage/billing metadata remain untested.
