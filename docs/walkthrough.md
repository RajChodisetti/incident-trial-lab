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
