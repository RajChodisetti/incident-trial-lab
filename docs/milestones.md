# Incremental milestone plan

Complete one milestone at a time, review evidence, then wait for continuation.
Use small coherent commits with real timestamps. Design tests/benchmarks before
claiming an improvement. Keep explanations short enough for Raj to walk through.

| Milestone | Scope | Acceptance evidence |
| --- | --- | --- |
| 1 — Foundation | Verify personal GitHub identity/tooling; private repo; Go module; conservative evidence evaluation; local checks; CI definition | Unit/race tests and vet pass; pushed SHA matches remote; report CI status explicitly |
| 2 — Reproducible baseline | Dedicated local kind setup; pinned synthetic service image/manifests; fixture/workload contract; business assertions; reproduce one baseline failure | Fresh setup reproduces expected failure; manifests reject unpinned images; tests define failure before implementation; cleanup demonstrated |
| 3 — Candidate trials | Fresh isolated trials for one candidate; measure recovery, regressions and resource budget; evidence bundles | Known bad/good/incomplete fixtures yield expected verdicts; repeatable workload; trial contamination checks; defined benchmark and measured results |
| 4 — Worker/controller lifecycle | Bounded concurrency; deadlines/cancellation; cleanup; restart handling | Failure-injection tests for cancellation/restarts; resource leaks and concurrency behavior measured |
| 5 — MCP and substantive agents | Evidence tools; investigate/propose/test/revise loop; approval boundary | Tool contract tests; agent evaluation suite; rejected proposals preserved; no optional apply without human approval |
| 6 — Portfolio presentation | Walkthrough, architecture, repeatable demo, LinkedIn screenshot preparation, limitations and related-work comparison; optional read-only Dynatrace evidence adapter | Raj can explain decisions; verified screenshots and concise captions; no commercial tenant required for core demo; explicit public-release decision |

Milestones 1–3 are implemented; verification and explicit coverage limits are
recorded in each evidence report. Milestone 3 checks sampled resource budgets;
peak sampling and statistically rigorous benchmarks remain future work. Next is
the next MVP-readiness slice, only after parent/user continuation. Lifecycle slices
provide cancellation, deadlines, explicit recorded-node reconciliation, and a
max-two local worker path. Creation gaps, host-wide admission, and durable/
distributed scheduling remain limitations. Fresh baseline/attempt provenance binding is verified in
`baseline-provenance-evidence.md`. The narrow local stdio MCP tool surface is verified in `local-mcp-evidence.md`.
The offline investigation-loop contract is evaluated in `offline-investigation-evidence.md`.
The first authorized live proposal-only wiring test is recorded in
`live-proposal-evidence.md`. Stop for review before connecting a live proposer to
fresh MCP trials or increasing the request budget. Docker
Desktop startup and official project-local kind installation were authorized for
milestone 2. Helm remains optional. Never use the current kubectl context as an implicit target: create a
dedicated kind cluster and explicitly select its context for every cluster command.

Raj explicitly authorized reuse of his user-created OpenAI key in ignored
`.env.local` for the one-request first test. No credential was created by the
project. Further API tests need a reviewed budget and scope.

## Agreed candidate-selector direction — 2026-10-05

