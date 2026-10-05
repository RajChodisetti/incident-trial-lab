// Package investigation evaluates offline control flow. A scripted proposer is
// not AI capability; proposal text and observed instructions never become tools.
package investigation

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/RajChodisetti/incident-trial-lab/internal/candidates"
	"github.com/RajChodisetti/incident-trial-lab/internal/evaluation"
	"github.com/RajChodisetti/incident-trial-lab/internal/toolserver"
)

type Caller interface {
	Call(context.Context, string, map[string]any, any) error
}
type Proposer interface {
	Propose(context.Context, View) (Proposal, error)
}
type Config struct {
	MaxAttempts  int
	Timeout      time.Duration
	PollInterval time.Duration
}
type Reference struct {
	ID      string `json:"id"`
	TrialID string `json:"trial_id"`
	Section string `json:"section"`
	SHA256  string `json:"sha256"`
}
type Signal struct {
	Recovery    evaluation.Verdict `json:"recovery"`
	Regressions evaluation.Verdict `json:"regressions"`
	Failed      int                `json:"failed"`
	Unknown     int                `json:"unknown"`
}
type View struct {
	ScenarioID        string     `json:"scenario_id"`
	LatestBehavior    Reference  `json:"latest_behavior"`
	LatestProof       Reference  `json:"latest_proof"`
	Signal            Signal     `json:"signal"`
	History           []Feedback `json:"history"`
	RemainingAttempts int        `json:"remaining_attempts"`
}
type Feedback struct {
	Variant  string    `json:"variant"`
	Outcome  string    `json:"outcome"`
	Signal   Signal    `json:"signal"`
	Behavior Reference `json:"behavior"`
	Proof    Reference `json:"proof"`
}

type Proposal struct {
	Variant    string   `json:"variant"`
	Rationale  string   `json:"rationale"`
	References []string `json:"references"`
}
type Attempt struct {
	Number   int                 `json:"number"`
	Proposal Proposal            `json:"proposal"`
	Trial    toolserver.Snapshot `json:"trial"`
	Outcome  string              `json:"outcome"`
	Behavior Reference           `json:"behavior"`
	Proof    Reference           `json:"proof"`
	Signal   Signal              `json:"signal"`
}
type Trace struct {
	Mode         string               `json:"mode"`
	LoopID       string               `json:"loop_id"`
	StartedAt    string               `json:"started_at"`
	FinishedAt   string               `json:"finished_at"`
	MaxAttempts  int                  `json:"max_attempts"`
	Timeout      string               `json:"timeout"`
	Baseline     toolserver.Snapshot  `json:"baseline"`
	Attempts     []Attempt            `json:"attempts"`
	References   []Reference          `json:"references"`
	StopReason   string               `json:"stop_reason"`
	Verdict      evaluation.Verdict   `json:"verdict"`
	Error        string               `json:"error,omitempty"`
	ToolCalls    map[string]int       `json:"tool_calls"`
	Cleanup      *toolserver.Snapshot `json:"cleanup,omitempty"`
	CleanupError string               `json:"cleanup_error,omitempty"`
}
type Capture struct {
	Reference Reference           `json:"reference"`
	Evidence  toolserver.Evidence `json:"evidence"`
}
type Result struct {
	Trace    Trace     `json:"trace"`
	Captures []Capture `json:"captures"`
}

var ids = regexp.MustCompile(`^[0-9a-f]{16}$`)

const maxCalls = 4096

var ErrNoProposal = errors.New("proposer has no further proposal")

