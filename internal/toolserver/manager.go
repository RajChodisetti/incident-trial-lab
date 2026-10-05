// Package toolserver provides bounded, process-owned operations for the local MCP demo.
package toolserver

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"regexp"
	"sort"
	"sync"
	"time"

	"github.com/RajChodisetti/incident-trial-lab/internal/baseline"
	"github.com/RajChodisetti/incident-trial-lab/internal/candidates"
	"github.com/RajChodisetti/incident-trial-lab/internal/evaluation"
	"github.com/RajChodisetti/incident-trial-lab/internal/lifecycle"
	"github.com/RajChodisetti/incident-trial-lab/internal/provenance"
	"github.com/RajChodisetti/incident-trial-lab/internal/trial"
	"github.com/RajChodisetti/incident-trial-lab/internal/worker"
)

const Scenario = "discount-rounding"
const MaxJobs = 8

var validID = regexp.MustCompile(`^[0-9a-f]{16}$`)

// Backend is supplied by the owner, never by tool arguments. Tests use simulations.
type Backend interface {
	Spec() (provenance.Spec, error)
	Execute(context.Context, string, string, string, *provenance.Experiment) Execution
	Verify(provenance.Receipt) (provenance.Receipt, error)
	VerifyExperiment(provenance.Experiment) error
}
type Execution struct {
	Baseline        *baseline.Report
	RunID           string
	State           string
	CleanupVerified bool
	Error           string
	Experiment      *provenance.Experiment
	Lifecycle       *lifecycle.Result
	Receipt         *provenance.Receipt
	Behavior        *trial.Report
	Budget          *trial.Budget
}
type Snapshot struct {
	TrialID            string             `json:"trial_id"`
	Kind               string             `json:"kind"`
	ScenarioID         string             `json:"scenario_id"`
	Variant            string             `json:"variant,omitempty"`
	ExperimentID       string             `json:"experiment_id,omitempty"`
	State              string             `json:"state"`
	RunID              string             `json:"run_id,omitempty"`
	Verdict            evaluation.Verdict `json:"verdict"`
	CleanupVerified    bool               `json:"cleanup_verified"`
	ProvenanceVerified bool               `json:"provenance_verified"`
	EligibleNow        bool               `json:"eligible_now"`
	Error              string             `json:"error,omitempty"`
	EligibilityError   string             `json:"eligibility_error,omitempty"`
}
type Evidence struct {
	Baseline          *baseline.Report       `json:"baseline,omitempty"`
	Trial             Snapshot               `json:"trial"`
	UntrustedEvidence bool                   `json:"untrusted_evidence"`
	Lifecycle         *lifecycle.Result      `json:"lifecycle,omitempty"`
	Behavior          *trial.Report          `json:"behavior,omitempty"`
	Budget            *trial.Budget          `json:"resource_budget,omitempty"`
	Receipt           *provenance.Receipt    `json:"receipt,omitempty"`
	Experiment        *provenance.Experiment `json:"experiment,omitempty"`
}
type entry struct {
	snapshot  Snapshot
	execution Execution
	ctx       context.Context
	cancel    context.CancelFunc
	finished  bool
	order     int
}
type Manager struct {
	mu          sync.Mutex
	ctx         context.Context
	cancel      context.CancelFunc
	slots       chan struct{}
	backend     Backend
	entries     map[string]*entry
	experiments map[string]provenance.Experiment
	wg          sync.WaitGroup
	closed      bool
}

