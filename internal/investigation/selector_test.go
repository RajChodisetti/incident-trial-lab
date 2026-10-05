package investigation

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/RajChodisetti/incident-trial-lab/internal/candidates"
	"github.com/RajChodisetti/incident-trial-lab/internal/evaluation"
	"github.com/RajChodisetti/incident-trial-lab/internal/toolserver"
)

type selectorFunc func(context.Context, View, []string) (Decision, error)

func (f selectorFunc) Select(ctx context.Context, v View, choices []string) (Decision, error) {
	return f(ctx, v, choices)
}

func selectorView() View {
	return View{ScenarioID: toolserver.Scenario, RemainingAttempts: 3, LatestBehavior: Reference{ID: "aaaaaaaaaaaaaaaa:r001", TrialID: "1111111111111111", Section: "behavior", SHA256: strings.Repeat("a", 64)}, LatestProof: Reference{ID: "aaaaaaaaaaaaaaaa:r002", TrialID: "1111111111111111", Section: "provenance", SHA256: strings.Repeat("b", 64)}}
}

func TestSelectorProposerChoicesAndInputIsolation(t *testing.T) {
	for _, id := range candidates.IDs() {
		t.Run(id, func(t *testing.T) {
			v := selectorView()
			p := SelectorProposer{Selector: selectorFunc(func(_ context.Context, in View, choices []string) (Decision, error) {
				if !reflect.DeepEqual(choices, candidates.IDs()) {
					t.Fatal("wrong finite choices")
				}
				choices[0] = "arbitrary command"
				in.LatestBehavior.ID = "untrusted override"
				return Decision{Candidate: id}, nil
			})}
			proposal, e := p.Propose(context.Background(), v)
			if e != nil || proposal.Variant != id || ValidateProposal(proposal, v) != nil || !strings.Contains(proposal.Rationale, "Deterministic audit text") {
				t.Fatalf("proposal=%+v error=%v", proposal, e)
			}
		})
	}
	v := selectorView()
	v.History = []Feedback{{Variant: "ineffective"}}
	p := SelectorProposer{Selector: selectorFunc(func(_ context.Context, in View, choices []string) (Decision, error) {
		if !reflect.DeepEqual(choices, []string{"fixed", "misleading"}) {
			t.Fatal("previous attempt still offered")
		}
		in.History[0].Variant = "fixed"
		return Decision{Candidate: "fixed"}, nil
	})}
	if _, e := p.Propose(context.Background(), v); e != nil || v.History[0].Variant != "ineffective" {
		t.Fatal("selector rewrote owned history")
	}
}

func TestSelectorRejectsInvalidDecisions(t *testing.T) {
	for _, d := range []Decision{{Candidate: "unknown"}, {Candidate: "sh -c anything"}, {}, {Candidate: "fixed", Abstain: true}} {
		p := SelectorProposer{Selector: selectorFunc(func(context.Context, View, []string) (Decision, error) { return d, nil })}
		f := &fakeTools{}
		r := Run(context.Background(), f, p, config(1))
		if r.Trace.StopReason != "proposer_error" || r.Trace.Verdict != evaluation.Inconclusive || len(f.variants) != 0 {
			t.Fatalf("decision=%+v trace=%+v", d, r.Trace)
		}
	}
	v := selectorView()
	v.History = []Feedback{{Variant: "ineffective"}}
	p := SelectorProposer{Selector: selectorFunc(func(context.Context, View, []string) (Decision, error) {
		return Decision{Candidate: "ineffective"}, nil
	})}
	if _, e := p.Propose(context.Background(), v); e == nil {
		t.Fatal("repeated candidate accepted")
	}
}

func TestSelectorAbstentionPreservesEarlierEvidence(t *testing.T) {
	for _, afterAttempt := range []bool{false, true} {
		t.Run(map[bool]string{false: "baseline", true: "after failed trial"}[afterAttempt], func(t *testing.T) {
			calls := 0
			p := SelectorProposer{Selector: selectorFunc(func(_ context.Context, v View, choices []string) (Decision, error) {
				calls++
				if afterAttempt && calls == 1 {
					return Decision{Candidate: "ineffective"}, nil
				}
				if afterAttempt && (v.Signal.Recovery != evaluation.Failed || len(v.History) != 1 || v.History[0].Outcome != "no_improvement") {
					t.Fatal("failed evidence was lost")
				}
				return Decision{Abstain: true}, nil
			})}
			f := &fakeTools{}
			r := Run(context.Background(), f, p, config(3))
			want := 0
			if afterAttempt {
				want = 1
			}
			if r.Trace.StopReason != "selector_abstained" || r.Trace.Verdict != evaluation.Inconclusive || len(r.Trace.Attempts) != want || len(f.variants) != want || len(r.Captures) != 2+2*want {
				t.Fatalf("trace %+v", r.Trace)
			}
			if afterAttempt && r.Trace.Attempts[0].Outcome != "no_improvement" {
				t.Fatal("abstention erased failed trial")
			}
		})
	}
}