// Run requires a dedicated initially empty MCP session. Cleanup is separately
// bounded so an exhausted decision deadline cannot suppress owned cancellation.
func Run(parent context.Context, c Caller, p Proposer, cfg Config) (result Result) {
	started := time.Now().UTC()
	nonce := make([]byte, 8)
	_, nonceErr := rand.Read(nonce)
	t := &result.Trace
	*t = Trace{Mode: "offline_control_flow_no_model", LoopID: hex.EncodeToString(nonce), StartedAt: started.Format(time.RFC3339Nano), MaxAttempts: cfg.MaxAttempts, Timeout: cfg.Timeout.String(), Verdict: evaluation.Inconclusive, Attempts: []Attempt{}, References: []Reference{}, ToolCalls: map[string]int{}}
	active := ""
	known := map[string]bool{}
	catalog := toolserver.Catalog{}
	refs := map[string]Reference{}
	calls := 0
	ctx, cancel := context.WithTimeout(parent, cfg.Timeout)
	defer cancel()
	call := func(callCtx context.Context, name string, args map[string]any, out any) error {
		if err := callCtx.Err(); err != nil {
			return err
		}
		if calls >= maxCalls {
			return fmt.Errorf("tool call budget exhausted")
		}
		calls++
		t.ToolCalls[name]++
		requestCtx, stop := context.WithTimeout(callCtx, 5*time.Second)
		defer stop()
		return c.Call(requestCtx, name, args, out)
	}
	// Dedicated-session recovery only: after an ambiguous submission, identify a
	// newly admitted job through the scoped listing, never a guessed PID/path.
	ambiguous := false
	defer func() {
		defer func() { t.FinishedAt = time.Now().UTC().Format(time.RFC3339Nano) }()
		if ambiguous || active != "" {
			cleanupCtx, stop := context.WithTimeout(context.Background(), 90*time.Second)
			defer stop()
			if ambiguous {
				var rows toolserver.Trials
				t.ToolCalls["list_trials"]++
				if err := c.Call(cleanupCtx, "list_trials", map[string]any{}, &rows); err != nil {
					t.CleanupError = err.Error()
				} else {
					for _, row := range rows.Trials {
						if !known[row.TrialID] && ids.MatchString(row.TrialID) {
							active = row.TrialID
							break
						}
					}
				}
			}
			if active != "" {
				var answer toolserver.Canceled
				t.ToolCalls["cancel_trial"]++
				if err := c.Call(cleanupCtx, "cancel_trial", map[string]any{"trial_id": active}, &answer); err != nil {
					t.CleanupError = err.Error()
				} else {
					tick := time.NewTicker(100 * time.Millisecond)
					defer tick.Stop()
					for {
						var s toolserver.Snapshot
						t.ToolCalls["get_trial"]++
						err := c.Call(cleanupCtx, "get_trial", map[string]any{"trial_id": active}, &s)
						if err != nil {
							t.CleanupError = err.Error()
							break
						}
						if terminal(s) {
							t.Cleanup = &s
							if !s.CleanupVerified {
								t.CleanupError = "owned cleanup remains unverified"
							}
							break
						}
						select {
						case <-cleanupCtx.Done():
							t.CleanupError = cleanupCtx.Err().Error()
							return
						case <-tick.C:
						}
					}
				}
			}
		}
	}()
	stop := func(reason string, err error) {
		t.StopReason = reason
		if err != nil {
			t.Error = err.Error()
		}
	}
	if c == nil || p == nil || nonceErr != nil || cfg.MaxAttempts < 1 || cfg.MaxAttempts > 3 || cfg.Timeout <= 0 || cfg.Timeout > 10*time.Minute || cfg.PollInterval <= 0 {
		stop("invalid_config", nonceErr)
		return
	}
	wait := func(id string) (toolserver.Snapshot, error) {
		var s toolserver.Snapshot
		tick := time.NewTicker(cfg.PollInterval)
		defer tick.Stop()
		for {
			if err := call(ctx, "get_trial", map[string]any{"trial_id": id}, &s); err != nil {
				return s, err
			}
			if s.TrialID != id {
				return s, fmt.Errorf("status identity mismatch")
			}
			if terminal(s) {
				return s, nil
			}
			select {
			case <-ctx.Done():
				return s, ctx.Err()
			case <-tick.C:
			}
		}
	}
	fail := func(err error) {
		reason := "tool_error"
		if ctx.Err() != nil {
			reason = "deadline_exhausted"
			if errors.Is(parent.Err(), context.Canceled) {
				reason = "canceled"
			}
		}
		stop(reason, err)
	}
	capture := func(id, section string) (toolserver.Evidence, Reference, error) {
		var e toolserver.Evidence
		if err := call(ctx, "get_evidence", map[string]any{"trial_id": id, "section": section}, &e); err != nil {
			return e, Reference{}, err
		}
		if e.Trial.TrialID != id || !e.UntrustedEvidence {
			return e, Reference{}, fmt.Errorf("evidence identity/trust marker mismatch")
		}
		b, err := json.Marshal(e)
		if err != nil {
			return e, Reference{}, err
		}
		h := sha256.Sum256(b)
		ref := Reference{fmt.Sprintf("%s:r%03d", t.LoopID, len(t.References)+1), id, section, hex.EncodeToString(h[:])}
		refs[ref.ID] = ref
		t.References = append(t.References, ref)
		result.Captures = append(result.Captures, Capture{ref, e})
		return e, ref, nil
	}
	var before toolserver.Trials
	if err := call(ctx, "list_trials", map[string]any{}, &before); err != nil {
		fail(err)
		return
	}
	if len(before.Trials) != 0 {
		stop("session_not_empty", nil)
		return
	}
	if err := call(ctx, "list_scenarios", map[string]any{}, &catalog); err != nil {
		fail(err)
		return
	}
	if catalog.ScenarioID != toolserver.Scenario || catalog.SessionJobLimit < cfg.MaxAttempts+1 {
		stop("unsupported_catalog", nil)
		return
	}
	var base toolserver.Snapshot
	ambiguous = true
	if err := call(ctx, "create_experiment", map[string]any{"scenario_id": toolserver.Scenario}, &base); err != nil {
		fail(err)
		return
	}
	ambiguous = false
	if !ids.MatchString(base.TrialID) {
		stop("invalid_submission_identity", nil)
		ambiguous = true
		return
	}
	active = base.TrialID
	known[active] = true
	t.Baseline = base
	base, err := wait(active)
	if err != nil {
		fail(err)
		return
	}
	active = ""
	t.Baseline = base
	if !eligible(base) || base.Kind != "baseline" || !ids.MatchString(base.ExperimentID) {
		stop("baseline_not_verified", nil)
		return
	}
	behavior, behaviorRef, err := capture(base.TrialID, "behavior")
	if err != nil {
		fail(err)
		return
	}
	proof, proofRef, err := capture(base.TrialID, "provenance")
	if err != nil {
		fail(err)
		return
	}
	if !eligible(behavior.Trial) || !eligible(proof.Trial) || behavior.Trial.ExperimentID != base.ExperimentID || proof.Trial.ExperimentID != base.ExperimentID || behavior.Baseline == nil || !behavior.Baseline.BaselineReproduced || proof.Experiment == nil || proof.Experiment.ID != base.ExperimentID || !same(proof.Experiment.Spec, catalog.Spec) {
		stop("inconsistent_baseline_evidence", nil)
		return
	}
	view := View{ScenarioID: toolserver.Scenario, LatestBehavior: behaviorRef, LatestProof: proofRef, Signal: baselineSignal(behavior), History: []Feedback{}, RemainingAttempts: cfg.MaxAttempts}
	seen := map[string]bool{}
	for number := 1; number <= cfg.MaxAttempts; number++ {
		if ctx.Err() != nil {
			fail(ctx.Err())
			return
		}
		proposal, err := propose(ctx, p, view)
		if err != nil {
			if ctx.Err() != nil {
				fail(err)
			} else if errors.Is(err, ErrNoProposal) {
				stop("proposer_exhausted", nil)
			} else if errors.Is(err, ErrAbstained) {
				stop("selector_abstained", nil)
			} else {
				stop("proposer_error", err)
			}
			return
		}
		if err = validateProposal(proposal, view, refs); err != nil {
			stop("invalid_proposal", err)
			return
		}
		if seen[proposal.Variant] {
			stop("no_new_proposal", nil)
			return
		}
		seen[proposal.Variant] = true
		attempt := Attempt{Number: number, Proposal: proposal, Outcome: "not_completed"}
		var submitted toolserver.Snapshot
		ambiguous = true
		if err = call(ctx, "start_trial", map[string]any{"experiment_id": base.ExperimentID, "variant": proposal.Variant}, &submitted); err != nil {
			attempt.Outcome = "submission_error"
			t.Attempts = append(t.Attempts, attempt)
			fail(err)
			return
		}
		ambiguous = false
		if !ids.MatchString(submitted.TrialID) {
			ambiguous = true
			stop("invalid_submission_identity", nil)
			return
		}
		active = submitted.TrialID
		known[active] = true
		attempt.Trial = submitted
		var observed toolserver.Snapshot
		observed, err = wait(active)
		if observed.TrialID != "" {
			attempt.Trial = observed
		}
		if err != nil {
			t.Attempts = append(t.Attempts, attempt)
			fail(err)
			return
		}
		active = ""
		data, ref, err := capture(attempt.Trial.TrialID, "behavior")
		if err != nil {
			t.Attempts = append(t.Attempts, attempt)
			fail(err)
			return
		}
		attempt.Behavior = ref
		attempt.Signal = candidateSignal(data)
		provenance, ref, err := capture(attempt.Trial.TrialID, "provenance")
		if err != nil {
			t.Attempts = append(t.Attempts, attempt)
			fail(err)
			return
		}
		attempt.Proof = ref
		s := attempt.Trial
		r := provenance.Receipt
		if !eligible(data.Trial) || !eligible(provenance.Trial) || !eligible(s) || s.Kind != "candidate" || s.Variant != proposal.Variant || s.ExperimentID != base.ExperimentID || r == nil || r.ExperimentID != base.ExperimentID || r.CandidateRunID != s.RunID || r.Variant != proposal.Variant || r.Verdict != s.Verdict || !same(r.Spec, catalog.Spec) {
			attempt.Outcome = "unverified"
			t.Attempts = append(t.Attempts, attempt)
			stop("trial_not_verified", nil)
			return
		}
		if data.Behavior == nil || !data.Behavior.Completed || data.Behavior.Requests != 120 || data.Behavior.Cycles != 20 || data.Budget == nil {
			attempt.Outcome = "inconclusive"
		} else if s.Verdict == evaluation.Passed && data.Behavior.Recovery == evaluation.Passed && data.Behavior.Regressions == evaluation.Passed && data.Behavior.Warmup == evaluation.Passed && data.Budget.Status == evaluation.Passed {
			attempt.Outcome = "verified_recovery"
		} else if s.Verdict == evaluation.Inconclusive {
			attempt.Outcome = "inconclusive"
		} else if data.Behavior.Recovery == evaluation.Failed {
			attempt.Outcome = "no_improvement"
		} else if data.Behavior.Regressions == evaluation.Failed {
			attempt.Outcome = "regression"
		} else {
			attempt.Outcome = "budget_or_behavior_failure"
		}
		t.Attempts = append(t.Attempts, attempt)
		if attempt.Outcome == "verified_recovery" {
			var comparison toolserver.Comparison
			if err = call(ctx, "compare_trials", map[string]any{"experiment_id": base.ExperimentID}, &comparison); err != nil {
				fail(err)
				return
			}
			confirmed := false
			for _, row := range comparison.Candidates {
				if row.TrialID == s.TrialID && eligible(row) && row.Verdict == evaluation.Passed {
					confirmed = true
				}
			}
			if !confirmed {
				stop("final_proof_not_current", nil)
				return
			}
			t.Verdict = evaluation.Passed
			stop("verified_recovery", nil)
			return
		}
		if attempt.Outcome == "inconclusive" {
			stop("inconclusive", nil)
			return
		}
		view = View{ScenarioID: toolserver.Scenario, LatestBehavior: attempt.Behavior, LatestProof: attempt.Proof, Signal: attempt.Signal, History: feedback(t.Attempts), RemainingAttempts: cfg.MaxAttempts - number}
	}
	stop("attempt_budget_exhausted", nil)
	return
}
func terminal(s toolserver.Snapshot) bool { return s.State != "queued" && s.State != "running" }
func eligible(s toolserver.Snapshot) bool {
	return s.State == "completed" && s.CleanupVerified && s.ProvenanceVerified && s.EligibleNow
}
func same(a, b any) bool {
	x, _ := json.Marshal(a)
	y, _ := json.Marshal(b)
	return string(x) == string(y)
}

