# Interview walkthrough: milestone 1

1. Describe the problem: candidate incident fixes need reproducible experiments
   and inspectable evidence, including uncertainty.
2. Read `internal/evaluation/evaluation.go`: completion and check statuses determine
   a conservative verdict. Missing evidence cannot certify a candidate.
3. Run `./scripts/check.sh`; explain the test cases for absent, malformed, failed,
   passing, and incomplete evidence.
4. Explain the boundary: this evaluator knows nothing about Kubernetes and cannot
   guarantee that a caller supplied every required business assertion.
5. Describe the next experiment: a pinned synthetic baseline in a dedicated kind
   cluster, with expected failures specified before implementing candidate trials.

Implementation ownership should be described accurately: Raj is developing this
portfolio with Codex assistance and reviewing incremental decisions, tests, and
commits. Only describe features demonstrated by the current code and evidence.


## Interview walkthrough: milestone 3

1. Show the frozen baseline reproducing the cent-rounding defect.
2. Explain why a fix that special-cases that one order is insufficient: the new
   bulk-order regression rejects it despite successful incident recovery.
3. Walk through distinct baked candidate images, digest locks, fresh clusters,
   baseline gating, raw observations, and conservative inconclusive verdicts.
4. Explain exactly what the kubelet CPU interval and two memory samples cover,
   and why they do not justify a peak-memory or performance improvement claim.
5. Name what is still missing: durable workers, lifecycle fault injection, broad
   workloads, and agent/MCP integration. Only claim what current evidence shows.


## Interview walkthrough: milestone 4 lifecycle slice

1. Explain why shell traps alone do not bound child-process behavior or prove cleanup.
2. Walk through a separate process group, TERM/KILL grace, output-pipe wait bounds,
   and cleanup using a fresh context after the execution context is canceled.
3. Show exact run-derived ownership names and label verification. Explain why this
   protects against accidental cross-project cleanup, not a malicious Docker peer.
4. Show real deadline/SIGTERM/failure/success evidence and the preserved sentinel.
5. State the gap: supervisor death/restarts and durable concurrency remain untested
   and unimplemented; do not claim guaranteed cleanup or a distributed controller.


## Interview walkthrough: hard-crash reconciliation

1. Show why SIGKILL bypasses the supervisor's ordinary cleanup.
2. Explain atomic intent records, OS-held locks, and the recorded child fingerprint.
3. Show strict node-ID/name/label matching and refusal of replacement resources.
4. Walk through real leftover-node cleanup and repeat recovery; explain why the
   original business outcome remains inconclusive even after cleanup succeeds.
5. State the creation-to-snapshot gap and cooperative-host assumption, rather than
   claiming complete crash safety or distributed durability.
