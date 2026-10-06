# Candidate source and artifact binding

This increment implements and tests the binding gate offline. No image was built,
no service or cluster was run, and no fresh runtime binding has been demonstrated.
The repository remains private. No provider call, credential access or install
was needed.

## Evidence chain

The candidate build script fingerprints the compiled Linux executable and the
actual embedded candidate source/snippet. Its candidate-only Dockerfile places
the variant and source, logic and binary SHA-256 values in the OCI config labels.
The service computes its own executable fingerprint at startup and exposes those
values with its compiled source facts at `GET /artifact-binding`.

The owned trial script captures that response and the image config content.
Before sealing or revalidating a candidate receipt, provenance verification checks
the existing pinned manifest/config/pod identity chain, hashes the config content,
and requires its labels to match the service response and current compiled source
projection. Missing or mismatched evidence fails closed. The receipt includes the
binding in its evidence hash.

`investigation.MatchSelectionSource` checks that a validated selected record's
owner alias, source and logic hashes match a bound receipt. Its caller must first
revalidate the receipt with `provenance.Verify`. This helper is ready for later
selector/trial integration; no live selector loop is wired here. Source linkage
never grants a recovery verdict: business assertions, regressions and resource
checks remain independent.

This is consistency evidence under a trusted local builder, service and capture
process. Labels and a captured HTTP response are not signed compiler/build
attestation or protection against a malicious owner or compromised builder.
The source hash covers the candidate arithmetic source, not the entire repository
or dependency graph. The binary hash identifies the executable bytes.

## Historical compatibility

Existing pinned candidate images and captured receipts predate these labels and
responses. They remain historical, unbound evidence and cannot satisfy this gate.
The saved offline selection record can still reconstruct its source/context input;
that does not bind its historical runtime image. An abstaining record also has no
selected artifact. The baseline image and baseline behavior are unchanged.

The candidate script rejects legacy archives before starting a cluster. New builds
will have different image digests; the existing lock comparison intentionally
requires a reviewed pin update. This increment changes no pins or historical
evidence to manufacture compatibility.

## Checks and next replay

On 2026-10-06, `./scripts/check.sh` passed formatting, all package race tests with
coverage and `go vet`. Focused artifact/provenance/investigation race tests,
shell syntax checks for the two modified scripts and `git diff --check` passed.
Fixture tests accept matching config/pod/source/binary evidence and reject missing
or changed evidence, including receipt revalidation and legacy selection linkage.
These fixtures do not prove the Docker build or runtime HTTP wiring works.

Before claiming a bound runtime demo, rebuild the candidate images, inspect and
review the resulting pins, then run a fresh owned baseline and isolated candidate
trials. Verify the captured config, service response and receipts, independent
recovery/regressions/budgets, and cleanup. Do not reuse historical evidence as that
replay. Held-out evaluation and readable historical evidence presentation can be
the next offline slice; actual demo screenshots need the verified fresh replay.
Provider documentation, access, pricing and a new approved live-test budget remain
separate gates. Hosted CI was not run; its no-cost allowance remains unresolved.