// ValidateProposal validates a proposal against the current view's two references.
// Structural validity does not prove that a rationale is correct or a fix works.
func ValidateProposal(p Proposal, v View) error {
	return validateProposal(p, v, map[string]Reference{v.LatestBehavior.ID: v.LatestBehavior, v.LatestProof.ID: v.LatestProof})
}

func validateProposal(p Proposal, v View, refs map[string]Reference) error {
	if err := validateLatestReferences(v); err != nil {
		return err
	}
	if !candidates.ValidID(p.Variant) {
		return fmt.Errorf("unsupported candidate; proposal text cannot grant commands")
	}
	if len(p.Rationale) == 0 || len(p.Rationale) > 512 || len(p.References) < 2 || len(p.References) > 8 {
		return fmt.Errorf("bounded rationale and evidence references required")
	}
	behavior, proof := false, false
	seen := map[string]bool{}
	for _, id := range p.References {
		if _, ok := refs[id]; !ok || seen[id] {
			return fmt.Errorf("unknown or duplicate evidence reference")
		}
		seen[id] = true
		behavior = behavior || id == v.LatestBehavior.ID
		proof = proof || id == v.LatestProof.ID
	}
	if !behavior || !proof {
		return fmt.Errorf("proposal must cite latest behavior and provenance")
	}
	return nil
}
func baselineSignal(e toolserver.Evidence) Signal {
	s := Signal{Recovery: evaluation.Failed, Regressions: evaluation.Passed}
	if e.Baseline != nil {
		for _, o := range e.Baseline.Observations {
			if o.BusinessCheck == "fail" {
				s.Failed++
			}
			if o.Error != "" {
				s.Unknown++
			}
		}
	}
	return s
}
func candidateSignal(e toolserver.Evidence) Signal {
	s := Signal{Recovery: evaluation.Inconclusive, Regressions: evaluation.Inconclusive}
	if e.Behavior != nil {
		s.Recovery = e.Behavior.Recovery
		s.Regressions = e.Behavior.Regressions
		for _, o := range e.Behavior.Observations {
			if o.Status == "fail" {
				s.Failed++
			}
			if o.Status == "unknown" {
				s.Unknown++
			}
		}
	}
	return s
}

