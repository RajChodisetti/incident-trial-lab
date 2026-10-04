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
| 6 — Portfolio presentation | Walkthrough, architecture, repeatable demo, limitations and related-work comparison; optional read-only Dynatrace evidence adapter | Raj can explain decisions; no commercial tenant required for core demo; explicit public-release decision |

Milestones 1–3 are implemented; verification and explicit coverage limits are
recorded in each evidence report. Milestone 3 checks sampled resource budgets;
peak sampling and statistically rigorous benchmarks remain future work. Next is
the remaining milestone 4 work, only after parent/user continuation. The first
lifecycle slice provides cancellation, deadlines, and owned cleanup; durable
restart recovery and bounded worker concurrency remain pending. Docker
Desktop startup and official project-local kind installation were authorized for
milestone 2. Helm remains optional. Never use the current kubectl context as an implicit target: create a
dedicated kind cluster and explicitly select its context for every cluster command.

AI API credentials are unnecessary now. When milestone 5 requires them, load the
relevant credential skill and resolve reuse/new credentials with Raj then.