Raj approved targeting **OpenAI Decisions API for candidate selection**, while
preserving the existing Responses probe and scripted loop until documentation and
access are confirmed. The [official September 29 announcement](https://openai.com/index/devday-2026-recap/)
describes Luna-based selection among finite predefined answers using text/image
context, and announced limited preview availability. That announcement is not a
request contract, pricing agreement or confirmation of Raj's account access.
Ordinary Luna model access must not be treated as Decisions API access.

This target fits the existing finite candidates (`fixed`, `ineffective`,
`misleading`); it does not imply novel patch generation or verified remediation.
As of this planning increment, no verified Decisions endpoint, request/response
schema, SDK method, pricing or project access is available to implement against.

### Existing replacement boundary

- `investigation.Proposer.Propose(context.Context, View)` is injected into the
  loop independently of the MCP `Caller`. Its view contains sanitized signals,
  references, prior outcomes and an attempt allowance; it receives no tool caller.
- The loop validates candidate IDs and latest behavior/provenance citations,
  rejects repeated variants, and retains existing deadline, trial provenance,
  evidence, comparison and cleanup gates. Selection cannot grant apply authority.
- `Scripted` already implements this boundary. `liveproposal.Probe` remains a
  standalone one-request Responses wiring test, not a `Proposer` implementation.
  It cannot be migrated by guessing an endpoint or swapping a model name.

### Migration prerequisites

- [ ] Obtain authoritative Decisions API documentation: endpoint/auth scope,
  request and finite-answer encoding, response contract, supported context/limits,
  error/refusal/timeout behavior, and any documented SDK support/version.
- [ ] Confirm explicit access for Raj's selected personal project through an
  authorized access/setup path. Do not infer it from another model or probe.
- [ ] Verify pricing, billing units, request/rate limits and retention behavior.
  Approve a new request/spend cap and credential-use scope before a live test;
  the consumed first Responses test does not authorize another request.
- [ ] Define offline fixtures and acceptance criteria before implementation or
  improvement claims, including failure and uncertainty handling. Preserve the
  existing integration; choose adapters explicitly with no automatic paid fallback.
- [ ] Only after these gates, implement against the verified contract and review
  one bounded proposal-only test before considering fresh MCP trial integration.

### Minimal offline implementation increment — completed

The [offline selector adapter](selector-adapter.md) now implements `Proposer`,
tested with injected fake selectors returning a candidate ID or explicit abstention.
Shared finite candidate validation is used across adapter, proposal and schema
boundaries. Already-attempted IDs are filtered before selection; exhausted choices
stop. Accepted IDs map to the trusted view's latest two references and deterministic
audit text. Citations establish structural linkage, not selector reasoning.

Offline tests cover allowed IDs, unknown/empty/repeated answers, exhausted choices,
malformed/mismatched references, cancellation, selector errors and abstention.
Fake tools confirm rejection starts no candidate trial and selection cannot
override independent recovery, regression, resource, uncertainty, provenance or
final-comparison failures. A compile-time assertion checks `Proposer` compatibility.
No provider SDK or live cluster is added; no worker/controller redesign is needed.

The prior planning-only increment made no implementation changes. This subsequent
authorized slice implements only the offline adapter; no Decisions API transport,
paid request, credential inspection/change, installation or live trial occurred.
Next is resolution of the migration prerequisites before any provider wiring.
Keep the private repository and the portfolio screenshot checklist intact.

The subsequent [neutral candidate ID slice](neutral-selector.md) adds randomized,
loop-stable anonymous options and aliased history for fake selector evaluation.
It addresses label/order leakage without changing the existing Responses prompt
or claiming model quality. See the [completion assessment](mvp-readiness.md) for
built/tested, unverified and blocked work, remaining increments and effort ranges.
Offline evaluation context and reproducible evidence presentation can progress
while the provider migration gates remain unresolved.

The next [bounded candidate-context increment](candidate-context.md) now adds
actual quote-arithmetic snippets with source/snippet hashes under the same neutral
aliases. Projection and fake-selection tests do not establish model reasoning or
source-to-image attestation. Business/incident context, held-out evaluation and
reproducible journaling remain before meaningful selection-quality claims.

The subsequent [business/incident and offline-record slice](business-context-records.md)
now projects the actual contract and observed failed baseline assertion, and records
an exact neutral input plus owner-side mapping for validated reconstruction.
Expected outcomes for untried candidates remain outside selector input. Held-out
evaluation and source-to-image binding still remain; no live provider is wired.

## Portfolio screenshot completion checklist

Raj requested: “Get screenshots to put in as project in LinkedIn once the product
is ready.” Prepare these assets when the working demo reaches a presentable,
verified milestone; this requirement does not authorize LinkedIn publication.

- [ ] Confirm the chosen demo completes with reproducible baseline failure,
  evidence-linked investigation, candidate comparisons, and verified
  recovery/regression results. Record the tested commit and evidence bundle.
- [ ] Capture actual screenshots of investigation, candidate fix comparisons,
  and recovery/regression evidence from that demonstrated demo. Preserve failed
  or inconclusive outcomes where relevant; do not stage unsupported behavior.
- [ ] Use readable framing, labels, text sizes and contrast. Exclude credentials,
  private local paths, account details and unrelated windows or information.
- [ ] Clearly label scripted preset orchestration separately from live-model
  results. The first live proposal-only wiring check is not a live agent trial
  or proof of model-driven recovery; claim those only after separately verified
  implementation and evaluation.
- [ ] Prepare concise recruiter-friendly captions explaining the engineering
  problem, Go/Kubernetes/MCP contribution, observed outcome and material limits.
  Keep the screenshot set ready for Raj's review; do not publish to LinkedIn.

Dependencies: a reviewed demo presentation and traceable verified evidence for
each screenshot. A deterministic demo can be labeled and presented independently;
any screenshots claiming live-model investigation/testing require a later
authorized, evaluated live-agent increment. Screenshot preparation itself does
not justify extra API requests, restarted trials or expanded implementation.
