package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/RajChodisetti/incident-trial-lab/internal/lifecycle"
	"github.com/RajChodisetti/incident-trial-lab/internal/worker"
)

func main() {
	workers := flag.Int("workers", 1, "one or two local trial workers")
	variants := flag.String("variants", "fixed", "comma-separated candidate variants; maximum eight jobs")
	timeout := flag.Duration("timeout", 180*time.Second, "per-supervisor trial execution deadline")
	flag.Parse()
	pool, err := worker.New(*workers)
	if err != nil {
		fatal(err)
	}
	if *timeout <= 0 {
		fatal(fmt.Errorf("positive timeout required"))
	}
	var jobs []worker.Job
	for i, v := range strings.Split(*variants, ",") {
		if v != "fixed" && v != "ineffective" && v != "misleading" {
			fatal(fmt.Errorf("invalid variant"))
		}
		jobs = append(jobs, worker.Job{ID: fmt.Sprintf("job-%d", i+1), Variant: v})
	}
	if len(jobs) > 8 {
		fatal(fmt.Errorf("maximum eight jobs"))
	}
	token := make([]byte, 8)
	if _, err = rand.Read(token); err != nil {
		fatal(err)
	}
	bundle := filepath.Join("evidence", "batch-"+hex.EncodeToString(token))
	if err = os.MkdirAll(filepath.Join(bundle, "cancel"), 0700); err != nil {
		fatal(err)
	}
	fmt.Println("Batch evidence: " + bundle)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	monitorCtx, stopMonitor := context.WithCancel(context.Background())
	defer stopMonitor()
	monitorDone := make(chan struct{})
	go func() {
		defer close(monitorDone)
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-monitorCtx.Done():
				return
			case <-ticker.C:
				for _, j := range jobs {
					path := filepath.Join(bundle, "cancel", j.ID)
					stat, e := os.Lstat(path)
					if e == nil && stat.Mode().IsRegular() && pool.Cancel(j.ID) {
						os.Remove(path)
					}
				}
			}
		}
	}()
	results, err := pool.Run(ctx, jobs, func(ctx context.Context, j worker.Job) worker.Result {
		r := worker.Result{Job: j, Status: "needs_recovery", BusinessVerdict: "inconclusive"}
		path := filepath.Join(bundle, j.ID+".log")
		log, e := os.Create(path)
		if e != nil {
			r.Error = e.Error()
			return r
		}
		cmd := exec.Command("./.build/trial-run", "-mode", "candidate", "-variant", j.Variant, "-timeout", timeout.String())
		cmd.Stdout = log
		cmd.Stderr = log
		cmd.WaitDelay = 3 * time.Second
		if e = cmd.Start(); e != nil {
			log.Close()
			r.Status = "start_failed"
			r.Error = e.Error()
			r.CleanupVerified = true
			return r
		}
		done := make(chan error, 1)
		go func() { done <- cmd.Wait() }()
		select {
		case e = <-done:
		case <-ctx.Done():
			_ = cmd.Process.Signal(syscall.SIGTERM) // Only this worker's live supervisor PID.
			timer := time.NewTimer(75 * time.Second)
			select {
			case e = <-done:
				timer.Stop()
			case <-timer.C:
				_ = cmd.Process.Kill()
				e = <-done
				r.Error = "supervisor termination unverified; explicit recorded recovery required"
			}
		}
		log.Close()
		data, readErr := os.ReadFile(path)
		if readErr == nil {
			line := strings.SplitN(string(data), "\n", 2)[0]
			parts := strings.Fields(line)
			if len(parts) > 2 && parts[0] == "Lifecycle" {
				r.RunID = strings.TrimSuffix(parts[2], ";")
			}
		}
		if r.RunID == "" {
			if r.Error == "" {
				r.Error = "supervisor run identity unavailable"
			}
			return r
		}
		resultData, readErr := os.ReadFile(filepath.Join("evidence", "lifecycle-"+r.RunID, "lifecycle.json"))
		var result lifecycle.Result
		if readErr != nil || json.Unmarshal(resultData, &result) != nil {
			if r.Error == "" {
				r.Error = "terminal supervisor evidence unavailable; use explicit recovery"
			}
			return r
		}
		r.Status = string(result.Outcome)
		r.CleanupVerified = result.Cleanup.Complete
		matches, _ := filepath.Glob(filepath.Join("evidence", "lifecycle-"+r.RunID, "candidate-"+j.Variant+"-*", "result.json"))
		if len(matches) == 1 {
			data, e := os.ReadFile(matches[0])
			var business struct {
				Verdict string `json:"verdict"`
			}
			if e == nil && json.Unmarshal(data, &business) == nil && (business.Verdict == "passed" || business.Verdict == "failed" || business.Verdict == "inconclusive") {
				r.BusinessVerdict = business.Verdict
			}
		}
		if e != nil && result.Outcome == lifecycle.Completed {
			r.Status = "failed"
			r.Error = e.Error()
		}
		return r
	})
	stopMonitor()
	<-monitorDone
	if err != nil {
		fatal(err)
	}
	payload := struct {
		Workers int             `json:"workers"`
		Results []worker.Result `json:"results"`
	}{*workers, results}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		fatal(err)
	}
	if err = os.WriteFile(filepath.Join(bundle, "batch.json"), append(data, '\n'), 0600); err != nil {
		fatal(err)
	}
	fmt.Println(string(data))
	for _, r := range results {
		if r.Status != "completed" || !r.CleanupVerified {
			os.Exit(1)
		}
	}
}

func fatal(err error) { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
