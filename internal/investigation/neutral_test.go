package investigation

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/RajChodisetti/incident-trial-lab/internal/candidates"
	"github.com/RajChodisetti/incident-trial-lab/internal/evaluation"
)

func TestNeutralChoicesCoverEveryPermutationWithoutNameLeakage(t *testing.T) {
	orders := [][]string{
		{"fixed", "ineffective", "misleading"}, {"fixed", "misleading", "ineffective"},
		{"ineffective", "fixed", "misleading"}, {"ineffective", "misleading", "fixed"},
		{"misleading", "fixed", "ineffective"}, {"misleading", "ineffective", "fixed"},
	}
	for _, order := range orders {
		for i, want := range order {
			alias := []string{"option-01", "option-02", "option-03"}[i]
			inner := selectorFunc(func(_ context.Context, v View, choices []string) (Decision, error) {
				if !reflect.DeepEqual(choices, []string{"option-01", "option-02", "option-03"}) {
					t.Fatal("choice ordering leaked semantic order")
				}
				data, _ := json.Marshal(struct {
					View    View
					Choices []string
				}{v, choices})
				for _, id := range candidates.IDs() {
					if strings.Contains(string(data), id) {
						t.Fatalf("preset label leaked: %s", id)
					}
				}
				return Decision{Candidate: alias}, nil
			})
			d, e := neutralWithOrder(inner, order).Select(context.Background(), selectorView(), candidates.IDs())
			if e != nil || d.Candidate != want {
				t.Fatalf("mapping=%v want=%s decision=%+v error=%v", order, want, d, e)
			}
		}
	}
}

func TestNeutralRevisionKeepsStableHistoryAndFiltersOptions(t *testing.T) {
	call := 0
	inner := selectorFunc(func(_ context.Context, v View, choices []string) (Decision, error) {
		call++
		if call == 1 {
			return Decision{Candidate: "option-03"}, nil
		} // ineffective
		if call == 2 {
			if v.History[0].Variant != "option-03" || v.History[0].Outcome != "no_improvement" || !reflect.DeepEqual(choices, []string{"option-01", "option-02"}) {
				t.Fatal("history alias changed or tried choice returned")
			}
			v.History[0].Variant = "untrusted mutation"
			return Decision{Candidate: "option-01"}, nil // misleading
		}
		if v.History[0].Variant != "option-03" || v.History[1].Variant != "option-01" || v.History[1].Outcome != "regression" || !reflect.DeepEqual(choices, []string{"option-02"}) {
			t.Fatal("revision linkage lost")
		}
		return Decision{Candidate: "option-02"}, nil // fixed
	})
	p := SelectorProposer{Selector: neutralWithOrder(inner, []string{"misleading", "fixed", "ineffective"})}
	f := &fakeTools{}
	r := Run(context.Background(), f, p, config(3))
	if r.Trace.Verdict != evaluation.Passed || !reflect.DeepEqual(f.variants, []string{"ineffective", "misleading", "fixed"}) {
		t.Fatalf("trace %+v", r.Trace)
	}
	for i, want := range []string{"no_improvement", "regression", "verified_recovery"} {
		if r.Trace.Attempts[i].Outcome != want {
			t.Fatal("verdict changed by aliases")
		}
	}
}

func TestNeutralInvalidAndAbstainingChoicesStopBeforeCandidateTrial(t *testing.T) {
	for _, tc := range []struct {
		name     string
		decision Decision
		reason   string
	}{
		{"semantic ID", Decision{Candidate: "fixed"}, "proposer_error"},
		{"unknown alias", Decision{Candidate: "option-99"}, "proposer_error"},
		{"empty", Decision{}, "proposer_error"},
		{"abstain", Decision{Abstain: true}, "selector_abstained"},
		{"ambiguous", Decision{Candidate: "option-02", Abstain: true}, "proposer_error"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			inner := selectorFunc(func(context.Context, View, []string) (Decision, error) { return tc.decision, nil })
			p := SelectorProposer{Selector: neutralWithOrder(inner, []string{"misleading", "fixed", "ineffective"})}
			f := &fakeTools{}
			r := Run(context.Background(), f, p, config(1))
			if r.Trace.StopReason != tc.reason || r.Trace.Verdict != evaluation.Inconclusive || len(f.variants) != 0 || len(r.Captures) != 2 {
				t.Fatalf("trace %+v", r.Trace)
			}
		})
	}
	inner := selectorFunc(func(context.Context, View, []string) (Decision, error) { return Decision{Candidate: "option-03"}, nil })
	n := neutralWithOrder(inner, []string{"misleading", "fixed", "ineffective"})
	if _, e := n.Select(context.Background(), selectorView(), []string{"fixed"}); e == nil {
		t.Fatal("unavailable alias accepted")
	}
}

func TestNeutralConstructionAndFailures(t *testing.T) {
	if _, e := NewNeutralSelector(nil); e == nil {
		t.Fatal("nil selector accepted")
	}
	inner := selectorFunc(func(context.Context, View, []string) (Decision, error) { return Decision{Candidate: "option-01"}, nil })
	n, e := NewNeutralSelector(inner)
	if e != nil {
		t.Fatal(e)
	}
	for _, id := range candidates.IDs() {
		if n.aliasToID[n.idToAlias[id]] != id {
			t.Fatal("random mapping not bijective")
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, e = n.Select(ctx, selectorView(), candidates.IDs()); !errors.Is(e, context.Canceled) {
		t.Fatal("canceled selection invoked")
	}
	ctx, cancel = context.WithCancel(context.Background())
	n.inner = selectorFunc(func(context.Context, View, []string) (Decision, error) {
		cancel()
		return Decision{Candidate: "option-01"}, nil
	})
	if _, e = n.Select(ctx, selectorView(), candidates.IDs()); !errors.Is(e, context.Canceled) {
		t.Fatal("late canceled answer accepted")
	}
	n.inner = selectorFunc(func(context.Context, View, []string) (Decision, error) {
		return Decision{}, errors.New("untrusted error body")
	})
	if _, e = n.Select(context.Background(), selectorView(), candidates.IDs()); !errors.Is(e, ErrSelection) || strings.Contains(e.Error(), "untrusted") {
		t.Fatal("raw error propagated")
	}
	if _, e = n.Select(context.Background(), selectorView(), nil); !errors.Is(e, ErrNoProposal) {
		t.Fatal("empty choices accepted")
	}
	if _, e = n.Select(context.Background(), selectorView(), []string{"fixed", "fixed"}); e == nil {
		t.Fatal("duplicate choices accepted")
	}
	v := selectorView()
	v.History = []Feedback{{Variant: "unknown"}}
	if _, e = n.Select(context.Background(), v, candidates.IDs()); e == nil {
		t.Fatal("unknown history accepted")
	}
}
