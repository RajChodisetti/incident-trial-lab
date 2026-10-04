// Package lifecycle supervises a single local trial process group.
// It does not provide persistence, restart recovery, or distributed scheduling.
package lifecycle

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"syscall"
	"time"
)

type Outcome string

const (
	Completed Outcome = "completed"
	Failed    Outcome = "failed"
	Canceled  Outcome = "canceled"
	TimedOut  Outcome = "timed_out"
)

type CleanupResult struct {
	Complete bool             `json:"complete"`
	Errors   []string         `json:"errors,omitempty"`
	Clusters []ClusterCleanup `json:"clusters"`
}

type ClusterCleanup struct {
	Name      string   `json:"name"`
	Found     []string `json:"found_container_ids,omitempty"`
	Remaining []string `json:"remaining_container_ids,omitempty"`
	Error     string   `json:"error,omitempty"`
}

type Result struct {
	ProcessGroupID int           `json:"process_group_id,omitempty"`
	Outcome        Outcome       `json:"outcome"`
	ExitCode       int           `json:"exit_code"`
	StartedAt      string        `json:"started_at"`
	FinishedAt     string        `json:"finished_at"`
	ForcedKill     bool          `json:"forced_kill"`
	Error          string        `json:"error,omitempty"`
	Cleanup        CleanupResult `json:"cleanup"`
}

type Config struct {
	Command        []string
	Dir            string
	Env            []string
	Stdout, Stderr io.Writer
	GracePeriod    time.Duration
	CleanupTimeout time.Duration
	Cleanup        func(context.Context) CleanupResult
}

// Run terminates the entire owned process group, waits for it, then cleans up
// with a fresh bounded context. Cancellation never becomes successful completion.
func Run(ctx context.Context, c Config) Result {
	r := Result{Outcome: Failed, ExitCode: -1, StartedAt: time.Now().UTC().Format(time.RFC3339Nano)}
	finish := func() Result {
		// This deliberately does not inherit the canceled trial context.
		cleanupCtx, cancel := context.WithTimeout(context.Background(), c.CleanupTimeout)
		defer cancel()
		if c.Cleanup != nil {
			r.Cleanup = c.Cleanup(cleanupCtx)
		} else {
			r.Cleanup.Complete = true
		}
		if !r.Cleanup.Complete && r.Outcome == Completed {
			r.Outcome = Failed
			r.Error = "trial completed but resource cleanup is incomplete"
		}
		r.FinishedAt = time.Now().UTC().Format(time.RFC3339Nano)
		return r
	}
	if c.GracePeriod <= 0 || c.CleanupTimeout <= 0 || len(c.Command) == 0 {
		r.Error = "invalid lifecycle configuration"
		return finish()
	}
	if ctx.Err() != nil {
		r.Outcome = cancelOutcome(ctx.Err())
		r.Error = ctx.Err().Error()
		return finish()
	}
	cmd := exec.Command(c.Command[0], c.Command[1:]...)
	cmd.Dir = c.Dir
	cmd.Env = c.Env
	cmd.Stdout = c.Stdout
	cmd.Stderr = c.Stderr
	// Bound inherited stdout/stderr pipes held open by orphaned descendants.
	cmd.WaitDelay = c.GracePeriod
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		r.Error = err.Error()
		return finish()
	}
	r.ProcessGroupID = cmd.Process.Pid
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	var err error
	select {
	case err = <-done:
		r.ExitCode = cmd.ProcessState.ExitCode()
		if ctx.Err() != nil {
			r.Outcome = cancelOutcome(ctx.Err())
			r.Error = ctx.Err().Error()
		} else if err == nil {
			r.Outcome = Completed
		} else {
			r.Error = err.Error()
		}
	case <-ctx.Done():
		r.Outcome = cancelOutcome(ctx.Err())
		r.Error = ctx.Err().Error()
		if e := syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM); e != nil && !errors.Is(e, syscall.ESRCH) {
			r.Error += fmt.Sprintf("; terminate group: %v", e)
		}
		timer := time.NewTimer(c.GracePeriod)
		select {
		case err = <-done:
			timer.Stop()
		case <-timer.C:
			r.ForcedKill = true
			if e := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL); e != nil && !errors.Is(e, syscall.ESRCH) {
				r.Error += fmt.Sprintf("; kill group: %v", e)
			}
			err = <-done
		}
		r.ExitCode = cmd.ProcessState.ExitCode()
	}
	// A shell may exit before an orphaned child; terminate only this owned group.
	if e := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL); e != nil && !errors.Is(e, syscall.ESRCH) {
		r.Error += fmt.Sprintf("; reap group: %v", e)
	}
	return finish()
}

func cancelOutcome(err error) Outcome {
	if errors.Is(err, context.DeadlineExceeded) {
		return TimedOut
	}
	return Canceled
}
