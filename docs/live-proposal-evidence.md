# First live proposal result

One actual API request ran on Raj's Mac from **2026-10-05T07:27:29.930093Z** to
**2026-10-05T07:27:32.535670Z**, after the credential reuse decision and US$2 total
first-test authorization. HTTP 200; model `gpt-5.4-mini-2026-03-17`;
status `valid_proposal_not_executed`. No second request, cluster, candidate trial
or apply action ran in this slice.

| Measurement | Observed |
| --- | --- |
| Attempt count | 1 |
| Input / cached input tokens | 453 / 0 |
| Output / reasoning tokens | 97 / 0 |
| Total tokens | 550 |
| Estimated standard token charge | US$0.00077625 |
| Conservative reservation before sending | US$0.613824 |
| Authorized first-test cap | US$2 |

The estimate uses the dated official rates in [the probe contract](live-proposal-probe.md).
It is calculated from returned usage, not a billing invoice or a claimed account
balance. No billing settings were queried or changed.

The model chose `fixed`, citing both exact historical baseline references. Strict
JSON decoding, allowlisted variant, rationale length, distinct references and
latest behavior/provenance linkage all passed. The rationale correctly describes
the supplied general formula and explicitly avoids claiming recovery. The prompt
already provided the candidate meanings, so this is **API/schema/evidence wiring
evidence**, not independent diagnosis, an agent benchmark, measured engineering
improvement or a verified repair. Existing offline trial results remain separate.

Local checks: repository `scripts/check.sh` passed formatting, race tests and vet.
The new package passed focused race tests again after adding the evidence-hash
tamper test. Tests cover concurrent single-attempt admission; auth/quota/transport
failure consuming the attempt; secret-free error/report handling; refusal and
truncation; duplicate/stale/unknown references; unknown JSON fields and trailing
JSON; bounded requests with no tools; canceled requests; cost bounds; ignored-file
loader symlink rejection; and raw-observation exclusion/hash mismatch rejection.
Tests use fake HTTP responses and a distinct fake credential. They do not test
provider billing enforcement or account limits. CLI unit coverage remains zero;
its authorized live invocation exercised the actual happy path.

The compact sanitized report is [live-proposal-first-test.json](evidence/live-proposal-first-test.json).
The ignored local ledger in `evidence/live-first-test/` preserves the consumed
attempt; no credential or raw provider response was copied into evidence. The
source baseline payloads remain in the ignored offline capture bundle, with their
hashes verified before projection. Re-running the command is blocked by its saved
reservation. Deleting that marker does not authorize another call.

Next: review this result before planning a separate bounded live-proposer/MCP
trial increment. A fresh baseline and new authorization/budget are needed for
any later live evaluation; historical references cannot authorize current trials.
