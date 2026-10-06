# Fresh deterministic source-bound runtime replay

This increment uses actual local kind trials through a dedicated one-worker stdio
MCP session. The proposer is the predetermined `ineffective → misleading → fixed`
sequence. There is no model diagnosis, Decisions API call or automatic patch
generation. Work began from `7a3cf0e3c7fb217aa7a088be443a897863104e44` with reviewed
candidate pin changes; source code and scenario assertions were unchanged.

## Build and pin review

Existing tools were Go 1.26.2 on macOS arm64, Docker 29.6.2, Buildx 0.35.0,
kubectl 1.36.1 and the project-local kind 0.33.0. Docker access required approved
sandbox escalation. No installation, credential read, model request, hosted CI or
shared-Docker shutdown occurred. The existing pinned BuildKit and kind node images
were used; only newly created project builders and disposable owned clusters were
eligible for cleanup.

The rebuilt baseline matched the frozen manifest digest
`sha256:8babadea39ab217d192db5dd32c43ab0d73a92eee633d6aac54a6182be558d9e`.
Each candidate build stopped on its changed digest for deliberate review. Review
rehashes the OCI manifest, config and layer; checks config labels against source/
logic/binary metadata; checks source bytes against the actual candidate source;
and compares the executable inside the layer byte-for-byte with the built binary.
Only then were [candidate pins](../deploy/local/candidates.json) updated. A complete
subsequent rebuild passed all three lock checks with identical digests. Reproducible
image timestamps do not alter actual commit or trial timestamps.

The [binding implementation](source-artifact-binding.md) connects the pinned image
to pod runtime identity, captured config content, the service's own executable
fingerprint and compiled source/snippet hashes. A receipt still needs independent
behavior, resource and cleanup checks; source linkage grants no recovery verdict.

## Preserved initial refusal

The initial loop `1a8231f067d7d6f1` ran from
`2026-10-06T13:32:32.555477Z` to `2026-10-06T13:34:12.110570Z` and stopped
`trial_not_verified`, with an inconclusive loop verdict. Baseline run
`142b13809b2309a7` reproduced the fault; ineffective run `6b54cc121ff2098f`
recorded failed recovery/regressions and a source-bound receipt. One provenance
evidence read reported a temporary journal-lock conflict and `eligible_now=false`,
so the loop stopped rather than accepting that result or retrying silently.

All 50 subsequent receipt revalidations passed. An independent audit recomputed
all four capture hashes, revalidated that receipt and its binary binding, and
confirmed both supervisors' cleanup plus absence of their exact owned Docker
labels and temporary builders. This does not turn the refused loop into success.
The conflict's cause remains unproven; the gates were not relaxed or changed.
A single separate fresh experiment was then run. The initial trace, receipt and
actual audit time remain in
[source-bound-first-refusal.json](evidence/source-bound-first-refusal.json).

## Reproduction and limits

The successful fresh loop `f2261299764ffd63` ran from
`2026-10-06T13:37:10.932551Z` to `2026-10-06T13:39:59.594621Z`, within its
eight-minute budget. Experiment `f138b3bd3b9b7c19` used baseline run
`a7d28ecce6aa0735`, which reproduced the known 283-versus-282-cent failure.

| Scripted candidate | Actual run | Recovery / regressions | Measured failures | Outcome |
| --- | --- | --- | ---: | --- |
| ineffective | `c5fb26c78fabe804` | failed / failed | 40 | no improvement |
| misleading | `95509cf013583101` | passed / failed | 20 | regression |
| fixed | `d21415b4f502f278` | passed / passed | 0 | verified recovery |

All three had zero unknown observations and passed the predefined sampled budget.
Observed CPU interval deltas were respectively 12,273,000, 12,586,000 and
11,453,000 nanoseconds; maximum of the two working-set samples was respectively
4,911,104, 4,952,064 and 7,053,312 bytes. These are raw scoped observations,
not improvements or peak values.

The loop read eight behavior/provenance responses and made one final
`compare_trials` call confirming current fixed recovery. An independent audit at
`2026-10-06T13:40:30.800972Z` recomputed all eight Go-serialized capture hashes,
revalidated all three receipts against current evidence, and compared each runtime
binding with a fresh fingerprint of its built executable. Thus the new config/
source/logic/binary gate actually held at runtime for this replay. All four
supervisor records had completed outcomes and verified cleanup; independent Docker
queries for only their exact owned labels found no remaining containers. The two
temporary project builders were also absent. No unrelated resource was removed.

The compact [source-bound-replay.json](evidence/source-bound-replay.json) preserves
the trace, hashes, receipts, pins, bindings, sampled budgets, actual audit time and
cleanup. Raw captures remain under `evidence/offline-loop-f2261299764ffd63`, with
the corresponding owned MCP/session, lifecycle and experiment bundles ignored by
Git. The initial refused experiment is retained separately. Six total trial runs
occurred across both experiments; neither failed/inconclusive evidence nor resource
cost was hidden by presenting only the successful run.

Build from the reviewed checkout with the existing prerequisites:

```sh
./scripts/build-baseline.sh
./scripts/build-candidates.sh
go build -o .build/trial-run ./cmd/trial-run
go build -o .build/trial-mcp ./cmd/trial-mcp
go build -o .build/offline-loop ./cmd/offline-loop
.build/offline-loop -real
```

This creates a new baseline and serial fresh candidate clusters; it never adopts
existing clusters or the default kubeconfig. Each loop has three attempts and an
eight-minute decision budget, followed by separately bounded cleanup. Application
limits remain 250m CPU / 64 MiB. Each candidate performs six workload cases over
20 measured cycles (120 requests), after warmup. These are the existing comparison
cases, **not** the separate 18-case offline held-out dataset.

The predefined resource gate is at most five container CPU-seconds over the
snapshot interval and at most 32 MiB at both working-set endpoints. It excludes
peak memory and throttling; no speedup or production-safety claim follows.
Revalidated receipts were current at their actual audit time, not indefinitely
eligible. Build/config/HTTP capture evidence assumes the trusted local builder and
owner; this is not signed or adversarial build attestation.

Local `./scripts/check.sh` passed formatting, all package race tests with coverage
and `go vet`. `git diff --check` is checked before commit. No implementation change
was needed for the build/source-binding paths. Historical pre-binding evidence is
kept intact and remains unbound. Provider contract/access/pricing, approved live
test budget and hosted CI allowance remain separate unresolved decisions.

Next is readable deterministic evidence presentation and actual demo screenshot/
caption preparation for Raj's review. This milestone does not publish screenshots,
change repository visibility or demonstrate a live Decisions agent. The transient
lock refusal remains a recorded reliability limitation with an unproven cause.
