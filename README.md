# Incident Trial Lab

Private working repository for Raj's Go + Kubernetes engineering portfolio.
The purpose is interviewable evidence of engineering depth, not a commercial product.
The working name may change before a public release; keep this repository private
until Raj makes that decision.

## Current scope

Milestones 1–3 supply a tested Go evidence evaluator, one reproducible faulty
synthetic service, and deterministic comparisons of three candidate images in
fresh dedicated local kind clusters. **There is no durable worker/controller,
agent, or MCP server yet.** Nothing here claims production safety or novelty.

Run the [baseline walkthrough](docs/milestone-2.md), then the
[candidate comparison walkthrough](docs/milestone-3.md).

The eventual experiment: reproduce a synthetic Go service incident from pinned
images/manifests, fixture data, a workload, and business assertions; confirm baseline
failure; test a candidate patch or fixed image in fresh isolated local trials;
compare recovery, regressions, and resource budgets; preserve raw evidence and
inconclusive outcomes. Use standard telemetry, including local Prometheus.

## Development

Requires Go 1.26.2. The foundation has no third-party dependencies.

```sh
./scripts/check.sh
```

The check script enforces formatting, runs tests with race detection, and runs
`go vet`. GitHub's manual workflow uses the same script. Hosted CI remains manual
until the account's no-cost Actions allowance is verified; do not enable automatic
runs or paid resources without resolving that constraint.

Start with [the milestone plan](docs/milestones.md),
[design decisions](docs/decisions.md), and [the walkthrough](docs/walkthrough.md).

## Boundaries

- Only synthetic data and a dedicated local kind cluster in later milestones.
- No employer repositories/data, production clusters, or paid services.
- No new credentials or permission expansion without the required confirmation.
- Human approval before any optional apply; evidence inspection does not authorize changes.
- No universal production cloning, guaranteed safety, or unmeasured improvement claims.
- Related work includes Dynatrace MCP/remediation/Guardian, KubeTective, and AIOpsLab;
  comparison and primary-source research belong in a later milestone.
