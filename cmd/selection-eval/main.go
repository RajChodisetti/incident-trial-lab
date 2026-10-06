// selection-eval runs offline controls only. It has no model or trial transport.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"os"

	"github.com/RajChodisetti/incident-trial-lab/internal/candidates"
	"github.com/RajChodisetti/incident-trial-lab/internal/investigation"
)

type control struct{ name, logic string }

func (c control) Select(_ context.Context, v investigation.View, choices []string) (investigation.Decision, error) {
	switch c.name {
	case "first_option":
		return investigation.Decision{Candidate: choices[0]}, nil
	case "abstain":
		return investigation.Decision{Abstain: true}, nil
	case "invalid":
		return investigation.Decision{Candidate: "option-04"}, nil
	case "selector_error":
		return investigation.Decision{}, errors.New("offline control error")
	default:
		for _, change := range v.CandidateChanges {
			if change.LogicSHA256 == c.logic {
				return investigation.Decision{Candidate: change.Option}, nil
			}
		}
		return investigation.Decision{}, errors.New("control source unavailable")
	}
}
func main() {
	if len(os.Args) != 1 {
		log.Fatal("selection-eval accepts no arguments")
	}
	seed, err := os.ReadFile("docs/evidence/selection-context-offline.json")
	if err != nil {
		log.Fatal(err)
	}
	changes, err := candidates.Changes()
	if err != nil {
		log.Fatal(err)
	}
	controls := []control{{"source_oracle_fixed", changes["fixed"].LogicSHA256}, {"source_oracle_ineffective", changes["ineffective"].LogicSHA256}, {"source_oracle_misleading", changes["misleading"].LogicSHA256}, {name: "first_option"}, {name: "abstain"}, {name: "invalid"}, {name: "selector_error"}}
	type result struct {
		Control string                      `json:"control"`
		Report  investigation.HeldOutReport `json:"report"`
	}
	results := []result{}
	for _, c := range controls {
		r, err := investigation.EvaluateHeldOut(context.Background(), seed, c)
		if err != nil {
			log.Fatal(err)
		}
		results = append(results, result{c.name, r})
	}
	e := json.NewEncoder(os.Stdout)
	e.SetIndent("", "  ")
	if err = e.Encode(results); err != nil {
		log.Fatal(err)
	}
}