func TestSelectorPreconditionsAndCancellation(t *testing.T) {
	calls := 0
	p := SelectorProposer{Selector: selectorFunc(func(context.Context, View, []string) (Decision, error) {
		calls++
		return Decision{Candidate: "fixed"}, nil
	})}
	for _, mutate := range []func(*View){
		func(v *View) { v.LatestBehavior.SHA256 = "bad" },
		func(v *View) { v.LatestProof.TrialID = "2222222222222222" },
		func(v *View) { v.LatestProof.ID = "bbbbbbbbbbbbbbbb:r002" },
		func(v *View) { v.LatestProof.Section = "behavior" },
		func(v *View) { v.History = []Feedback{{Variant: "unknown"}} },
		func(v *View) { v.History = []Feedback{{Variant: "fixed"}, {Variant: "fixed"}} },
		func(v *View) { v.RemainingAttempts = 4 },
	} {
		v := selectorView()
		mutate(&v)
		if _, e := p.Propose(context.Background(), v); e == nil {
			t.Fatal("invalid view accepted")
		}
	}
	for _, exhaustHistory := range []bool{false, true} {
		v := selectorView()
		v.RemainingAttempts = 0
		if exhaustHistory {
			v.RemainingAttempts = 1
			for _, id := range candidates.IDs() {
				v.History = append(v.History, Feedback{Variant: id})
			}
		}
		if _, e := p.Propose(context.Background(), v); !errors.Is(e, ErrNoProposal) {
			t.Fatal("exhausted choices did not stop")
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, e := p.Propose(ctx, selectorView()); !errors.Is(e, context.Canceled) {
		t.Fatal("cancellation ignored")
	}
	if calls != 0 {
		t.Fatal("invalid view reached selector")
	}
	ctx, cancel = context.WithCancel(context.Background())
	p.Selector = selectorFunc(func(context.Context, View, []string) (Decision, error) {
		cancel()
		return Decision{Candidate: "fixed"}, nil
	})
	if _, e := p.Propose(ctx, selectorView()); !errors.Is(e, context.Canceled) {
		t.Fatal("late canceled answer accepted")
	}
	p.Selector = selectorFunc(func(context.Context, View, []string) (Decision, error) {
		return Decision{}, errors.New("untrusted provider error")
	})
	if _, e := p.Propose(context.Background(), selectorView()); !errors.Is(e, ErrSelection) || strings.Contains(e.Error(), "untrusted") {
		t.Fatal("provider error leaked or fallback enabled")
	}
	if _, e := (SelectorProposer{}).Propose(context.Background(), selectorView()); !errors.Is(e, ErrSelection) {
		t.Fatal("nil selector accepted")
	}
}

type callerFunc func(context.Context, string, map[string]any, any) error

func (f callerFunc) Call(ctx context.Context, name string, args map[string]any, out any) error {
	return f(ctx, name, args, out)
}

func TestSelectorCannotOverrideIndependentTrialVerdicts(t *testing.T) {
	for _, which := range []string{"passed", "recovery", "regression", "budget", "unknown", "stale", "comparison"} {
		t.Run(which, func(t *testing.T) {
			f := &fakeTools{unknown: which == "unknown", stale: which == "stale"}
			c := callerFunc(func(ctx context.Context, name string, args map[string]any, out any) error {
				if e := f.Call(ctx, name, args, out); e != nil {
					return e
				}
				if e, ok := out.(*toolserver.Evidence); ok && e.Behavior != nil {
					switch which {
					case "recovery":
						e.Behavior.Recovery = evaluation.Failed
					case "regression":
						e.Behavior.Regressions = evaluation.Failed
					case "budget":
						e.Budget.Status = evaluation.Failed
					}
				}
				if comparison, ok := out.(*toolserver.Comparison); ok && which == "comparison" {
					comparison.Candidates[0].Verdict = evaluation.Failed
				}
				return nil
			})
			p := SelectorProposer{Selector: selectorFunc(func(context.Context, View, []string) (Decision, error) { return Decision{Candidate: "fixed"}, nil })}
			r := Run(context.Background(), c, p, config(1))
			want := map[string]string{"passed": "verified_recovery", "recovery": "no_improvement", "regression": "regression", "budget": "budget_or_behavior_failure", "unknown": "inconclusive", "stale": "unverified", "comparison": "verified_recovery"}[which]
			if len(r.Trace.Attempts) != 1 || r.Trace.Attempts[0].Outcome != want || (r.Trace.Verdict == evaluation.Passed) != (which == "passed") {
				t.Fatalf("trace %+v", r.Trace)
			}
		})
	}
}

func TestSelectorRevisionSequence(t *testing.T) {
	sequence := []string{"ineffective", "misleading", "fixed"}
	next := 0
	p := SelectorProposer{Selector: selectorFunc(func(_ context.Context, v View, choices []string) (Decision, error) {
		id := sequence[next]
		next++
		for _, h := range v.History {
			for _, choice := range choices {
				if h.Variant == choice {
					t.Fatal("tried choice reoffered")
				}
			}
		}
		return Decision{Candidate: id}, nil
	})}
	r := Run(context.Background(), &fakeTools{}, p, config(3))
	if r.Trace.Verdict != evaluation.Passed || len(r.Trace.Attempts) != 3 {
		t.Fatalf("trace %+v", r.Trace)
	}
	for i, want := range []string{"no_improvement", "regression", "verified_recovery"} {
		if r.Trace.Attempts[i].Outcome != want {
			t.Fatal("wrong revision outcome")
		}
	}
}

func TestSharedProposalValidationRejectsDuplicateCitations(t *testing.T) {
	p := proposerFunc(func(_ context.Context, v View) (Proposal, error) {
		return Proposal{Variant: "fixed", Rationale: "Offline audit", References: []string{v.LatestBehavior.ID, v.LatestProof.ID, v.LatestProof.ID}}, nil
	})
	f := &fakeTools{}
	r := Run(context.Background(), f, p, config(1))
	if r.Trace.StopReason != "invalid_proposal" || r.Trace.Verdict != evaluation.Inconclusive || len(f.variants) != 0 {
		t.Fatalf("duplicate citations reached a trial: %+v", r.Trace)
	}
}
