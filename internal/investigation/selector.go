package investigation

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/RajChodisetti/incident-trial-lab/internal/candidates"
)

// Decision is a local domain type, not an assumed provider response contract.
// Abstention is explicit; an empty or unknown candidate is not a fallback.
type Decision struct {
	Candidate string `json:"candidate"`
	Abstain   bool   `json:"abstain"`
}

// Selector receives sanitized evidence and the untried finite choices only.
// It receives no Caller or capability to evaluate, start trials or apply changes.
type Selector interface {
	Select(context.Context, View, []string) (Decision, error)
}

type SelectorProposer struct{ Selector Selector }

var _ Proposer = SelectorProposer{}
var ErrAbstained = errors.New("selector abstained")
var ErrSelection = errors.New("selector failed; no fallback")

var referenceID = regexp.MustCompile(`^[0-9a-f]{16}:r[0-9]{3}$`)
var referenceHash = regexp.MustCompile(`^[0-9a-f]{64}$`)

func validateLatestReferences(v View) error {
	for i, ref := range []Reference{v.LatestBehavior, v.LatestProof} {
		section := "behavior"
		if i == 1 {
			section = "provenance"
		}
		if !referenceID.MatchString(ref.ID) || !ids.MatchString(ref.TrialID) || !referenceHash.MatchString(ref.SHA256) || ref.Section != section {
			return errors.New("malformed latest evidence reference")
		}
	}
	if v.LatestBehavior.ID == v.LatestProof.ID || v.LatestBehavior.TrialID != v.LatestProof.TrialID || strings.Split(v.LatestBehavior.ID, ":")[0] != strings.Split(v.LatestProof.ID, ":")[0] {
		return errors.New("latest evidence references do not share a loop and trial")
	}
	return nil
}

func (p SelectorProposer) Propose(ctx context.Context, v View) (Proposal, error) {
	if err := ctx.Err(); err != nil {
		return Proposal{}, err
	}
	if p.Selector == nil {
		return Proposal{}, ErrSelection
	}
	if err := validateLatestReferences(v); err != nil {
		return Proposal{}, err
	}
	if v.RemainingAttempts <= 0 {
		return Proposal{}, ErrNoProposal
	}
	if v.RemainingAttempts > 3 || len(v.History) > 3 {
		return Proposal{}, errors.New("invalid selector attempt budget")
	}
	seen := map[string]bool{}
	for _, feedback := range v.History {
		if !candidates.ValidID(feedback.Variant) || seen[feedback.Variant] {
			return Proposal{}, errors.New("invalid selector history")
		}
		seen[feedback.Variant] = true
	}
	allowed := []string{}
	for _, id := range candidates.IDs() {
		if !seen[id] {
			allowed = append(allowed, id)
		}
	}
	if len(allowed) == 0 {
		return Proposal{}, ErrNoProposal
	}
	// Give the selector independent slices; it cannot rewrite loop-owned history
	// or the locally validated choice set by mutating its arguments.
	input := v
	input.History = append([]Feedback{}, v.History...)
	input.CandidateChanges = append([]CandidateChange{}, v.CandidateChanges...)
	if v.BusinessIncident != nil {
		c := *v.BusinessIncident
		input.BusinessIncident = &c
	}
	d, err := p.Selector.Select(ctx, input, append([]string{}, allowed...))
	if e := ctx.Err(); e != nil {
		return Proposal{}, e
	}
	if err != nil {
		return Proposal{}, ErrSelection
	}
	if d.Abstain {
		if d.Candidate != "" {
			return Proposal{}, errors.New("ambiguous decision: abstention and candidate")
		}
		return Proposal{}, ErrAbstained
	}
	if !candidates.ValidID(d.Candidate) || seen[d.Candidate] {
		return Proposal{}, errors.New("decision is not an allowed untried candidate")
	}
	proposal := Proposal{Variant: d.Candidate, Rationale: fmt.Sprintf("Candidate %s selected for testing. Deterministic audit text, not selector reasoning or a recovery claim.", d.Candidate), References: []string{v.LatestBehavior.ID, v.LatestProof.ID}}
	if err := ValidateProposal(proposal, v); err != nil {
		return Proposal{}, err
	}
	return proposal, nil
}