func NewManager(ctx context.Context, b Backend, workers int) (*Manager, error) {
	if b == nil || workers < 1 || workers > 2 {
		return nil, fmt.Errorf("backend and one or two workers required")
	}
	ctx, cancel := context.WithCancel(ctx)
	return &Manager{ctx: ctx, cancel: cancel, slots: make(chan struct{}, workers), backend: b, entries: map[string]*entry{}, experiments: map[string]provenance.Experiment{}}, nil
}
func newID() (string, error) {
	b := make([]byte, 8)
	_, err := rand.Read(b)
	return hex.EncodeToString(b), err
}
func variantValid(v string) bool { return candidates.ValidID(v) }
func (m *Manager) Create(ctx context.Context, scenario string) (Snapshot, error) {
	if scenario != Scenario {
		return Snapshot{}, fmt.Errorf("unsupported scenario")
	}
	return m.submit(ctx, "baseline", "", "", nil)
}
func (m *Manager) Start(ctx context.Context, experimentID, variant string) (Snapshot, error) {
	if !validID.MatchString(experimentID) || !variantValid(variant) {
		return Snapshot{}, fmt.Errorf("invalid experiment or unsupported candidate")
	}
	m.mu.Lock()
	e, ok := m.experiments[experimentID]
	m.mu.Unlock()
	if !ok {
		return Snapshot{}, fmt.Errorf("experiment is not owned by this server")
	}
	if !time.Now().Before(e.ExpiresAt) {
		return Snapshot{}, fmt.Errorf("experiment expired; create a fresh baseline")
	}
	return m.submit(ctx, "candidate", variant, experimentID, &e)
}
func (m *Manager) submit(ctx context.Context, kind, variant, experimentID string, experiment *provenance.Experiment) (Snapshot, error) {
	if err := ctx.Err(); err != nil {
		return Snapshot{}, err
	}
	id, err := newID()
	if err != nil {
		return Snapshot{}, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed || m.ctx.Err() != nil {
		return Snapshot{}, fmt.Errorf("server is closing")
	}
	if len(m.entries) >= MaxJobs {
		return Snapshot{}, fmt.Errorf("session job budget exhausted; maximum eight submissions")
	}
	jobCtx, cancel := context.WithCancel(m.ctx)
	s := Snapshot{TrialID: id, Kind: kind, ScenarioID: Scenario, Variant: variant, ExperimentID: experimentID, State: "queued", Verdict: evaluation.Inconclusive}
	e := &entry{snapshot: s, ctx: jobCtx, cancel: cancel, order: len(m.entries)}
	m.entries[id] = e
	m.wg.Add(1)
	go m.run(e, experiment)
	return s, nil
}
func (m *Manager) run(e *entry, experiment *provenance.Experiment) {
	defer m.wg.Done()
	defer e.cancel()
	select {
	case m.slots <- struct{}{}:
		defer func() { <-m.slots }()
	case <-e.ctx.Done():
		m.finish(e, Execution{State: "canceled_before_start", CleanupVerified: true})
		return
	}
	if e.ctx.Err() != nil {
		m.finish(e, Execution{State: "canceled_before_start", CleanupVerified: true})
		return
	}
	m.mu.Lock()
	e.snapshot.State = "running"
	s := e.snapshot
	m.mu.Unlock()
	result := m.backend.Execute(e.ctx, s.TrialID, s.Kind, s.Variant, experiment)
	m.finish(e, result)
}
func (m *Manager) finish(e *entry, r Execution) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if e.ctx.Err() != nil && r.State == "completed" {
		r.State = "canceled"
		r.Error = "cancellation takes precedence over completed observations"
		r.Receipt = nil
		r.Experiment = nil
	}
	e.execution = r
	e.finished = true
	e.snapshot.State = r.State
	e.snapshot.RunID = r.RunID
	e.snapshot.CleanupVerified = r.CleanupVerified
	e.snapshot.Error = r.Error
	observed := "inconclusive"
	if r.Receipt != nil {
		observed = string(r.Receipt.Verdict)
		e.snapshot.ProvenanceVerified = true
	}
	e.snapshot.Verdict = evaluation.Verdict(worker.BusinessVerdict(r.State, r.CleanupVerified, observed))
	if e.snapshot.Kind == "baseline" && r.Experiment != nil && r.State == "completed" && r.CleanupVerified {
		m.experiments[r.Experiment.ID] = *r.Experiment
		e.snapshot.ExperimentID = r.Experiment.ID
		e.snapshot.ProvenanceVerified = true
	}
}
func (m *Manager) Cancel(id string) (bool, error) {
	if !validID.MatchString(id) {
		return false, fmt.Errorf("invalid trial ID")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	e := m.entries[id]
	if e == nil {
		return false, fmt.Errorf("trial is not owned by this server")
	}
	if e.finished {
		return false, nil
	}
	e.cancel()
	return true, nil
}
func (m *Manager) Get(id string) (Evidence, error) {
	if !validID.MatchString(id) {
		return Evidence{}, fmt.Errorf("invalid trial ID")
	}
	m.mu.Lock()
	e := m.entries[id]
	if e == nil {
		m.mu.Unlock()
		return Evidence{}, fmt.Errorf("trial is not owned by this server")
	}
	s, r := e.snapshot, e.execution
	m.mu.Unlock()
	// Execution values are immutable after publication. Never let a stale proof
	// turn a historical outcome into current eligibility.
	if r.Experiment != nil && s.State == "completed" && s.CleanupVerified {
		err := m.backend.VerifyExperiment(*r.Experiment)
		s.EligibleNow = err == nil
		if err != nil {
			s.EligibilityError = err.Error()
		}
	}
	if r.Receipt != nil && s.State == "completed" && s.CleanupVerified {
		_, err := m.backend.Verify(*r.Receipt)
		s.EligibleNow = err == nil
		if err != nil {
			s.EligibilityError = err.Error()
		}
	}
	return Evidence{Baseline: r.Baseline, Trial: s, UntrustedEvidence: true, Lifecycle: r.Lifecycle, Behavior: r.Behavior, Budget: r.Budget, Receipt: r.Receipt, Experiment: r.Experiment}, nil
}
func (m *Manager) List(experimentID string) ([]Snapshot, error) {
	if experimentID != "" && !validID.MatchString(experimentID) {
		return nil, fmt.Errorf("invalid experiment ID")
	}
	m.mu.Lock()
	if _, ok := m.experiments[experimentID]; experimentID != "" && !ok {
		m.mu.Unlock()
		return nil, fmt.Errorf("experiment is not owned by this server")
	}
	type indexed struct {
		id    string
		order int
	}
	var ids []indexed
	for id, e := range m.entries {
		if experimentID == "" || e.snapshot.ExperimentID == experimentID {
			ids = append(ids, indexed{id, e.order})
		}
	}
	m.mu.Unlock()
	sort.Slice(ids, func(i, j int) bool { return ids[i].order < ids[j].order })
	out := make([]Snapshot, 0, len(ids))
	for _, v := range ids {
		e, err := m.Get(v.id)
		if err != nil {
			return nil, err
		}
		out = append(out, e.Trial)
	}
	return out, nil
}

// Close cancels all owned work and waits for the existing supervisor's bounded
// shutdown/cleanup. Restart recovery of this in-memory job map is not implemented.
func (m *Manager) Close() { m.mu.Lock(); m.closed = true; m.cancel(); m.mu.Unlock(); m.wg.Wait() }
