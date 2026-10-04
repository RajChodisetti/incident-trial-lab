# Milestone 2 verified evidence

Observed 2026-10-04 UTC on Raj's macOS arm64 machine. Source code checked out for
this final run: `ff14129c6da2f8e058feaabeaa077cd392046e3c`. Later documentation-only
commits preserve these results. No dates/activity were fabricated.

## Behavioral result

[Recorded checker output](evidence/milestone-2-check.json):

| Request | Business assertion (cents) | Observed | Result |
| --- | ---: | ---: | --- |
| No-discount control | 315 | 315 | Pass |
| Single-unit control | 94 | 94 | Pass |
| Bulk discount | 283 | 282 | Fail, expected seeded defect |

All responses were HTTP 200. `baseline_reproduced: true` and runner exit status 0
mean this known defect was reproduced, **not** that recovery occurred.
Fixture SHA-256: `a747f28e9f56d745a687afebbe55a6ec1b1a390dffac4fe7e0ce997eb7b02d28`.

## Artifact and runtime identity

- kind v0.33.0, official macOS arm64 binary SHA-256:
  `0c8c7dbe5e23594a198b786c4bc13dacc101fa6196b0cb0b23a1ca44e61f4b4f`.
- Kubernetes node: `kindest/node:v1.36.4@sha256:099e049362a1526b2db71494e1947aae99bd16290d7c895f2b7ea312e3cbfaed`.
- Application OCI manifest:
  `sha256:8babadea39ab217d192db5dd32c43ab0d73a92eee633d6aac54a6182be558d9e`.
- Two independently rebuilt, timestamp-normalized artifacts produced that same
  manifest digest. The runner rejects an archive differing from its pinned manifest.
- Running config matched the manifest's config:
  `sha256:911af812795333f2097f355104cb5a0dc91ed5a0da7e25b0c4a3c0cf05e91711`.
- containerd's runtime image identity was an imported OCI index,
  `sha256:0e466455be561a5e8acde8c03f41de421aab4e6d25d1e7238a623ad439978ba9`.
  Its contents referenced the locked application manifest; the runner verified
  this relationship. The pod was ready with zero restarts.

Full local bundle (ignored by Git):
`evidence/baseline-20261004T055347Z-85118/`. It includes raw responses, fixtures,
manifest, node/pod JSON, image identities, logs, cleanup output, and exit status.
The temporary kubeconfig was removed after teardown.

## Checks and limits

- `./scripts/check.sh`: formatting, race-enabled uncached tests, and vet passed.
- Statement coverage: `internal/synthetic` 100%, `internal/evaluation` 100%,
  `internal/baseline` 96.4%. CLI main packages have 0% unit statement coverage;
  the service/checker were exercised by the actual cluster run.
- `sh -n scripts/*.sh` and `git diff --check`: passed.
- A tag-only `:latest` application manifest was rejected by the pin validator.
- The final fresh-cluster run completed with expected behavior and automatic cleanup.
- An earlier OCI naming attempt failed before deployment and cleaned up its cluster;
  its incomplete evidence is retained locally. An earlier non-normalized rebuild
  differed in digest; layer timestamp normalization corrected this before final lock.
- Hosted GitHub CI remains manual and was **not run**. No paid resource, scope
  expansion, AI credential, production access, or candidate fix was introduced.
- No throughput/latency benchmark or pod-level measured resource-budget comparison
  was run; those belong to milestone 3. These numbers are not improvement claims.

## Cleanup and footprint

Verified no containers remain with the project cluster label and no project builder
container or builder volume remains. The port-forward child was stopped/waited.
Docker Desktop remains running as approved; no other containers or clusters were
stopped. The default Docker context remains `desktop-linux`; the default kubeconfig
was not used or modified.

During the final run, a single Docker stats snapshot of the entire control-plane
container showed **658.5 MiB RAM and 15.45% CPU**. This is a startup/idle snapshot,
not a benchmark or application-only measurement. Application requests are 25m CPU /
16Mi memory; limits are 250m CPU / 64Mi memory.

Reusable caches remain: kind node image logical size 908,755,098 bytes, BuildKit
image logical size 247,682,541 bytes (Docker storage sharing/compression differs);
project tools about 9.5 MiB, build artifacts about 12 MiB, small evidence bundles,
and the isolated `/tmp/incident-trial-lab-go-cache` about 354 MiB. None are active
trial workloads. No broad Docker prune was performed.

Next milestone: define and implement one candidate trial with fresh isolation,
recovery/regression assertions, repeated workload methodology, and resource-budget
evidence. Wait for parent/user continuation before starting it.
