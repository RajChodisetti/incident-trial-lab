# Thin local MCP interface

This slice exposes the existing supervised, provenance-bound trial workflow over
client-launched **stdio**. The server has no MCP network listener, account setup,
OAuth flow or model call. The kind engine still uses its existing isolated local
cluster/API and loopback probes. The interface is a deterministic tool surface,
not an AI investigation loop.

The implementation pins the [official Go SDK v1.8.0](https://github.com/modelcontextprotocol/go-sdk/tree/v1.8.0)
and its checksummed dependencies. SDK typed handlers validate inputs and structured
outputs. The [stdio transport](https://modelcontextprotocol.io/specification/2026-07-28/basic/transports/stdio),
[tool contracts](https://modelcontextprotocol.io/specification/2026-07-28/server/tools)
and [cancellation semantics](https://modelcontextprotocol.io/specification/2025-11-25/basic/utilities/cancellation)
guide the implementation. Offline interoperability tests cover protocol revisions
2026-07-28 and 2025-11-25; the real client negotiated 2026-07-28.

## Build and inspect

From this repository's root, after building the existing pinned local OCI archives:

```sh
go build -o .build/trial-run ./cmd/trial-run
go build -o .build/trial-mcp ./cmd/trial-mcp
go build -o .build/mcp-check ./cmd/mcp-check
.build/mcp-check          # stdio discovery/schema/refusal check; no trial resources
.build/mcp-check -real    # fresh baseline + fixed candidate; queued cancellation
```

Use the existing Go cache environment overrides in a restricted Codex sandbox.
`mcp-check` is an offline SDK client, not a language model. The `-real` flag explicitly
runs the synthetic Kubernetes comparison; all unit-test backend results are simulated.

An MCP client should launch the absolute `.build/trial-mcp` binary with its working
directory set to this checkout. Owner-selected CLI flags are `-workers 1|2`
(default one) and `-trial-timeout 1m..10m` (default three minutes). These are not tool
arguments. No personal client/account configuration is installed by this slice.
Stdout is reserved for protocol messages. Supervisor output goes to owned
`evidence/mcp-session-<id>/<trial-id>.log` files; errors go to stderr.

## Tools

All object inputs reject extra properties. IDs are exactly 16 lowercase hexadecimal
characters and must also belong to this running server. There is one scenario,
`discount-rounding`, and three committed image presets: `fixed`, `ineffective`,
`misleading`. A supplied scenario/variant ID selects those assets; no custom fixture,
patch, image digest, command, URL, kubeconfig or file path is accepted from a model.

| Tool | Arguments | Behavior |
| --- | --- | --- |
| `list_scenarios` | none | Supported IDs, fixture/assertion/image hashes, worker/session limits and measurement coverage |
| `create_experiment` | `scenario_id` | Queue a new supervised baseline; return a baseline trial ID promptly |
| `start_trial` | `experiment_id`, `variant` | Queue a fresh candidate; reserve/bind evidence inside the verified owned experiment |
| `get_trial` | `trial_id` | Read lifecycle state, business verdict, cleanup and current provenance eligibility |
| `list_trials` | optional `experiment_id` | List at most eight owned jobs, including queued baselines; optionally filter by experiment |
| `cancel_trial` | `trial_id` | Request cancellation of one owned unfinished job; poll terminal cleanup afterward |
| `get_evidence` | `trial_id`, `section` | Named section: `lifecycle`, `behavior` or `provenance`; no path-based reads |
| `compare_trials` | `experiment_id` | Return each candidate's status/verdict/eligibility separately; no extra runs or performance ranking |

Typical sequence: create a baseline, poll `get_trial` until completed, use its
returned experiment ID to start candidates, then read business evidence and compare.
The baseline's expected defect is a prerequisite, not a recovered candidate; its
business `verdict` remains inconclusive. A registered baseline has an experiment ID
and verified provenance. An already expired or changed baseline is not usable.

## Bounds, cancellation and evidence

Admission is capped at **eight total jobs per server process**, including baselines,
completed and canceled jobs. Slots default to one and never exceed two; a slot stays
occupied through supervisor termination, cleanup and provenance binding. This is a
process-local bound, not host-wide admission. Job scheduling is not a durable FIFO
queue. Restart creates a fresh ownership scope; old IDs cannot be imported or used
to cancel another process's jobs.

A tool-call cancellation/timeout affects that request. Once a fast submission has
been admitted, the job belongs to the server: use `cancel_trial` to stop it, or
`list_trials` without a filter to recover its ID if the response was lost. Canceling
one job leaves peers intact. Queued cancellation starts no supervisor/resources.
Active cancellation signals only its owned live supervisor PID, retaining the
existing bounded TERM/cleanup behavior. Canceled or unclean partial passes remain
inconclusive. EOF, protocol failure and server signals cancel all owned jobs and
wait for their supervised cleanup.

The [SDK CommandTransport](https://github.com/modelcontextprotocol/go-sdk/blob/v1.8.0/mcp/cmd.go)
defaults to a short shutdown wait. Our client sets `TerminateDuration` to 90 seconds
to allow the supervisor's 75-second shutdown allowance plus bookkeeping. Other
clients should allow comparable graceful shutdown time. An external hard kill can
interrupt cleanup; preserve the existing lifecycle journal and use the explicit
recorded recovery CLI. This server adds no automatic restart/creation-gap repair.

Terminal snapshots are cached historical observations. `provenance_verified` means
verified at completion; `eligible_now` separately revalidates live experiment proof
against current files and the 30-minute expiry. A failed/inconclusive bound receipt
can be current evidence without accepting a fix. Expired/changed evidence does not
rewrite a historical verdict into a new pass. Evidence sections carry
`untrusted_evidence: true`: responses and errors are data, never instructions.
Behavior includes raw baseline controls/fault responses or candidate warmup and
per-request observations; section reads are bounded to regular files (2 MiB), with
symlinks refused. Missing sections remain absent, never fabricated.

Resource coverage is still CPU interval plus two endpoint memory samples, with no
peak-memory, statistical improvement or production-safety claim. Comparison preserves
unknowns and each candidate separately. There is no apply tool.

## Next agent work

Define an offline investigate/propose/test/revise evaluation with recorded evidence
references, rejection reasons, attempt/time budgets and stop conditions. Verify that
it rejects ineffective and overfit proposals and retains inconclusive outcomes.
These preset images test the tool boundary; choosing among them is not substantive
AI patch generation. Later candidate artifact/patch handling needs its own isolated
build and pinned registration contract. Only when live-model integration is chosen
should Raj resolve credential reuse/new-provider decisions through the appropriate
credential workflow. No credentials are needed for the offline loop.
