# Portfolio MVP completion assessment

Assessment updated 2026-10-06. The deterministic trial foundation is implemented;
the full live-agent portfolio MVP is not complete. A percent would conceal the
external provider-access dependency and the difference between an engine demo and
verified AI capability.

| Area | Built/tested evidence | Unverified or blocked |
| --- | --- | --- |
| Synthetic Go/Kubernetes incident and comparisons | Historical actual kind runs reproduce baseline failure and three candidate outcomes; pinned inputs, assertions, sampled budgets, cleanup | No fresh replay of the current checkout in this offline increment; resource sampling excludes peak memory/throttling and rigorous performance claims |
| Local lifecycle and workers | Deadline/cancellation, scoped cleanup/reconciliation, bounded local queue and provenance checks; local race tests and earlier real-run evidence | Durable/distributed controller and broad outage hardening remain deferred |
| MCP and investigation loop | Typed stdio tools; historical actual scripted loop; fake-tool rejection, uncertainty and revision tests | Actual model-driven investigate/test/revise loop is not demonstrated |
| Selector boundary | Fake-tested finite selector and independent verdicts; neutral source-linked code, bounded observed business/incident context, reconstructable offline records and fixture-tested source/artifact gate | Held-out evaluation and a reviewed image rebuild/pin update/fresh bound replay remain; fake choices prove no reasoning |
| Live provider | One successful Responses proposal/schema wiring request, with measured usage | Decisions contract, pricing/limits/retention and selected-project access are unverified; a new live-test budget is required |
| Portfolio presentation | Walkthroughs and preserved evidence summaries | Reproducible readable evidence presentation, screenshot assets/captions, interview rehearsal and related-work comparison remain |
| Telemetry and release | Sampled container resource evidence | Local Prometheus demo is not implemented; hosted CI allowance and public-release decision remain unresolved |

## Remaining small increments and effort assumptions

1. **Evaluation fixtures:** [candidate code facts](candidate-context.md),
   [observed business/incident context and offline records](business-context-records.md)
   now provide a reconstructable input. Add held-out fixtures and
   invalid/abstaining/permutation metrics. Neutral IDs remove label/order hints,
   not other answer leakage. The [source/artifact gate](source-artifact-binding.md)
   is implemented offline; real bound image/runtime validation remains.
2. **Deterministic presentation:** render verified historical evidence clearly,
   distinguish scripted/fake/live wiring results, then rehearse an owned local
   demo and retain fresh evidence from the tested checkout.
3. **Portfolio assets:** capture actual readable screenshots, write concise
   captions and walkthrough/related-work notes, and review with Raj. Prepare only;
   LinkedIn publication and repository visibility remain separate decisions.
4. **Live selector, after gates:** implement against verified Decisions docs,
   review one bounded proposal-only evaluation, then a fresh owned MCP loop with
   independently checked recovery/regressions. Do not reset the first-test ledger.
5. **Telemetry:** provide the local Prometheus view using standard telemetry and
   synthetic data, with a verified walkthrough and no commercial tenant.

Planning estimate: **2–4 focused development days** for the deterministic
evaluation/presentation/screenshot work, then roughly **3–6 additional focused
days** for documented live-selector evaluation and the local telemetry demo.
Assumptions: one current synthetic scenario, cached working Mac/Docker/kind setup,
small reviewed increments, timely owner feedback and usable provider docs/access.
These are effort ranges, not completion dates; provider-access waiting time is
unknown and excluded. Scope expansion or provider-contract surprises change them.
Production hardening and optional Dynatrace integration are outside these ranges.

The local deterministic trial foundation now has reproducible baseline evidence,
three candidate outcomes, explicit uncertainty, bounded supervision, conservative
recorded crash cleanup, and a maximum-two candidate worker path. Do not require
solving every outage case before a narrow local demo, and do not claim a complete
production controller.

The [fresh provenance gate](baseline-provenance.md) now binds each candidate to a
verified local baseline, exact scenario/image identities and a reserved attempt.
It rejects stale/mismatched/reused evidence and revalidates stored live receipts.
This is local consistency checking, not signed or adversarial attestation. Legacy
standalone results remain unbound. See [verification evidence](baseline-provenance-evidence.md).

The [local stdio MCP surface](local-mcp.md) now provides typed, process-owned
scenario submission, status/evidence retrieval, comparison and cancellation.
[Offline and real protocol evidence](local-mcp-evidence.md) distinguish simulations
from actual kind runs. Preset selection is still not substantive AI investigation.

The [offline loop contract](offline-investigation.md) now evaluates bounded
investigate/propose/test/revise control flow with structural evidence references.
[Its evidence](offline-investigation-evidence.md) includes one actual MCP experiment
and explicitly simulated failure/cancellation tests. A predetermined preset
sequence remains control-flow evaluation, not AI diagnosis or patch generation.

## Essential next work

1. **Resolve the provider migration prerequisites.** Raj approved
   Decisions API as the target, pending verified documentation, pricing and
   project access. Follow [the migration gates and offline adapter plan](milestones.md#agreed-candidate-selector-direction--2026-10-05).
   The [offline selector adapter](selector-adapter.md) now connects injected fake
   selections to `Proposer`, including explicit abstention. Keep the scripted loop
   and existing Responses probe intact before any verified provider wiring.
   The [neutral ID wrapper](neutral-selector.md) removes semantic labels from
   selector-facing options/history, with source-linked [candidate code context](candidate-context.md).
   Observed [business/incident context and offline records](business-context-records.md)
   are now implemented; held-out evaluation and evidence presentation can progress
   offline while provider prerequisites remain unresolved.
   The deterministic local demo continues to require no provider or credential.
   Existing candidate images lack the new source/artifact proof: rebuild, review
   pins and verify a fresh owned replay before presenting a current bound demo.

## Deferred hardening for the local demo

- Creation-to-node-snapshot crash gap: retain refusal/manual review; do not weaken
  ownership to chase cleanup completeness.
- Host-wide admission across multiple CLI batches, durable pending queues, batch
  crash recovery, distributed leases, and multi-host scheduling.
- Daemon outage/host sleep/power loss, corruption, adversarial ownership/PID tampering,
  long soaks, broad real-service scenarios, and multi-platform portability.
- Peak-memory/throttling metrics and statistically rigorous performance benchmarks.
- Optional read-only Dynatrace evidence, production adapters, public release, and
  hosted CI enablement once no-cost allowance is resolved.

The [first Responses proposal-only test](live-proposal-evidence.md) completed after
explicit credential reuse and spend-cap approval. It verified wiring over
historical synthetic evidence, not model-driven recovery. Its attempt is consumed.
The offline selector adapter is implemented and locally evaluated with fakes.
A live Decisions test depends on verified access/contract/pricing and a newly
approved request budget. No endpoint, schema, SDK method or availability is assumed.
