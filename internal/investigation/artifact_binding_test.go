package investigation

import (
	"strings"
	"testing"

	"github.com/RajChodisetti/incident-trial-lab/internal/artifact"
	"github.com/RajChodisetti/incident-trial-lab/internal/evaluation"
	"github.com/RajChodisetti/incident-trial-lab/internal/provenance"
)

func TestSelectionSourceRequiresMatchingBoundReceipt(t *testing.T) {
	r := makeRecord(t, Decision{Candidate: "option-02"}, nil)
	id := r.OwnerAliases[r.Decision.Candidate]
	b, err := artifact.ForBinary(id, strings.NewReader("offline fixture binary"))
	if err != nil {
		t.Fatal(err)
	}
	receipt := provenance.Receipt{Variant: id, SourceBinding: &b, Spec: provenance.Spec{Candidates: map[string]string{id: "sha256:" + strings.Repeat("a", 64)}}, Verdict: evaluation.Failed}
	if MatchSelectionSource(r, receipt) != nil {
		t.Fatal("matching source rejected; a failed verdict does not negate source linkage")
	}
	for _, mutate := range []func(*provenance.Receipt){
		func(r *provenance.Receipt) { r.SourceBinding = nil },
		func(r *provenance.Receipt) { r.Variant = "unknown" },
		func(r *provenance.Receipt) { r.SourceBinding.SourceSHA256 = strings.Repeat("0", 64) },
		func(r *provenance.Receipt) { r.SourceBinding.LogicSHA256 = strings.Repeat("0", 64) },
		func(r *provenance.Receipt) { r.Spec.Candidates = map[string]string{} },
	} {
		copy := receipt
		binding := b
		copy.SourceBinding = &binding
		mutate(&copy)
		if MatchSelectionSource(r, copy) == nil {
			t.Fatal("missing/mismatched source linkage accepted")
		}
	}
	if MatchSelectionSource(makeRecord(t, Decision{Abstain: true}, nil), receipt) == nil {
		t.Fatal("abstention became selected artifact")
	}
}

func TestHistoricalRecordCannotClaimBoundRecovery(t *testing.T) {
	r := makeRecord(t, Decision{Candidate: "option-02"}, nil)
	legacy := provenance.Receipt{Variant: r.OwnerAliases[r.Decision.Candidate], Verdict: evaluation.Passed}
	if MatchSelectionSource(r, legacy) == nil {
		t.Fatal("legacy receipt without source proof accepted")
	}
}
