# Baseline provenance verification

Verified locally on 2026-10-04 UTC in the working tree based on `7439a8d`.
Service binaries, OCI image pins and existing supervisor/worker behavior were
unchanged. No live AI, new credential, hosted CI or paid resource was used.

`./scripts/check.sh` passed: formatting, all package race tests and `go vet`.
The new provenance package has 78.9% statement coverage; command entry points are
exercised by the real comparison, not covered by unit tests. `sh -n` passed for the
new wrapper, and `git diff --check` passed.

Unit tests reject changed fixtures, workloads, contracts, manifests and image locks;
expired/future experiments, stale/future baselines and old candidate runs; a boolean
baseline stub; altered raw responses, case roles/cycles and forged summaries;
incorrect runtime identities/resource pod IDs; reused baselines, candidates and
attempts; concurrent duplicate consumers; and canceled/failed/timed-out lifecycles.
Tests preserve failed and inconclusive receipts from valid observations, and reject
changed evidence when a previously stored receipt is revalidated. These unit data
are synthetic test fixtures, not claims about Kubernetes execution.

The real check was `./scripts/run-bound-comparison.sh`, followed by revalidation of
all three stored receipts using the final verifier, including applied-manifest
checks. Experiment: `dc2bae1ed65bb31f`. Baseline run:
`37e4fc48cde1fd79`; its fault reproduced 282 cents against the independent expected
283, with both controls passing. Each candidate had its own fresh kind cluster,
one warmup cycle and 120 measured requests.

| Candidate | Run ID | Recovery | Regressions | Final verdict |
| --- | --- | --- | --- | --- |
| ineffective | `c3933ee353f7ace4` | failed | failed | failed |
| misleading | `2f464b70a29ab4f7` | passed | failed | failed |
| fixed | `2eee49cd2e921f0b` | passed | passed | passed |

All three passed the existing **sampled** resource budget. The recorded CPU deltas
were 11,827,000, 11,627,000 and 11,262,000 nanoseconds respectively; maximum endpoint
working sets were 4,890,624, 2,768,896 and 4,886,528 bytes. These are scoped
observations, not peak-memory measurements, statistical benchmarks or improvements.

Actual completed candidate/attempt replay attempts were refused, as was baseline
re-registration. All four lifecycle journals reported completed outcomes and
verified cleanup. Exact owned cluster labels were independently queried afterward;
zero remaining containers were found. No unrelated containers were removed.

Compact receipt/specification, verdict, resource and cleanup evidence is committed
in [baseline-provenance-summary.json](evidence/baseline-provenance-summary.json).
Raw local evidence remains under `evidence/bound-comparison-HFnI9MCx`,
`evidence/experiment-dc2bae1ed65bb31f`, and the four `evidence/lifecycle-<run-id>`
directories. Raw evidence is ignored by Git. Receipt validity expires at
`2026-10-04T21:47:00.983214Z`; preserved historical outcomes do not become proof
for a new experiment. Report remote SHA and CI status separately after pushing.

The prior cancellation/worker tests passed unchanged. This increment did not run a
new real cancellation/crash matrix or extend host-wide admission, durable queues,
creation-gap cleanup, outage tolerance, peak metrics or adversarial attestation.
Marker-to-receipt interruption fails closed and needs review/fresh execution.

Next: a thin, typed local MCP contract with `create_experiment(scenario_id)`,
`start_trial(experiment_id, variant)`, `list_trials(experiment_id)`,
`get_trial(trial_id)` and `cancel_trial(trial_id)`. The implementation should own
fresh baseline execution and reservations, retain the current local worker bounds,
and expose lifecycle/cleanup/coverage/eligibility separately from business verdicts.
No arbitrary shell, apply tool, model credentials or live-agent claims are needed.
