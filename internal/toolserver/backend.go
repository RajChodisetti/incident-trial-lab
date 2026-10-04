package toolserver

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/RajChodisetti/incident-trial-lab/internal/baseline"
	"github.com/RajChodisetti/incident-trial-lab/internal/lifecycle"
	"github.com/RajChodisetti/incident-trial-lab/internal/provenance"
	"github.com/RajChodisetti/incident-trial-lab/internal/trial"
)

type LocalBackend struct {
	root, logs string
	timeout    time.Duration
	proofMu    sync.Mutex
}

func NewLocalBackend(root string, timeout time.Duration) (*LocalBackend, error) {
	if !filepath.IsAbs(root) || timeout < time.Minute || timeout > 10*time.Minute {
		return nil, fmt.Errorf("absolute project root and 1m..10m deadline required")
	}
	b, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil || !strings.HasPrefix(string(b), "module github.com/RajChodisetti/incident-trial-lab\n") {
		return nil, fmt.Errorf("run from the Incident Trial Lab project root")
	}
	path := filepath.Join(root, ".build", "trial-run")
	s, err := os.Lstat(path)
	if err != nil || !s.Mode().IsRegular() || s.Mode()&0111 == 0 {
		return nil, fmt.Errorf("build .build/trial-run first")
	}
	if _, err = provenance.CurrentSpec(root); err != nil {
		return nil, err
	}
	evidence := filepath.Join(root, "evidence")
	if err = os.MkdirAll(evidence, 0700); err != nil {
		return nil, err
	}
	s, err = os.Lstat(evidence)
	if err != nil || !s.IsDir() || s.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("evidence must be an owned real directory")
	}
	id, err := newID()
	if err != nil {
		return nil, err
	}
	logs := filepath.Join(evidence, "mcp-session-"+id)
	if err = os.Mkdir(logs, 0700); err != nil {
		return nil, err
	}
	return &LocalBackend{root: root, logs: logs, timeout: timeout}, nil
}
func (b *LocalBackend) Spec() (provenance.Spec, error) { return provenance.CurrentSpec(b.root) }
func (b *LocalBackend) Verify(r provenance.Receipt) (provenance.Receipt, error) {
	b.proofMu.Lock()
	defer b.proofMu.Unlock()
	return provenance.Verify(b.root, r.ExperimentID, r.AttemptID, time.Now().UTC())
}
func (b *LocalBackend) VerifyExperiment(e provenance.Experiment) error {
	b.proofMu.Lock()
	defer b.proofMu.Unlock()
	_, err := provenance.Load(b.root, e.ID, time.Now().UTC())
	return err
}
func (b *LocalBackend) Execute(ctx context.Context, jobID, kind, variant string, e *provenance.Experiment) Execution {
	r := Execution{State: "start_failed", CleanupVerified: true}
	if !validID.MatchString(jobID) || (kind != "baseline" && kind != "candidate") {
		r.Error = "invalid owned job"
		return r
	}
	args := []string{"-mode", kind, "-timeout", b.timeout.String()}
	var a provenance.Attempt
	if kind == "candidate" {
		if e == nil || !variantValid(variant) {
			r.Error = "registered experiment/candidate required"
			return r
		}
		b.proofMu.Lock()
		current, err := provenance.Load(b.root, e.ID, time.Now().UTC())
		var proof string
		if err == nil {
			proof, err = provenance.BaselineProofPath(b.root, current)
		}
		if err == nil {
			a, err = provenance.Reserve(b.root, current.ID, variant, time.Now().UTC())
		}
		b.proofMu.Unlock()
		if err != nil {
			r.State = "provenance_refused"
			r.Error = err.Error()
			return r
		}
		args = append(args, "-variant", variant, "-baseline", proof)
	}
	if ctx.Err() != nil {
		r.State = "canceled_before_start"
		return r
	}
	path := filepath.Join(b.logs, jobID+".log")
	log, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		r.Error = err.Error()
		return r
	}
	cmd := exec.Command(filepath.Join(b.root, ".build", "trial-run"), args...)
	cmd.Dir = b.root
	cmd.Stdout = log
	cmd.Stderr = log
	cmd.WaitDelay = 3 * time.Second
	if err = cmd.Start(); err != nil {
		log.Close()
		r.Error = err.Error()
		return r
	}
	r.CleanupVerified = false
	r.State = "needs_recovery"
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	forced := false
	select {
	case err = <-done:
	case <-ctx.Done():
		_ = cmd.Process.Signal(syscall.SIGTERM) // Only this live owned supervisor PID.
		timer := time.NewTimer(75 * time.Second)
		select {
		case err = <-done:
			timer.Stop()
		case <-timer.C:
			forced = true
			_ = cmd.Process.Kill()
			err = <-done
		}
	}
	if closeErr := log.Close(); closeErr != nil {
		r.Error = closeErr.Error()
	}
	f, readErr := os.Open(path)
	if readErr != nil {
		r.Error = "supervisor log unavailable"
		return r
	}
	scanner := bufio.NewScanner(io.LimitReader(f, 1024))
	scanner.Scan()
	line := scanner.Text()
	f.Close()
	parts := strings.Fields(line)
	if len(parts) > 2 && parts[0] == "Lifecycle" && parts[1] == "run" {
		r.RunID = strings.TrimSuffix(parts[2], ";")
	}
	if !validID.MatchString(r.RunID) {
		r.RunID = ""
		r.Error = "supervisor identity unavailable; owned recovery review required"
		return r
	}
	dir, readErr := lifecycle.RecordDir(b.root, r.RunID)
	if readErr != nil {
		r.Error = readErr.Error()
		return r
	}
	record, readErr := lifecycle.LoadRecord(dir, r.RunID)
	if readErr != nil || record.Result == nil {
		r.Error = "terminal supervisor journal unavailable; explicit recovery required"
		return r
	}
	r.Lifecycle = record.Result
	r.CleanupVerified = record.Result.Cleanup.Complete
	r.State = string(record.Result.Outcome)
	if record.State != "finished" || forced {
		r.State = "needs_recovery"
		r.CleanupVerified = false
		r.Error = "supervisor termination unverified; explicit recorded recovery required"
		return r
	}
	if ctx.Err() != nil {
		r.State = "canceled"
	}
	if err != nil && r.State == "completed" {
		r.State = "failed"
		r.Error = err.Error()
	}
	if kind == "candidate" {
		b.readCandidate(&r, dir, variant)
	} else {
		matches, _ := filepath.Glob(filepath.Join(dir, "baseline-*"))
		if len(matches) == 1 {
			s, err := os.Lstat(matches[0])
			if err == nil && s.IsDir() && s.Mode()&os.ModeSymlink == 0 {
				var report baseline.Report
				if readJSON(filepath.Join(matches[0], "check.json"), &report) == nil {
					r.Baseline = &report
				}
			}
		}
	}
	if r.State != "completed" || !r.CleanupVerified {
		if r.Error == "" {
			r.Error = record.Result.Error
		}
		return r
	}
	b.proofMu.Lock()
	defer b.proofMu.Unlock()
	if ctx.Err() != nil {
		r.State = "canceled"
		return r
	}
	if kind == "baseline" {
		experiment, err := provenance.Prepare(b.root, r.RunID, time.Now().UTC())
		if err != nil {
			r.State = "provenance_refused"
			r.Error = err.Error()
		} else {
			r.Experiment = &experiment
		}
	} else {
		receipt, err := provenance.Bind(b.root, a, r.RunID, time.Now().UTC())
		if err != nil {
			r.State = "provenance_refused"
			r.Error = err.Error()
		} else {
			r.Receipt = &receipt
		}
	}
	return r
}

