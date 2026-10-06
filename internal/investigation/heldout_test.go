package investigation

import (
	"context"
	"encoding/json"
	"errors"
	"math/big"
	"os"
	"strings"
	"testing"

	"github.com/RajChodisetti/incident-trial-lab/internal/candidates"
	"github.com/RajChodisetti/incident-trial-lab/internal/synthetic"
)

type heldoutSelectorFunc func(context.Context, View, []string) (Decision, error)

func (f heldoutSelectorFunc) Select(ctx context.Context, v View, c []string) (Decision, error) {
	return f(ctx, v, c)
}
func heldoutSeed(t *testing.T) []byte {
	t.Helper()
	b, err := os.ReadFile("../../docs/evidence/selection-context-offline.json")
	if err != nil {
		t.Fatal(err)
	}
	return b
}
func choiceControl(t *testing.T, id string) Selector {
	t.Helper()
	changes, err := candidates.Changes()
	if err != nil {
		t.Fatal(err)
	}
	return heldoutSelectorFunc(func(_ context.Context, v View, _ []string) (Decision, error) {
		for _, c := range v.CandidateChanges {
			if c.LogicSHA256 == changes[id].LogicSHA256 {
				return Decision{Candidate: c.Option}, nil
			}
		}
		return Decision{}, errors.New("control source unavailable")
	})
}

func TestHeldOutDatasetContractAndSplit(t *testing.T) {
	var cases []heldoutCase
	if err := json.Unmarshal(heldoutDataset, &cases); err != nil {
		t.Fatal(err)
	}
	if len(cases) != 18 {
		t.Fatal("dataset bound")
	}
	var visible []struct {
		Request synthetic.Quote `json:"request"`
	}
	seen := map[synthetic.Quote]bool{}
	for _, path := range []string{"../../scenarios/discount-rounding/workload.json", "../../scenarios/discount-rounding/comparison-workload.json"} {
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if err = json.Unmarshal(body, &visible); err != nil {
			t.Fatal(err)
		}
		for _, v := range visible {
			seen[v.Request] = true
		}
	}
	names := map[string]bool{}
	valid, invalid := 0, 0
	for _, c := range cases {
		if names[c.Name] || seen[c.Request] {
			t.Fatalf("overlap/duplicate: %s", c.Name)
		}
		names[c.Name] = true
		seen[c.Request] = true
		q := c.Request
		legal := q.UnitCents >= 1 && q.UnitCents <= 1000000 && q.Quantity >= 1 && q.Quantity <= 100 && q.DiscountPercent >= 0 && q.DiscountPercent <= 100
		if !legal {
			invalid++
			if c.Status != 400 {
				t.Fatal(c.Name)
			}
			continue
		}
		valid++
		// Independent arbitrary-precision contract oracle, not candidate code.
		n := big.NewInt(int64(q.UnitCents))
		n.Mul(n, big.NewInt(int64(q.Quantity)))
		n.Mul(n, big.NewInt(int64(100-q.DiscountPercent)))
		n.Quo(n, big.NewInt(100))
		if c.Status != 200 || n.Int64() != int64(c.Cents) {
			t.Fatalf("wrong expected result: %s", c.Name)
		}
	}
	if valid != 12 || invalid != 6 {
		t.Fatal("split changed")
	}
}