// Scripted intentionally selects a fixed sequence. It checks control flow, not
// diagnosis, reasoning quality, novel patch generation or model capability.
type Scripted struct {
	Variants []string
	next     int
}

func (s *Scripted) Propose(ctx context.Context, v View) (Proposal, error) {
	if ctx.Err() != nil {
		return Proposal{}, ctx.Err()
	}
	if s.next >= len(s.Variants) {
		return Proposal{}, ErrNoProposal
	}
	variant := s.Variants[s.next]
	s.next++
	return Proposal{variant, "Offline preset proposal for deterministic control-flow evaluation", []string{v.LatestBehavior.ID, v.LatestProof.ID}}, nil
}

func feedback(attempts []Attempt) []Feedback {
	out := make([]Feedback, 0, len(attempts))
	for _, a := range attempts {
		out = append(out, Feedback{a.Proposal.Variant, a.Outcome, a.Signal, a.Behavior, a.Proof})
	}
	return out
}

// A proposer has no Caller capability. A noncooperative proposer cannot extend
// the loop deadline; its single abandoned goroutine is confined to this process.
func propose(ctx context.Context, p Proposer, v View) (Proposal, error) {
	type response struct {
		p   Proposal
		err error
	}
	done := make(chan response, 1)
	go func() { proposal, err := p.Propose(ctx, v); done <- response{proposal, err} }()
	select {
	case <-ctx.Done():
		return Proposal{}, ctx.Err()
	case r := <-done:
		return r.p, r.err
	}
}
