// Package presentation renders recorded synthetic evidence, never current trial authority.
package presentation

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"time"

	"github.com/RajChodisetti/incident-trial-lab/internal/artifact"
	"github.com/RajChodisetti/incident-trial-lab/internal/candidates"
	"github.com/RajChodisetti/incident-trial-lab/internal/investigation"
	"github.com/RajChodisetti/incident-trial-lab/internal/lifecycle"
	"github.com/RajChodisetti/incident-trial-lab/internal/provenance"
	"github.com/RajChodisetti/incident-trial-lab/internal/trial"
)

type Summary struct {
	CaptureHashes   map[string]string                  `json:"capture_hashes_recomputed"`
	Execution       string                             `json:"execution"`
	Cleanup         map[string]lifecycle.CleanupResult `json:"owned_cleanup"`
	ResourcesAbsent bool                               `json:"owned_resources_rechecked_absent"`
	RevalidatedAt   time.Time                          `json:"revalidated_at"`
	BindingMatches  bool                               `json:"runtime_binding_matches_built_executables"`
	Budgets         map[string]trial.Budget            `json:"sampled_resource_budgets"`
	SourceCheckout  string                             `json:"source_checkout"`
	Receipts        map[string]provenance.Receipt      `json:"source_bound_receipts"`
	BuildersAbsent  bool                               `json:"temporary_owned_builders_rechecked_absent"`
	Trace           investigation.Trace                `json:"trace"`
	Proposer        string                             `json:"proposer"`
}
type Bundle struct {
	Replay, Refusal Summary
	Captures        investigation.Result
}

func decode(body []byte, value any) error {
	d := json.NewDecoder(bytes.NewReader(body))
	d.DisallowUnknownFields()
	if d.Decode(value) != nil || d.Decode(new(any)) != io.EOF {
		return errors.New("malformed evidence")
	}
	return nil
}
func digest(body []byte) string { return fmt.Sprintf("%x", sha256.Sum256(body)) }
func jsonHash(v any) string     { b, _ := json.Marshal(v); return digest(b) }
func boundedRead(path string) ([]byte, error) {
	s, err := os.Lstat(path)
	if err != nil || !s.Mode().IsRegular() || s.Size() > 2<<20 {
		return nil, errors.New("evidence missing or exceeds bound")
	}
	return os.ReadFile(path)
}

// Load anchors these two reviewed snapshots, then checks their linked raw captures.
// Hashes are local integrity checks under trusted version control, not signatures.
func Load(root string, spec provenance.Spec) (Bundle, error) {
	var b Bundle
	for _, f := range []struct {
		name, hash string
		dst        *Summary
	}{
		{"source-bound-replay.json", "b68b56d629b2ddb2d01453800d9f9702adfddc9679cd5ef4c556e92a85e4345c", &b.Replay},
		{"source-bound-first-refusal.json", "f064a54f027c0bf538c8a3478309f60712eb15884169703ff9ac1b5581627ef0", &b.Refusal},
	} {
		body, err := boundedRead(filepath.Join(root, "docs", "evidence", f.name))
		if err != nil {
			return b, err
		}
		if digest(body) != f.hash {
			return b, errors.New("reviewed snapshot integrity changed")
		}
		if err = decode(body, f.dst); err != nil {
			return b, err
		}
	}
	body, err := boundedRead(filepath.Join(root, "docs", "evidence", "source-bound-replay-captures.json.gz"))
	if err != nil {
		return b, err
	}
	z, err := gzip.NewReader(bytes.NewReader(body))
	if err != nil {
		return b, errors.New("invalid capture bundle")
	}
	raw, err := io.ReadAll(io.LimitReader(z, (2<<20)+1))
	closeErr := z.Close()
	if err != nil || closeErr != nil || len(raw) > 2<<20 {
		return b, errors.New("invalid or oversized capture bundle")
	}
	if err = decode(raw, &b.Captures); err != nil {
		return b, err
	}
	return b, Validate(b, spec)
}

