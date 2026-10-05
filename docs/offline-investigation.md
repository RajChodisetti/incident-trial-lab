# Offline investigation-loop evaluation

This increment evaluates **control flow with a scripted proposer**, not AI
capability. It connects the existing typed MCP operations into an
investigate/propose/test/revise sequence. No provider, model call, credential,
custom patch, arbitrary command or apply operation is added.

## Contract and walkthrough

`investigation.Run` receives a dedicated initially empty tool session, a proposer
with only a typed `View`, and owner-selected budgets. A proposal contains one
allowlisted candidate ID, a bounded rationale and evidence reference IDs. The
proposer has no tool caller, shell, filesystem or permission capability.

1. Inspect the supported catalog and run a fresh supervised baseline.
2. Require complete cleanup/current provenance and the reproduced fault. Capture
   behavior and provenance tool responses with immutable SHA-256 references.
3. Give the proposer the latest references, typed recovery/regression signals,
   failure/unknown counts and sanitized prior feedback. Raw responses, observation
   names/errors, tool errors and earlier rationale text are not passed as instructions.
4. Require a supported variant and citations of **both latest behavior and latest
   provenance**. Unknown or stale references, oversized rationale, arbitrary
   commands and repeated variants stop before another trial is started.
5. Test the candidate through `start_trial`, then read fresh typed evidence. Require
   matching experiment, candidate image specification, run identity and receipt.
   Known no-improvement/regression/budget failures feed a revision. Unknown or
   unverified evidence stops conservatively. Recheck current comparison before
   accepting the existing workload's recovery verdict.

Reference IDs belong to this loop and name trial/section/hash; they are not paths,
credentials or permissions. Citations establish structural evidence linkage, not
that the rationale logically entails its references. Full captured responses remain
untrusted data and are retained separately for review. The scripted sequence
`ineffective → misleading → fixed` is predetermined; it does not discover the bug
or invent the general formula.

The decision budget permits one to three candidate attempts, at most ten minutes,
and at most 4,096 tool calls. The executable uses three attempts/eight minutes and
250 ms polling. Each normal tool request has a five-second deadline. A proposer
cannot extend the decision deadline by ignoring its context; one abandoned proposer
goroutine can remain confined to the process until it exits. No automatic retries
hide tool failures or observations.

Cancellation/deadline exhaustion uses a separate 90-second cleanup context to cancel
and poll the owned active job. A lost submission response is reconciled through the
dedicated session's bounded owned-job listing, not guessed PIDs or paths. Nonempty
sessions are refused so unrelated jobs are not adopted. Cleanup failures remain
explicit/inconclusive; no recovery outcome is fabricated. The client closes the
stdio process using its existing graceful shutdown allowance. Durable loop restart,
shared-session orchestration, adversarial attestation and host-wide admission remain
outside this contract.

## Running it

Simulated control-flow tests create no cluster or model resources:

```sh
go test -race -count=1 ./internal/investigation
```

After building the existing pinned archives and MCP/supervisor binaries, explicitly
run one actual synthetic experiment:

```sh
go build -o .build/offline-loop ./cmd/offline-loop
.build/offline-loop -real
```

The command requires `-real`; it launches its own local stdio MCP server with one
worker. It prints a compact trace and saves complete captured evidence under
`evidence/offline-loop-<loop-id>/captures.json`. This is actual Kubernetes execution
with an offline scripted proposer, not simulated Kubernetes or live AI.

A verified recovery means only the existing fixture assertions and sampled budgets
passed in fresh trials. There is no speedup, peak-memory, production-safety, novel
AI capability or universal cloning claim. Human review remains necessary before any
future optional apply, and this interface has no apply tool.

## Decision before live AI

Raj must choose the **provider/model** and whether to reuse an already authorized
project credential or separately approve creating a new one. Resolve the provider's
appropriate credential workflow then; do not inspect secret values, persist them in
the repo, create OAuth grants or broaden permissions. Also explicitly authorize a
small request/spend cap; this increment authorizes no paid model calls.

The smallest next validation is one bounded, read-only **proposal dry run** using a
sanitized captured synthetic view: no fresh Kubernetes trials, patch build or apply.
Request only the strict proposal schema, validate the candidate allowlist and known
current reference IDs, and record model/version, latency, validation outcome and
usage with secrets excluded. Confirm timeout/cancellation and refusal paths with
fakes first. A successful request would verify transport/schema wiring, not reasoning
quality. Broader model evaluations, artifact-based candidate generation and budgets
need separate reviewed increments.
