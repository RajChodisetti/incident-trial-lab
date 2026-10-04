// mcp-check is a deterministic offline MCP client. -real opts into the owned
// baseline/fixed comparison; no language model, account, or network listener.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/RajChodisetti/incident-trial-lab/internal/evaluation"
	"github.com/RajChodisetti/incident-trial-lab/internal/toolserver"
	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	real := flag.Bool("real", false, "run fresh owned baseline/fixed comparison plus queued cancellation")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancel()
	command := exec.Command("./.build/trial-mcp", "-workers", "1")
	command.Stderr = os.Stderr
	client := mcp.NewClient(&mcp.Implementation{Name: "incident-trial-lab-offline-check", Version: "0.1.0"}, nil)
	session, err := client.Connect(ctx, &mcp.CommandTransport{Command: command, TerminateDuration: 90 * time.Second}, nil)
	if err != nil {
		return err
	}
	defer session.Close()
	listed, err := session.ListTools(ctx, nil)
	if err != nil {
		return err
	}
	if len(listed.Tools) != 8 {
		return fmt.Errorf("expected eight tools")
	}
	schemas := map[string]*jsonschema.Resolved{}
	names := make([]string, 0, 8)
	for _, tool := range listed.Tools {
		names = append(names, tool.Name)
		raw, _ := json.Marshal(tool.OutputSchema)
		var schema jsonschema.Schema
		if err = json.Unmarshal(raw, &schema); err != nil {
			return err
		}
		schemas[tool.Name], err = schema.Resolve(nil)
		if err != nil {
			return err
		}
	}
	call := func(name string, args map[string]any, out any) error {
		r, err := session.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
		if err != nil {
			return err
		}
		if r.IsError {
			return fmt.Errorf("%s refused: %+v", name, r.Content)
		}
		raw, err := json.Marshal(r.StructuredContent)
		if err != nil {
			return err
		}
		var instance any
		if err = json.Unmarshal(raw, &instance); err != nil {
			return err
		}
		if err = schemas[name].Validate(instance); err != nil {
			return fmt.Errorf("%s output schema: %w", name, err)
		}
		return json.Unmarshal(raw, out)
	}
	refused := map[string]bool{}
	for _, bad := range []struct {
		name string
		args map[string]any
	}{{"create_experiment", map[string]any{"scenario_id": "../../other", "command": "sh"}}, {"get_evidence", map[string]any{"trial_id": "1111111111111111", "section": "/etc/passwd"}}, {"cancel_trial", map[string]any{"trial_id": "1111111111111111"}}} {
		r, err := session.CallTool(ctx, &mcp.CallToolParams{Name: bad.name, Arguments: bad.args})
		if err == nil && !r.IsError {
			return fmt.Errorf("invalid %s call accepted", bad.name)
		}
		refused[bad.name] = true
	}
	var catalog toolserver.Catalog
	if err = call("list_scenarios", map[string]any{}, &catalog); err != nil {
		return err
	}
	summary := map[string]any{"mode": "protocol_only_no_trials", "protocol_version": session.InitializeResult().ProtocolVersion, "tools": names, "invalid_calls_refused": refused, "catalog": catalog}
	if *real {
		poll := func(id string, wantRunning bool) (toolserver.Snapshot, error) {
			ticker := time.NewTicker(250 * time.Millisecond)
			defer ticker.Stop()
			for {
				var s toolserver.Snapshot
				if err := call("get_trial", map[string]any{"trial_id": id}, &s); err != nil {
					return s, err
				}
				terminal := s.State != "queued" && s.State != "running"
				if terminal || (wantRunning && s.State == "running") {
					return s, nil
				}
				select {
				case <-ctx.Done():
					return s, ctx.Err()
				case <-ticker.C:
				}
			}
		}
		var base toolserver.Snapshot
		if err = call("create_experiment", map[string]any{"scenario_id": toolserver.Scenario}, &base); err != nil {
			return err
		}
		base, err = poll(base.TrialID, false)
		if err != nil {
			return err
		}
		if base.State != "completed" || !base.EligibleNow || !base.CleanupVerified || base.ExperimentID == "" {
			return fmt.Errorf("baseline not eligible: %+v", base)
		}
		var baseEvidence toolserver.Evidence
		if err = call("get_evidence", map[string]any{"trial_id": base.TrialID, "section": "behavior"}, &baseEvidence); err != nil {
			return err
		}
		if baseEvidence.Baseline == nil || !baseEvidence.Baseline.BaselineReproduced {
			return fmt.Errorf("baseline business evidence absent")
		}
		var fixed toolserver.Snapshot
		if err = call("start_trial", map[string]any{"experiment_id": base.ExperimentID, "variant": "fixed"}, &fixed); err != nil {
			return err
		}
		if _, err = poll(fixed.TrialID, true); err != nil {
			return err
		}
		var queued toolserver.Snapshot
		if err = call("start_trial", map[string]any{"experiment_id": base.ExperimentID, "variant": "misleading"}, &queued); err != nil {
			return err
		}
		var canceled toolserver.Canceled
		if err = call("cancel_trial", map[string]any{"trial_id": queued.TrialID}, &canceled); err != nil {
			return err
		}
		queued, err = poll(queued.TrialID, false)
		if err != nil {
			return err
		}
		if queued.State != "canceled_before_start" || queued.RunID != "" || queued.Verdict != evaluation.Inconclusive || !queued.CleanupVerified {
			return fmt.Errorf("queued cancellation failed: %+v", queued)
		}
		fixed, err = poll(fixed.TrialID, false)
		if err != nil {
			return err
		}
		if fixed.Verdict != evaluation.Passed || !fixed.EligibleNow || !fixed.CleanupVerified || !fixed.ProvenanceVerified {
			return fmt.Errorf("candidate not verified: %+v", fixed)
		}
		var behavior, proof, lifecycle toolserver.Evidence
		for section, out := range map[string]*toolserver.Evidence{"behavior": &behavior, "provenance": &proof, "lifecycle": &lifecycle} {
			if err = call("get_evidence", map[string]any{"trial_id": fixed.TrialID, "section": section}, out); err != nil {
				return err
			}
		}
		if behavior.Behavior == nil || behavior.Behavior.Requests != 120 || proof.Receipt == nil || lifecycle.Lifecycle == nil || !lifecycle.Lifecycle.Cleanup.Complete {
			return fmt.Errorf("expected bounded sections absent")
		}
		var comparison toolserver.Comparison
		var trials toolserver.Trials
		if err = call("compare_trials", map[string]any{"experiment_id": base.ExperimentID}, &comparison); err != nil {
			return err
		}
		if err = call("list_trials", map[string]any{"experiment_id": base.ExperimentID}, &trials); err != nil {
			return err
		}
		summary["mode"] = "real_kind_baseline_and_fixed_candidate"
		summary["baseline"] = base
		summary["baseline_observations"] = baseEvidence.Baseline
		summary["fixed"] = fixed
		summary["queued_cancellation"] = queued
		summary["comparison"] = comparison
		summary["listed_count"] = len(trials.Trials)
		summary["receipt"] = proof.Receipt
		summary["behavior"] = map[string]any{"recovery": behavior.Behavior.Recovery, "regressions": behavior.Behavior.Regressions, "requests": behavior.Behavior.Requests}
		summary["resource_budget"] = behavior.Budget
		summary["cleanup"] = lifecycle.Lifecycle.Cleanup
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(summary)
}
