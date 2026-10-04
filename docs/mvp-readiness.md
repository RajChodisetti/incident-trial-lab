# MVP readiness before thin MCP / agent integration

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

## Essential next work

1. **Evaluate the investigation loop before live-model claims.** Script an offline
   investigate/propose/test/revise sequence that rejects ineffective and overfit
   proposals, preserves unknown outcomes, and stops at attempt/time budgets. Human
   approval remains necessary for any future optional apply. A wrapper around three
   static variants is not yet a substantive AI agent.
2. **Resolve credentials only when model work starts.** Use the relevant credential
   skill and ask reuse/new decision then. No credential or provider is needed to
   define/test the deterministic tool surface now.

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

Next paced increment should define and evaluate the offline
investigation loop; defer live AI/API calls and broad controller work until review.
