package investigation

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"

	"github.com/RajChodisetti/incident-trial-lab/internal/candidates"
)

// NeutralSelector hides semantic preset IDs at the selector boundary. One instance
// is scoped to one evaluation loop; its aliases stay stable across revisions.
// This prevents label leakage, not leakage from prompts, code or other context.
type NeutralSelector struct {
	inner     Selector
	aliasToID map[string]string
	idToAlias map[string]string
}

var _ Selector = (*NeutralSelector)(nil)

func NewNeutralSelector(inner Selector) (*NeutralSelector, error) {
	if inner == nil {
		return nil, errors.New("selector required")
	}
	order := candidates.IDs()
	for i := len(order) - 1; i > 0; i-- {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(i+1)))
		if err != nil {
			return nil, errors.New("cannot randomize candidate aliases")
		}
		j := int(n.Int64())
		order[i], order[j] = order[j], order[i]
	}
	return neutralWithOrder(inner, order), nil
}

// Constructor helper is private: tests enumerate permutations without relying on
// chance. Production uses an unbiased random permutation, not answer order.
func neutralWithOrder(inner Selector, order []string) *NeutralSelector {
	n := &NeutralSelector{inner: inner, aliasToID: map[string]string{}, idToAlias: map[string]string{}}
	for i, id := range order {
		alias := fmt.Sprintf("option-%02d", i+1)
		n.aliasToID[alias] = id
		n.idToAlias[id] = alias
	}
	return n
}

func (n *NeutralSelector) Select(ctx context.Context, v View, choices []string) (Decision, error) {
	if err := ctx.Err(); err != nil {
		return Decision{}, err
	}
	if n == nil || n.inner == nil {
		return Decision{}, ErrSelection
	}
	// Iterate aliases, rather than preserving semantic ID order in the choices.
	allowed := map[string]bool{}
	for _, id := range choices {
		if _, ok := n.idToAlias[id]; !ok || allowed[id] {
			return Decision{}, errors.New("invalid neutral candidate choices")
		}
		allowed[id] = true
	}
	if len(allowed) == 0 {
		return Decision{}, ErrNoProposal
	}
	aliases := []string{}
	for i := 1; i <= len(n.aliasToID); i++ {
		alias := fmt.Sprintf("option-%02d", i)
		if allowed[n.aliasToID[alias]] {
			aliases = append(aliases, alias)
		}
	}
	input := v
	input.History = append([]Feedback{}, v.History...)
	for i := range input.History {
		alias, ok := n.idToAlias[input.History[i].Variant]
		if !ok {
			return Decision{}, errors.New("invalid neutral candidate history")
		}
		input.History[i].Variant = alias
	}
	d, err := n.inner.Select(ctx, input, aliases)
	if e := ctx.Err(); e != nil {
		return Decision{}, e
	}
	if err != nil {
		return Decision{}, ErrSelection
	}
	if d.Abstain {
		if d.Candidate != "" {
			return Decision{}, errors.New("ambiguous neutral abstention")
		}
		return d, nil
	}
	id, ok := n.aliasToID[d.Candidate]
	if !ok || !allowed[id] {
		return Decision{}, errors.New("unknown or unavailable neutral choice")
	}
	return Decision{Candidate: id}, nil
}
