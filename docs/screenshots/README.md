# Recorded deterministic demo screenshots

These are actual headless-browser captures of validated persisted synthetic
evidence, not live Decisions execution or model reasoning. Prepared for Raj's
review; nothing has been posted to LinkedIn or made public.

| Asset | Suggested caption |
| --- | --- |
| [01-incident.png](01-incident.png) | A one-cent checkout error reproduced in Go on Kubernetes. Fresh isolated trials separate recovery, regressions and sampled resource checks; this replay uses a scripted proposer. |
| [02-comparison.png](02-comparison.png) | Passing the incident is not enough: the misleading candidate fixes the original order but fails a different rounding case. Independent regression assertions expose overfitting; the general fix passes both. |
| [03-provenance.png](03-provenance.png) | Traceable evidence connects candidate source and logic hashes to executable bytes, pinned runtime identity and independently checked receipts. Local ownership and cleanup are audited; this is not signed build attestation or a production-safety guarantee. |

All screenshots retain the historical audit date, initial inconclusive refusal,
sampled-measurement limits and pending live-agent integration label. The
[manifest](manifest.json) records actual PNG and source HTML hashes. Full evidence,
verification, reproduction and limits are in [the demo walkthrough](../local-evidence-demo.md).
