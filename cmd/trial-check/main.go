package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/RajChodisetti/incident-trial-lab/internal/baseline"
	"github.com/RajChodisetti/incident-trial-lab/internal/evaluation"
	"github.com/RajChodisetti/incident-trial-lab/internal/trial"
)

func main() {
	url := flag.String("url", "http://127.0.0.1:18080", "local trial URL")
	fixture := flag.String("fixture", "scenarios/discount-rounding/comparison-workload.json", "workload")
	cycles := flag.Int("cycles", 20, "measured cycles after one warmup")
	behaviorFile := flag.String("behavior", "", "existing behavioral report to combine with budget")
	beforePath := flag.String("before", "", "before kubelet summary")
	afterPath := flag.String("after", "", "after kubelet summary")
	baselinePath := flag.String("baseline", "", "verified baseline report")
	flag.Parse()
	if *behaviorFile != "" {
		var behavior trial.Report
		readJSON(*behaviorFile, &behavior)
		var proof baseline.Report
		readJSON(*baselinePath, &proof)
		before, e1 := readSample(*beforePath)
		after, e2 := readSample(*afterPath)
		budget := trial.AssessBudget(before, after)
		if err := trial.CoversWorkload(before, after, behavior.StartedAt, behavior.FinishedAt); err != nil {
			budget.Status = evaluation.Inconclusive
			budget.Error = err.Error()
		}
		if e1 != nil || e2 != nil {
			budget.Status = evaluation.Inconclusive
			budget.Error = fmt.Sprintf("before=%v; after=%v", e1, e2)
		}
		result := struct {
			Verdict          evaluation.Verdict `json:"verdict"`
			BaselineVerified bool               `json:"baseline_verified"`
			Budget           trial.Budget       `json:"resource_budget"`
			Behavior         trial.Report       `json:"behavior"`
		}{trial.Decide(proof.BaselineReproduced, behavior.BehavioralVerdict, budget), proof.BaselineReproduced, budget, behavior}
		emit(result)
		return // A failed candidate is evidence, not a runner infrastructure failure.
	}
	data, err := os.ReadFile(*fixture)
	if err != nil {
		fatal(err)
	}
	cases, err := trial.Load(data)
	if err != nil {
		fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	report := trial.Run(ctx, &http.Client{Timeout: 2 * time.Second}, *url, cases, *cycles)
	emit(struct {
		FixtureSHA256 string `json:"fixture_sha256"`
		trial.Report
	}{fmt.Sprintf("%x", sha256.Sum256(data)), report})
	if !report.Completed {
		os.Exit(1)
	}
}

func readSample(path string) (trial.Sample, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return trial.Sample{}, err
	}
	return trial.ParseSample(data)
}
func readJSON(path string, value any) {
	data, err := os.ReadFile(path)
	if err != nil {
		fatal(err)
	}
	if err = json.Unmarshal(data, value); err != nil {
		fatal(err)
	}
}
func emit(value any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(value); err != nil {
		fatal(err)
	}
}
func fatal(err error) { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
