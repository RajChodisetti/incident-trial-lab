# MVP readiness before live agent integration

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
   The deterministic local demo continues to require no provider or credential.

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
