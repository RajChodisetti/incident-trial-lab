package presentation

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/RajChodisetti/incident-trial-lab/internal/provenance"
)

func fixture(t *testing.T) (Bundle, provenance.Spec, string) {
	t.Helper()
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	spec, err := provenance.CurrentSpec(root)
	if err != nil {
		t.Fatal(err)
	}
	b, err := Load(root, spec)
	if err != nil {
		t.Fatal(err)
	}
	return b, spec, root
}
func TestLinkedRecordedEvidence(t *testing.T) {
	b, _, _ := fixture(t)
	v, err := makeView(b)
	if err != nil {
		t.Fatal(err)
	}
	if v.Required != 283 || v.Observed != 282 || len(v.Rows) != 3 || len(v.Cases) != 6 {
		t.Fatal("incomplete presentation")
	}
	if v.Cases[3].Required != 370 || v.Cases[3].Values[0] != 368 || v.Cases[3].Values[1] != 368 || v.Cases[3].Values[2] != 370 {
		t.Fatal("regression evidence drift")
	}
}
func TestRejectBrokenProof(t *testing.T) {
	original, spec, _ := fixture(t)
	for name, mutate := range map[string]func(*Bundle){
		"capture payload": func(b *Bundle) { b.Captures.Captures[2].Evidence.Behavior.Observations[0].RawResponse = "forged" },
		"missing capture": func(b *Bundle) { b.Captures.Captures = b.Captures.Captures[:7] },
		"source":          func(b *Bundle) { b.Replay.Receipts["fixed"].SourceBinding.SourceSHA256 = strings.Repeat("0", 64) },
		"receipt": func(b *Bundle) {
			r := b.Replay.Receipts["fixed"]
			r.CandidateRunID = "0000000000000000"
			b.Replay.Receipts["fixed"] = r
		},
		"missing binding": func(b *Bundle) {
			r := b.Replay.Receipts["fixed"]
			r.SourceBinding = nil
			b.Replay.Receipts["fixed"] = r
		},
		"budget": func(b *Bundle) {
			r := b.Replay.Budgets["fixed"]
			r.MaxObservedWorkingSetBytes = 1 << 30
			b.Replay.Budgets["fixed"] = r
		},
		"cleanup": func(b *Bundle) {
			r := b.Replay.Cleanup[b.Replay.Trace.Baseline.RunID]
			r.Complete = false
			b.Replay.Cleanup[b.Replay.Trace.Baseline.RunID] = r
		},
		"refusal hidden": func(b *Bundle) { b.Refusal.Trace.StopReason = "verified_recovery" },
		"hash":           func(b *Bundle) { b.Replay.CaptureHashes[b.Replay.Trace.References[0].ID] = strings.Repeat("0", 64) },
		"not audited":    func(b *Bundle) { b.Replay.ResourcesAbsent = false },
	} {
		t.Run(name, func(t *testing.T) {
			raw, _ := json.Marshal(original)
			var b Bundle
			if err := json.Unmarshal(raw, &b); err != nil {
				t.Fatal(err)
			}
			mutate(&b)
			if Validate(b, spec) == nil {
				t.Fatal("bad proof accepted")
			}
		})
	}
	changed := spec
	changed.Workload = strings.Repeat("0", 64)
	if Validate(original, changed) == nil {
		t.Fatal("scenario drift accepted")
	}
}
func TestMissingBadEvidenceReplacesStalePresentation(t *testing.T) {
	_, spec, source := fixture(t)
	for _, bad := range []string{"missing", "changed", "truncated gzip", "symlink"} {
		t.Run(bad, func(t *testing.T) {
			root := t.TempDir()
			dir := filepath.Join(root, "docs", "evidence")
			if err := os.MkdirAll(dir, 0700); err != nil {
				t.Fatal(err)
			}
			for _, name := range []string{"source-bound-replay.json", "source-bound-first-refusal.json", "source-bound-replay-captures.json.gz"} {
				data, err := os.ReadFile(filepath.Join(source, "docs", "evidence", name))
				if err != nil {
					t.Fatal(err)
				}
				if err = os.WriteFile(filepath.Join(dir, name), data, 0600); err != nil {
					t.Fatal(err)
				}
			}
			if err := Export(root, spec); err != nil {
				t.Fatal(err)
			}
			for _, name := range []string{"index.html", "comparison.html", "provenance.html"} {
				data, err := os.ReadFile(filepath.Join(root, ".build", "demo", name))
				if err != nil {
					t.Fatal(err)
				}
				text := string(data)
				if !strings.Contains(text, "Live Decisions integration pending") || !strings.Contains(text, "Recorded deterministic replay") || strings.Contains(text, "/Users/") || strings.Contains(text, "raw_response") {
					t.Fatal("unsafe or misleading page")
				}
			}
			target := filepath.Join(dir, "source-bound-replay.json")
			switch bad {
			case "missing":
				if err := os.Remove(target); err != nil {
					t.Fatal(err)
				}
			case "changed":
				if err := os.WriteFile(target, []byte(`{}`), 0600); err != nil {
					t.Fatal(err)
				}
			case "truncated gzip":
				if err := os.WriteFile(filepath.Join(dir, "source-bound-replay-captures.json.gz"), []byte("bad gzip"), 0600); err != nil {
					t.Fatal(err)
				}
			case "symlink":
				if err := os.Remove(target); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(filepath.Join(source, "docs", "evidence", "source-bound-replay.json"), target); err != nil {
					t.Fatal(err)
				}
			}
			if Export(root, spec) == nil {
				t.Fatal("bad input rendered")
			}
			for _, name := range []string{"index.html", "comparison.html", "provenance.html"} {
				data, _ := os.ReadFile(filepath.Join(root, ".build", "demo", name))
				if !strings.Contains(string(data), "Evidence unavailable") || strings.Contains(string(data), "verified recovery") {
					t.Fatal("stale success retained")
				}
			}
			if _, err := os.Stat(filepath.Join(root, ".build", "demo", "evidence.json")); !os.IsNotExist(err) {
				t.Fatal("stale JSON retained")
			}
		})
	}
	if Generate(t.TempDir()) == nil {
		t.Fatal("missing scenario accepted")
	}
}