func TestHeldOutControlsAndPermutationBalance(t *testing.T) {
	for _, id := range candidates.IDs() {
		r, err := EvaluateHeldOut(context.Background(), heldoutSeed(t), choiceControl(t, id))
		if err != nil {
			t.Fatal(err)
		}
		want := 0
		if id == "fixed" {
			want = 6
		}
		if r.Selected != 6 || r.AllCasesPassed != want || r.Invalid+r.Abstained+r.SelectorErrors+r.Canceled != 0 {
			t.Fatalf("%s: %+v", id, r)
		}
		positions := map[string][3]int{}
		hashes := map[string]bool{}
		for _, run := range r.Runs {
			if run.Candidate != id {
				t.Fatal(run)
			}
			passed, failed := 12, 6
			if id == "fixed" {
				passed, failed = 18, 0
			}
			if run.Passed != passed || len(run.Failed) != failed {
				t.Fatal(run)
			}
			if hashes[run.RequestSHA256] {
				t.Fatal("duplicate permutation input")
			}
			hashes[run.RequestSHA256] = true
			for pos, id := range run.OwnerOrder {
				counts := positions[id]
				counts[pos]++
				positions[id] = counts
			}
		}
		for _, counts := range positions {
			if counts != [3]int{2, 2, 2} {
				t.Fatal("unbalanced", positions)
			}
		}
	}
	first := heldoutSelectorFunc(func(_ context.Context, _ View, c []string) (Decision, error) { return Decision{Candidate: c[0]}, nil })
	r, err := EvaluateHeldOut(context.Background(), heldoutSeed(t), first)
	if err != nil || r.Selected != 6 || r.AllCasesPassed != 2 {
		t.Fatalf("first-option control: %+v %v", r, err)
	}
}

func TestHeldOutAbstentionInvalidErrorsAndIsolation(t *testing.T) {
	for _, tc := range []struct {
		name     string
		decision Decision
		err      error
		status   string
	}{
		{"abstention", Decision{Abstain: true}, nil, "abstained"},
		{"empty", Decision{}, nil, "invalid"},
		{"unknown", Decision{Candidate: "option-04"}, nil, "invalid"},
		{"semantic", Decision{Candidate: "fixed"}, nil, "invalid"},
		{"ambiguous", Decision{Candidate: "option-01", Abstain: true}, nil, "invalid"},
		{"error", Decision{}, errors.New("private selector error must not be retained"), "selector_error"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := heldoutSelectorFunc(func(_ context.Context, v View, c []string) (Decision, error) {
				body, _ := json.Marshal(SelectionRequest{View: v, Choices: c})
				for _, forbidden := range []string{"failed_cases", "expected_cents", "minimum-fraction", "owner_order", "ineffective", "misleading"} {
					if strings.Contains(string(body), forbidden) {
						t.Fatal("input leakage", forbidden)
					}
				}
				if len(v.History) != 0 || v.BusinessIncident.Request != (synthetic.Quote{UnitCents: 105, Quantity: 3, DiscountPercent: 10}) || v.BusinessIncident.RequiredCents != 283 {
					t.Fatal("visible context drift")
				}
				// Mutation cannot change later calls or the recorded request hash.
				v.BusinessIncident.RequiredCents = 999
				c[0] = "injected"
				v.CandidateChanges[0].Logic = "injected"
				return tc.decision, tc.err
			})
			r, err := EvaluateHeldOut(context.Background(), heldoutSeed(t), s)
			if err != nil {
				t.Fatal(err)
			}
			if r.Selected != 0 || r.AllCasesPassed != 0 || r.Abstained+r.Invalid+r.SelectorErrors != 6 {
				t.Fatal(r)
			}
			for _, run := range r.Runs {
				if run.Status != tc.status || run.Passed != 0 || len(run.Failed) != 0 {
					t.Fatal(run)
				}
			}
		})
	}
}

func TestHeldOutFailClosedAndCancellation(t *testing.T) {
	seed := heldoutSeed(t)
	if _, err := EvaluateHeldOut(context.Background(), seed, nil); err == nil {
		t.Fatal("nil accepted")
	}
	s := choiceControl(t, "fixed")
	if _, err := EvaluateHeldOut(context.Background(), []byte(`{}`), s); err == nil {
		t.Fatal("bad seed accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	s = heldoutSelectorFunc(func(context.Context, View, []string) (Decision, error) {
		cancel()
		return Decision{Candidate: "option-01"}, nil
	})
	r, err := EvaluateHeldOut(ctx, seed, s)
	if !errors.Is(err, context.Canceled) || r.Canceled != 1 || r.Selected != 0 || len(r.Runs) != 1 {
		t.Fatal(r, err)
	}
	r, err = EvaluateHeldOut(ctx, seed, s)
	if !errors.Is(err, context.Canceled) || len(r.Runs) != 0 {
		t.Fatal(r, err)
	}
}