// Fixed section names only. Evidence is bounded and symlinks are refused. Logs
// never enter tool instructions; raw evidence is explicitly returned as data.
func readJSON(path string, v any) error {
	s, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !s.Mode().IsRegular() || s.Size() > 2*1024*1024 {
		return fmt.Errorf("evidence is not a bounded regular file")
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	dec := json.NewDecoder(io.LimitReader(f, 2*1024*1024+1))
	if err = dec.Decode(v); err != nil {
		return err
	}
	if dec.Decode(new(any)) != io.EOF {
		return fmt.Errorf("extra evidence data")
	}
	return nil
}
func (b *LocalBackend) readCandidate(r *Execution, dir, variant string) {
	matches, err := filepath.Glob(filepath.Join(dir, "candidate-"+variant+"-*"))
	if err != nil || len(matches) != 1 {
		return
	}
	s, err := os.Lstat(matches[0])
	if err != nil || !s.IsDir() || s.Mode()&os.ModeSymlink != 0 {
		return
	}
	var behavior trial.Report
	if readJSON(filepath.Join(matches[0], "behavior.json"), &behavior) == nil {
		r.Behavior = &behavior
	}
	var result struct {
		Budget trial.Budget `json:"resource_budget"`
	}
	if readJSON(filepath.Join(matches[0], "result.json"), &result) == nil {
		r.Budget = &result.Budget
	}
}
