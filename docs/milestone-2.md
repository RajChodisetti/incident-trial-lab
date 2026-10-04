# Reproducible synthetic baseline

This is one intentionally broken application, not a trial engine or an agent.
The [contract](../scenarios/discount-rounding/contract.md) explains its seeded
rounding defect. The fixture and business assertions were written before the
implementation. Initial service tests failed because `Handler` did not exist;
after implementation they passed. The defect is intentionally retained.

On this Mac, Docker Desktop is already installed. Start it normally. Install
project-local kind with `./scripts/install-kind.sh`, which downloads the
[official release](https://github.com/kubernetes-sigs/kind/releases/tag/v0.33.0)
and verifies a recorded SHA-256. No sudo or global PATH/settings changes needed.
Follow the [official kind guide](https://kind.sigs.k8s.io/docs/user/quick-start/)
for background. The initial build supports macOS arm64/Go 1.26.2 only.

```sh
./scripts/check.sh
./scripts/build-baseline.sh
./scripts/run-baseline.sh
```

The build creates a static binary from this checkout and packages it in `scratch`.
A temporary project-specific BuildKit builder uses a pinned image and is removed
on exit. Artifact timestamps are normalized for reproducibility, while real Git
commit dates are preserved. No application image is pushed to a registry.
The runner checks the OCI manifest against the committed digest, loads it only
into a fresh named kind cluster, and applies a digest-pinned application manifest.
`imagePullPolicy: Never` prevents fallback to a remote application image.

The runner refuses to reuse a cluster with its name; every kubectl command uses
a dedicated generated kubeconfig and explicit context. It captures responses,
logs, pod image identity and resource statistics under ignored `evidence/`, then
removes only its own cluster and port forward on exit. A successful command means
the *known failure was reproduced*, not that the application was fixed.
Infrastructure errors or unexpected behavior exit nonzero and preserve evidence.

The application is non-root with a read-only root filesystem, no service-account
token, dropped capabilities, one replica, and CPU/memory requests/limits. These
are local containment measures, not a production-safety guarantee. Docker VM and
control-plane resource consumption are outside pod limits. Do not change other
clusters, Docker settings, or resources to run this demo.

Next: milestone 3 candidate trials, only after continuation. First define bad/good/
incomplete fixtures and repeated benchmark methodology; then compare a candidate
in fresh isolation. No candidate implementation, telemetry integration, API
credentials, MCP tools, or production adapter has been added in this increment.
