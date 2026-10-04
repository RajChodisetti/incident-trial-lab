// experiment registers and binds evidence produced by the existing supervisor.
// It does not run arbitrary commands, accept boolean baseline proof, or apply fixes.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/RajChodisetti/incident-trial-lab/internal/provenance"
)

func main() {
	action := flag.String("action", "inspect", "register, reserve, bind, verify, or inspect")
	baseline := flag.String("baseline-run", "", "completed fresh supervised baseline run ID")
	experiment := flag.String("experiment", "", "experiment ID")
	variant := flag.String("variant", "fixed", "registered candidate variant")
	attempt := flag.String("attempt", "", "reserved attempt ID")
	candidate := flag.String("candidate-run", "", "completed supervised candidate run ID")
	flag.Parse()
	root, err := os.Getwd()
	if err != nil {
		fatal(err)
	}
	now := time.Now().UTC()
	var result any
	switch *action {
	case "register":
		result, err = provenance.Prepare(root, *baseline, now)
	case "inspect":
		result, err = provenance.Load(root, *experiment, now)
	case "reserve":
		var e provenance.Experiment
		e, err = provenance.Load(root, *experiment, now)
		if err != nil {
			break
		}
		var proof string
		proof, err = provenance.BaselineProofPath(root, e)
		if err != nil {
			break
		}
		var a provenance.Attempt
		a, err = provenance.Reserve(root, *experiment, *variant, now)
		result = struct {
			provenance.Attempt
			BaselineProof string `json:"baseline_proof"`
		}{a, proof}
	case "verify":
		result, err = provenance.Verify(root, *experiment, *attempt, now)
	case "bind":
		var a provenance.Attempt
		a, err = provenance.LoadAttempt(root, *experiment, *attempt)
		if err == nil {
			result, err = provenance.Bind(root, a, *candidate, now)
		}
	default:
		err = fmt.Errorf("unknown action")
	}
	if err != nil {
		fatal(err)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err = enc.Encode(result); err != nil {
		fatal(err)
	}
}
func fatal(err error) { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
