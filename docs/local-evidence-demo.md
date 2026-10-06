# Local recorded-evidence demo

The presentation is a three-page, dependency-free local HTML view of the actual
source-bound replay. It presents recorded results, not a running experiment or
live model. The private repository contains the renderer, evidence, screenshot
assets and captions; no website was hosted or LinkedIn post published.

## Open and explain it

From the project root, using the existing Go toolchain:

```sh
go run ./cmd/evidence-demo
open .build/demo/index.html
```

No server or internet connection is needed. Native links move between incident,
comparison and source/runtime views. The provenance view has expandable full
hashes/receipts, and each page links the audit JSON. Generated pages are ignored by
Git and can be recreated from the committed inputs.

Suggested walkthrough:

1. **Incident:** explain why applying/truncating a discount per unit loses a cent.
   Show required 283¢ versus observed 282¢, then the separate candidate verdicts.
2. **Comparison:** explain why the misleading variant passes the original request
   but returns 368¢ instead of 370¢ on a different order. All 20 measured cycles
   contribute to the verdicts; displayed case values are the actual first cycle.
3. **Source/runtime:** follow source and logic hashes to executable bytes, image
   config/pod identity and a receipt with independent assertions and owned cleanup.
   Expand full hashes if asked. Explain that this assumes a trusted local builder,
   is not signed attestation and grants no universal production-safety claim.

Every page identifies deterministic/scripted execution, pending live Decisions
integration and an incomplete live-agent MVP. The initial inconclusive lock refusal
remains visible. Resource observations are an interval CPU delta and two memory
endpoints, not a performance improvement or peak-memory measure. Historical audit
times do not grant current trial authority.

## Validation before rendering

The generator anchors the two reviewed JSON snapshots by their file SHA-256,
rejects unknown JSON fields, and checks current scenario/image specification,
receipt/source/logic linkage, budgets, audit timing, complete scoped cleanup and
the preserved refusal. It decompresses the committed
[actual sanitized capture bundle](evidence/source-bound-replay-captures.json.gz)
with a two-MiB bound and recomputes all eight original Go-serialized capture hashes.
Its trace, receipts, budgets and trial identities must match the reviewed summary.
The compressed bytes come from the existing replay; no new trial ran here.

These are local integrity/linkage checks under trusted version control, not a new
runtime observation or cryptographic signature. The demo is intentionally a fixed
snapshot of loop `f2261299764ffd63`. Scenario/source/pin drift requires explicit
evidence review and a new snapshot; it cannot silently update the visible result.

Missing, changed, malformed, oversized or symlinked input is rejected. Failed
generation replaces all three prior result pages with **Evidence unavailable** and
removes the stale JSON export. It returns nonzero, exposes no private error path
in the page and starts no trial. Tests cover payload/hash tampering, missing source
proof, receipt/run mismatch, budget violations, missing cleanup, hidden refusal,
changed scenario, missing/changed snapshots, invalid gzip and stale-page removal.

## Actual screenshots and QA

Recreate the images on this Mac using already installed Chrome:

```sh
python3 scripts/capture-demo.py
```

The script first validates/generates the pages, then uses fresh temporary profiles
and bounded headless captures. It captures only local page content, not desktop
windows or a browser address bar, and terminates only processes it started.
No dependencies are downloaded. Chrome must already exist; absence is an explicit
blocker. The [manifest](screenshots/manifest.json) binds the final three 1440×1200
PNGs to their generated HTML hashes and replay identity.

On 2026-10-06 all three final images were visually inspected for readable framing,
complete scope/uncertainty footers and absence of credentials, private paths,
account information and unrelated windows. The earlier 1000-pixel frames clipped
two footers and were replaced. No screenshot is AI-generated or a staged live trial.
Captions are in [the screenshot set](screenshots/README.md), ready for Raj's review.

Focused presentation race tests and `./scripts/check.sh` passed formatting, all
package race tests and `go vet`; presentation coverage was 84.4%. The CLI was
actually exercised offline; its unit coverage remains zero. Browser captures were
verified separately from Go coverage. `git diff --check` and Python syntax checks
passed. There were no model calls, credential reads, installations, new clusters
or hosted CI in this slice.

This completes the local recorded-demo/screenshot preparation slice. Interview
review, broader related-work notes, substantive live-agent evaluation and local
Prometheus remain separate work. Provider documentation/access/pricing and a new
approved test budget still gate live Decisions integration.