func Validate(b Bundle, spec provenance.Spec) error {
	fail := func() error { return errors.New("recorded evidence linkage invalid") }
	changes, err := candidates.Changes()
	if err != nil {
		return err
	}
	for _, s := range []Summary{b.Replay, b.Refusal} {
		if s.Execution != "actual_local_kind_via_dedicated_mcp" || s.Proposer != "scripted_preset_sequence_no_model" || !s.ResourcesAbsent || !s.BuildersAbsent || !s.BindingMatches || !s.Trace.Baseline.CleanupVerified || !s.Trace.Baseline.ProvenanceVerified {
			return fail()
		}
		ended, err := time.Parse(time.RFC3339Nano, s.Trace.FinishedAt)
		if err != nil || s.RevalidatedAt.Before(ended) {
			return fail()
		}
		if len(s.Receipts) != len(s.Trace.Attempts) || len(s.Budgets) != len(s.Trace.Attempts) || len(s.Cleanup) != len(s.Trace.Attempts)+1 || len(s.CaptureHashes) != len(s.Trace.References) {
			return fail()
		}
		for _, ref := range s.Trace.References {
			if s.CaptureHashes[ref.ID] != ref.SHA256 {
				return fail()
			}
		}
		runs := []string{s.Trace.Baseline.RunID}
		for _, a := range s.Trace.Attempts {
			r, ok := s.Receipts[a.Proposal.Variant]
			if !ok || r.SourceBinding == nil || r.CandidateRunID != a.Trial.RunID || r.BaselineRunID != s.Trace.Baseline.RunID || r.ExperimentID != a.Trial.ExperimentID || r.ExperimentID != s.Trace.Baseline.ExperimentID || r.Variant != a.Proposal.Variant || r.Verdict != a.Trial.Verdict || !reflect.DeepEqual(r.Spec, spec) {
				return fail()
			}
			c := changes[r.Variant]
			if artifact.Validate(*r.SourceBinding, r.Variant, c.SourceSHA256, c.LogicSHA256, r.SourceBinding.Labels()) != nil {
				return fail()
			}
			budget, ok := s.Budgets[r.Variant]
			if !ok || budget.Status != "passed" || budget.PeakMemoryMeasured || budget.CPULimitNanos != 5000000000 || budget.WorkingSetLimitBytes != 33554432 || budget.CPUDeltaNanos > budget.CPULimitNanos || budget.MaxObservedWorkingSetBytes > budget.WorkingSetLimitBytes {
				return fail()
			}
			runs = append(runs, a.Trial.RunID)
		}
		for _, run := range runs {
			c, ok := s.Cleanup[run]
			if !ok || !c.Complete || len(c.Errors) != 0 {
				return fail()
			}
			names, err := lifecycle.OwnedNames(run)
			if err != nil || len(c.Clusters) != len(names) {
				return fail()
			}
			for i, n := range names {
				if c.Clusters[i].Name != n || c.Clusters[i].Error != "" || len(c.Clusters[i].Remaining) != 0 {
					return fail()
				}
			}
		}
	}
	if b.Replay.Trace.StopReason != "verified_recovery" || b.Replay.Trace.Verdict != "passed" || len(b.Replay.Trace.Attempts) != 3 || len(b.Replay.Trace.References) != 8 || b.Replay.Trace.ToolCalls["compare_trials"] != 1 || b.Refusal.Trace.StopReason != "trial_not_verified" || b.Refusal.Trace.Verdict != "inconclusive" || len(b.Refusal.Trace.Attempts) != 1 {
		return fail()
	}
	for i, want := range []string{"no_improvement", "regression", "verified_recovery"} {
		if b.Replay.Trace.Attempts[i].Outcome != want {
			return fail()
		}
	}
	if !reflect.DeepEqual(b.Captures.Trace, b.Replay.Trace) || len(b.Captures.Captures) != 8 {
		return fail()
	}
	for i, c := range b.Captures.Captures {
		if c.Reference != b.Replay.Trace.References[i] || jsonHash(c.Evidence) != c.Reference.SHA256 || c.Evidence.Trial.TrialID != c.Reference.TrialID {
			return fail()
		}
		if c.Evidence.Receipt != nil && !reflect.DeepEqual(*c.Evidence.Receipt, b.Replay.Receipts[c.Evidence.Receipt.Variant]) {
			return fail()
		}
		if c.Evidence.Baseline != nil && !c.Evidence.Baseline.BaselineReproduced {
			return fail()
		}
		if c.Evidence.Behavior != nil {
			r := c.Evidence.Behavior
			if !r.Completed || r.Requests != 120 || r.Cycles != 20 || len(r.Observations) != 120 || c.Evidence.Budget == nil || !reflect.DeepEqual(*c.Evidence.Budget, b.Replay.Budgets[c.Evidence.Trial.Variant]) {
				return fail()
			}
		}
	}
	return nil
}
