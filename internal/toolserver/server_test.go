package toolserver

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/RajChodisetti/incident-trial-lab/internal/evaluation"
	"github.com/RajChodisetti/incident-trial-lab/internal/lifecycle"
	"github.com/RajChodisetti/incident-trial-lab/internal/provenance"
	"github.com/RajChodisetti/incident-trial-lab/internal/trial"
	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// simulatedBackend never starts a process, cluster, model or network service.
type simulatedBackend struct {
	mu           sync.Mutex
	active, peak int
	gate         chan struct{}
	verifyErr    error
}

func (b *simulatedBackend) Spec() (provenance.Spec, error) {
	return provenance.Spec{Candidates: map[string]string{"fixed": "unit", "ineffective": "unit", "misleading": "unit"}}, nil
}
func (b *simulatedBackend) Verify(r provenance.Receipt) (provenance.Receipt, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return r, b.verifyErr
}
func (b *simulatedBackend) VerifyExperiment(e provenance.Experiment) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.verifyErr
}
func (b *simulatedBackend) Execute(ctx context.Context, id, kind, variant string, e *provenance.Experiment) Execution {
	b.mu.Lock()
	b.active++
	if b.active > b.peak {
		b.peak = b.active
	}
	b.mu.Unlock()
	defer func() { b.mu.Lock(); b.active--; b.mu.Unlock() }()
	if b.gate != nil {
		select {
		case <-b.gate:
		case <-ctx.Done():
			return Execution{State: "canceled", CleanupVerified: true}
		}
	}
	if kind == "baseline" {
		spec, _ := b.Spec()
		exp := provenance.Experiment{Spec: spec, ID: id, CreatedAt: time.Now(), ExpiresAt: time.Now().Add(provenance.Lifetime)}
		return Execution{RunID: id, State: "completed", CleanupVerified: true, Experiment: &exp, Lifecycle: &lifecycle.Result{Outcome: lifecycle.Completed, Cleanup: lifecycle.CleanupResult{Complete: true}}}
	}
	verdict := evaluation.Passed
	if variant != "fixed" {
		verdict = evaluation.Failed
	}
	receipt := provenance.Receipt{Spec: e.Spec, ExperimentID: e.ID, AttemptID: id, CandidateRunID: id, Variant: variant, Verdict: verdict}
	return Execution{RunID: id, State: "completed", CleanupVerified: true, Receipt: &receipt, Behavior: &trial.Report{Completed: true, Requests: 120, BehavioralVerdict: verdict}, Budget: &trial.Budget{Status: evaluation.Passed}}
}
func terminal(s Snapshot) bool { return s.State != "queued" && s.State != "running" }
func await(t *testing.T, m *Manager, id string) Evidence {
	t.Helper()
	deadline := time.After(3 * time.Second)
	tick := time.NewTicker(time.Millisecond)
	defer tick.Stop()
	for {
		e, err := m.Get(id)
		if err != nil {
			t.Fatal(err)
		}
		if terminal(e.Trial) {
			return e
		}
		select {
		case <-deadline:
			t.Fatal("job did not finish")
		case <-tick.C:
		}
	}
}
func decodeResult(t *testing.T, r *mcp.CallToolResult, out any) {
	t.Helper()
	data, err := json.Marshal(r.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(data, out); err != nil {
		t.Fatalf("structured output %s: %v", data, err)
	}
}
func connect(t *testing.T, version string) (*Manager, *simulatedBackend, *mcp.ClientSession, context.Context) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	backend := &simulatedBackend{}
	manager, err := NewManager(ctx, backend, 1)
	if err != nil {
		t.Fatal(err)
	}
	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	server := NewServer(manager)
	done := make(chan error, 1)
	go func() { done <- server.Run(ctx, serverTransport) }()
	client := mcp.NewClient(&mcp.Implementation{Name: "offline-tests", Version: "1"}, nil)
	session, err := client.Connect(ctx, clientTransport, &mcp.ClientSessionOptions{ProtocolVersion: version})
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	t.Cleanup(func() { session.Close(); cancel(); manager.Close(); <-done })
	return manager, backend, session, ctx
}
func call(t *testing.T, ctx context.Context, s *mcp.ClientSession, name string, args map[string]any) *mcp.CallToolResult {
	t.Helper()
	r, err := s.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil || r.IsError {
		t.Fatalf("%s failed: result %+v error %v", name, r, err)
	}
	return r
}
func TestOfflineProtocol(t *testing.T) {
	for _, version := range []string{"2026-07-28", "2025-11-25"} {
		t.Run(version, func(t *testing.T) {
			manager, backend, session, ctx := connect(t, version)
			tools, err := session.ListTools(ctx, nil)
			if err != nil {
				t.Fatal(err)
			}
			if len(tools.Tools) != 8 {
				t.Fatalf("%d tools", len(tools.Tools))
			}
			schemas := map[string]*jsonschema.Resolved{}
			for _, tool := range tools.Tools {
				data, _ := json.Marshal(tool.InputSchema)
				var schema jsonschema.Schema
				if err = json.Unmarshal(data, &schema); err != nil {
					t.Fatal(err)
				}
				resolved, err := schema.Resolve(nil)
				if err != nil {
					t.Fatal(err)
				}
				if err = resolved.Validate(map[string]any{"command": "rm -rf /"}); err == nil {
					t.Fatalf("%s accepts arbitrary property", tool.Name)
				}
				if tool.OutputSchema == nil {
					t.Fatalf("%s has no output schema", tool.Name)
				}
				outBytes, _ := json.Marshal(tool.OutputSchema)
				var outputSchema jsonschema.Schema
				if err = json.Unmarshal(outBytes, &outputSchema); err != nil {
					t.Fatal(err)
				}
				schemas[tool.Name], err = outputSchema.Resolve(nil)
				if err != nil {
					t.Fatal(err)
				}
			}
			check := func(name string, args map[string]any, out any) {
				r := call(t, ctx, session, name, args)
				raw, _ := json.Marshal(r.StructuredContent)
				var instance any
				json.Unmarshal(raw, &instance)
				if err := schemas[name].Validate(instance); err != nil {
					t.Fatalf("%s output mismatch: %v", name, err)
				}
				decodeResult(t, r, out)
			}
			var catalog Catalog
			check("list_scenarios", map[string]any{}, &catalog)
			if catalog.ScenarioID != Scenario || catalog.WorkerLimit != 1 || catalog.SessionJobLimit != 8 {
				t.Fatalf("catalog %+v", catalog)
			}
			for _, bad := range []struct {
				name string
				args map[string]any
			}{{"create_experiment", map[string]any{"scenario_id": "../../employer"}}, {"start_trial", map[string]any{"experiment_id": "1111111111111111", "variant": "custom", "shell": "sh"}}, {"get_evidence", map[string]any{"trial_id": "1111111111111111", "section": "/etc/passwd"}}, {"get_trial", map[string]any{"trial_id": "../../other"}}, {"get_trial", map[string]any{"trial_id": "1111111111111111"}}, {"cancel_trial", map[string]any{"trial_id": "1111111111111111"}}} {
				r, err := session.CallTool(ctx, &mcp.CallToolParams{Name: bad.name, Arguments: bad.args})
				if err == nil && !r.IsError {
					t.Fatalf("accepted %+v", bad)
				}
			}
			var base Snapshot
			check("create_experiment", map[string]any{"scenario_id": Scenario}, &base)
			experiment := await(t, manager, base.TrialID).Trial.ExperimentID
			var job Snapshot
			check("start_trial", map[string]any{"experiment_id": experiment, "variant": "fixed"}, &job)
			await(t, manager, job.TrialID)
			var status Snapshot
			check("get_trial", map[string]any{"trial_id": job.TrialID}, &status)
			if status.Verdict != evaluation.Passed || !status.ProvenanceVerified || !status.EligibleNow || !status.CleanupVerified {
				t.Fatalf("status %+v", status)
			}
			var evidence Evidence
			for _, section := range []string{"behavior", "lifecycle", "provenance"} {
				check("get_evidence", map[string]any{"trial_id": job.TrialID, "section": section}, &evidence)
				if !evidence.UntrustedEvidence {
					t.Fatal("evidence lacks data-only marker")
				}
			}
			var listed Trials
			check("list_trials", map[string]any{"experiment_id": experiment}, &listed)
			check("list_trials", map[string]any{}, &listed)
			if len(listed.Trials) != 2 {
				t.Fatalf("list %+v", listed)
			}
			var comparison Comparison
			check("compare_trials", map[string]any{"experiment_id": experiment}, &comparison)
			if len(comparison.Candidates) != 1 {
				t.Fatalf("comparison %+v", comparison)
			}
			var canceled Canceled
			check("cancel_trial", map[string]any{"trial_id": job.TrialID}, &canceled)
			if canceled.Requested {
				t.Fatal("finished trial was canceled")
			}
			backend.mu.Lock()
			backend.verifyErr = fmt.Errorf("simulated changed evidence")
			backend.mu.Unlock()
			check("get_trial", map[string]any{"trial_id": job.TrialID}, &status)
			if status.EligibleNow || status.Verdict != evaluation.Passed {
				t.Fatal("freshness and historical verdict conflated")
			}
		})
	}
}
func TestBoundsAndClose(t *testing.T) {
	ctx := context.Background()
	b := &simulatedBackend{gate: make(chan struct{})}
	m, err := NewManager(ctx, b, 2)
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	var jobs []Snapshot
	for range MaxJobs {
		s, err := m.Create(ctx, Scenario)
		if err != nil {
			t.Fatal(err)
		}
		jobs = append(jobs, s)
	}
	if _, err = m.Create(ctx, Scenario); err == nil {
		t.Fatal("unbounded admission")
	}
	deadline := time.Now().Add(time.Second)
	for {
		b.mu.Lock()
		active := b.active
		b.mu.Unlock()
		if active == 2 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("workers did not start")
		}
		time.Sleep(time.Millisecond)
	}
	m.Close()
	for _, job := range jobs {
		e := await(t, m, job.TrialID)
		if e.Trial.Verdict != evaluation.Inconclusive || !e.Trial.CleanupVerified || e.Trial.EligibleNow {
			t.Fatalf("canceled %+v", e)
		}
	}
	b.mu.Lock()
	peak := b.peak
	b.mu.Unlock()
	if peak != 2 {
		t.Fatalf("peak %d", peak)
	}
	if _, err = m.Create(ctx, Scenario); err == nil {
		t.Fatal("closed manager accepted job")
	}
}
func TestCancellationIsolated(t *testing.T) {
	b := &simulatedBackend{}
	m, _ := NewManager(context.Background(), b, 1)
	defer m.Close()
	base, _ := m.Create(context.Background(), Scenario)
	e := await(t, m, base.TrialID)
	b.gate = make(chan struct{})
	first, _ := m.Start(context.Background(), e.Trial.ExperimentID, "fixed")
	deadline := time.Now().Add(time.Second)
	for {
		running, _ := m.Get(first.TrialID)
		if running.Trial.State == "running" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("first worker did not start")
		}
		time.Sleep(time.Millisecond)
	}
	second, _ := m.Start(context.Background(), e.Trial.ExperimentID, "misleading")
	if _, err := m.Cancel(second.TrialID); err != nil {
		t.Fatal(err)
	}
	canceled := await(t, m, second.TrialID)
	if canceled.Trial.State != "canceled_before_start" || canceled.Trial.Verdict != evaluation.Inconclusive {
		t.Fatalf("canceled %+v", canceled)
	}
	close(b.gate)
	completed := await(t, m, first.TrialID)
	if completed.Trial.Verdict != evaluation.Passed {
		t.Fatalf("peer %+v", completed)
	}
}
func TestAdmissionValidation(t *testing.T) {
	if _, err := NewManager(context.Background(), &simulatedBackend{}, 3); err == nil {
		t.Fatal("three workers accepted")
	}
	m, _ := NewManager(context.Background(), &simulatedBackend{}, 1)
	defer m.Close()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := m.Create(ctx, Scenario); err == nil {
		t.Fatal("canceled request accepted")
	}
	if _, err := m.Create(context.Background(), "other"); err == nil {
		t.Fatal("unsupported fixture")
	}
	if _, err := m.Start(context.Background(), "1111111111111111", "fixed"); err == nil {
		t.Fatal("unowned experiment")
	}
	if _, err := m.List("../other"); err == nil {
		t.Fatal("invalid experiment")
	}
}
