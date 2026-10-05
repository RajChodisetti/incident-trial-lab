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

1. **Resolve credentials only when model work starts.** Use the relevant credential
   workflow and ask provider/model, reuse/new and request/spend-cap decisions then.
   Start with one read-only proposal validation over captured synthetic evidence;
   the deterministic local demo continues to require no provider or credential.

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

Next decision: provider/model, authorized credential reuse versus separately
approved creation, and a request/spend cap. The smallest proposed validation is
a single read-only proposal dry run over captured synthetic evidence. Defer live
requests and broader agent/controller work until that decision and review.
