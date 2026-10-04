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
)

func main() {
	mode := flag.String("mode", "comparison", "comparison, baseline, or candidate")
	variant := flag.String("variant", "fixed", "candidate variant")
	baseline := flag.String("baseline", "docs/evidence/milestone-2-check.json", "baseline proof for candidate-only runs")
	timeout := flag.Duration("timeout", 10*time.Minute, "whole trial deadline")
	grace := flag.Duration("grace", 20*time.Second, "process group termination grace")
	cleanupTimeout := flag.Duration("cleanup-timeout", 45*time.Second, "independent cleanup deadline")
	flag.Parse()
	if *timeout <= 0 || *grace <= 0 || *cleanupTimeout <= 0 {
		fatal(fmt.Errorf("deadlines must be positive"))
	}
	root, err := os.Getwd()
	if err != nil {
		fatal(err)
	}
	if _, err = os.Stat(filepath.Join(root, "go.mod")); err != nil {
		fatal(fmt.Errorf("run from project root"))
	}
	var command []string
	switch *mode {
	case "comparison":
		command = []string{"./scripts/run-comparison.sh"}
	case "baseline":
		command = []string{"./scripts/run-baseline.sh"}
	case "candidate":
		if *variant != "fixed" && *variant != "ineffective" && *variant != "misleading" {
			fatal(fmt.Errorf("invalid variant"))
		}
		command = []string{"./scripts/run-candidate.sh", *variant, *baseline}
	default:
		fatal(fmt.Errorf("invalid mode"))
	}
	token := make([]byte, 8)
	if _, err = rand.Read(token); err != nil {
		fatal(err)
	}
	runID := hex.EncodeToString(token) // Resource identifier, not an authentication credential.
	evidence := filepath.Join("evidence", "lifecycle-"+runID)
	if err = os.MkdirAll(evidence, 0700); err != nil {
		fatal(err)
	}
	if err = os.WriteFile(filepath.Join(evidence, "supervisor.pid"), []byte(fmt.Sprintf("%d\n", os.Getpid())), 0600); err != nil {
		fatal(err)
	}
	log, err := os.Create(filepath.Join(evidence, "process.log"))
	if err != nil {
		fatal(err)
	}
	defer log.Close()
	env := []string{}
	for _, entry := range os.Environ() {
		if !strings.HasPrefix(entry, "TRIAL_RUN_ID=") && !strings.HasPrefix(entry, "TRIAL_EVIDENCE_PARENT=") {
			env = append(env, entry)
		}
	}
	env = append(env, "TRIAL_RUN_ID="+runID, "TRIAL_EVIDENCE_PARENT="+evidence)
	signalCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	ctx, cancel := context.WithTimeout(signalCtx, *timeout)
	defer cancel()
	fmt.Printf("Lifecycle run %s; evidence: %s\n", runID, evidence)
	config := lifecycle.Config{Command: command, Dir: root, Env: env, Stdout: log, Stderr: log, GracePeriod: *grace, CleanupTimeout: *cleanupTimeout}
	config.Cleanup = func(cleanupCtx context.Context) lifecycle.CleanupResult {
		command := func(ctx context.Context, args []string) ([]byte, error) {
			stepCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
			defer cancel()
			cmd := exec.CommandContext(stepCtx, args[0], args[1:]...)
			cmd.Dir = root
			output, err := cmd.CombinedOutput()
			if err != nil {
				return output, fmt.Errorf("%s: %w: %s", args[0], err, strings.TrimSpace(string(output)))
			}
			return output, nil
		}
		result := lifecycle.CleanResources(cleanupCtx, runID, "./.tools/kind-v0.33.0/kind", command)
		// Keep local access/lock state if cleanup is incomplete, for diagnosis/recovery.
		if result.Complete {
			if errors := lifecycle.RemoveLocalState(root, runID); len(errors) > 0 {
				result.Complete = false
				result.Errors = append(result.Errors, errors...)
			}
		}
		return result
	}
	result := lifecycle.Run(ctx, config)
	payload := struct {
		RunID string `json:"run_id"`
		Mode  string `json:"mode"`
		lifecycle.Result
	}{runID, *mode, result}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		fatal(err)
	}
	if err = os.WriteFile(filepath.Join(evidence, "lifecycle.json"), append(data, '\n'), 0600); err != nil {
		fatal(err)
	}
	fmt.Println(string(data))
	if result.Outcome != lifecycle.Completed || !result.Cleanup.Complete {
		os.Exit(1)
	}
}

func fatal(err error) { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
