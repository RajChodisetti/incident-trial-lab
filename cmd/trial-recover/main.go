package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/RajChodisetti/incident-trial-lab/internal/lifecycle"
)

func main() {
	id := flag.String("run-id", "", "explicit recorded run ID; never scans unrelated resources")
	timeout := flag.Duration("timeout", 60*time.Second, "whole reconciliation deadline")
	grace := flag.Duration("grace", 5*time.Second, "verified child process group TERM grace")
	flag.Parse()
	if *timeout <= 0 || *grace <= 0 {
		fatal(fmt.Errorf("positive deadlines required"))
	}
	root, err := os.Getwd()
	if err != nil {
		fatal(err)
	}
	dir, err := lifecycle.RecordDir(root, *id)
	if err != nil {
		fatal(err)
	}
	lock, err := lifecycle.LockRecord(dir)
	if err != nil {
		fatal(err)
	}
	defer lock.Close()
	record, err := lifecycle.LoadRecord(dir, *id)
	if err != nil {
		fatal(err)
	}
	owned, err := lifecycle.OwnershipSnapshots(dir, *id)
	if err != nil {
		fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	command := func(ctx context.Context, args []string) ([]byte, error) {
		step, cancel := context.WithTimeout(ctx, 15*time.Second)
		defer cancel()
		out, err := exec.CommandContext(step, args[0], args[1:]...).CombinedOutput()
		if err != nil {
			return out, fmt.Errorf("%s: %w", args[0], err)
		}
		return out, nil
	}
	recovery := lifecycle.Recovery{OriginalOutcome: "unknown", BusinessVerdict: "inconclusive", ReconciledAt: time.Now().UTC().Format(time.RFC3339Nano)}
	if record.Result != nil {
		recovery.OriginalOutcome = string(record.Result.Outcome)
	}
	if record.ProcessGroupID > 0 {
		identity, e := lifecycle.ProcessIdentity(ctx, record.ProcessGroupID, command)
		if e != nil {
			recovery.Error = "process inventory unavailable: " + e.Error()
		} else if identity == "" {
			recovery.ProcessesVerified = groupGone(record.ProcessGroupID)
			if !recovery.ProcessesVerified {
				recovery.Error = "group leader absent but descendants may remain; refusing signals"
			}
		} else if identity != record.ProcessIdentity || !strings.Contains(identity, "lifecycle-owner="+*id) {
			recovery.Error = "process identity changed; refusing signals"
		} else {
			if e = syscall.Kill(-record.ProcessGroupID, syscall.SIGTERM); e != nil && !errors.Is(e, syscall.ESRCH) {
				recovery.Error = e.Error()
			} else {
				end := time.Now().Add(*grace)
				for !groupGone(record.ProcessGroupID) && time.Now().Before(end) && ctx.Err() == nil {
					time.Sleep(100 * time.Millisecond)
				}
				if !groupGone(record.ProcessGroupID) && ctx.Err() == nil {
					_ = syscall.Kill(-record.ProcessGroupID, syscall.SIGKILL)
				}
				for attempt := 0; attempt < 20 && !groupGone(record.ProcessGroupID) && ctx.Err() == nil; attempt++ {
					time.Sleep(100 * time.Millisecond)
				}
				recovery.ProcessesVerified = groupGone(record.ProcessGroupID)
				if !recovery.ProcessesVerified {
					recovery.Error = "process group termination unverified"
				}
			}
		}
	} else {
		recovery.Error = "child-start record missing; process ownership cannot be reconstructed"
	}
	recovery.Cleanup = lifecycle.CleanRecordedResources(ctx, *id, "./.tools/kind-v0.33.0/kind", owned, command)
	if recovery.ProcessesVerified && recovery.Cleanup.Complete {
		if problems := lifecycle.RemoveLocalState(root, *id); len(problems) > 0 {
			recovery.Cleanup.Complete = false
			recovery.Cleanup.Errors = append(recovery.Cleanup.Errors, problems...)
		}
	}
	record.Recovery = &recovery
	if recovery.ProcessesVerified && recovery.Cleanup.Complete {
		record.State = "reconciled"
	}
	if err = lifecycle.SaveRecord(dir, record); err != nil {
		fatal(err)
	}
	data, err := json.MarshalIndent(recovery, "", "  ")
	if err != nil {
		fatal(err)
	}
	if err = os.WriteFile(filepath.Join(dir, "recovery.json"), append(data, '\n'), 0600); err != nil {
		fatal(err)
	}
	fmt.Println(string(data))
	if !recovery.ProcessesVerified || !recovery.Cleanup.Complete {
		os.Exit(1)
	}
}

func groupGone(pid int) bool { return errors.Is(syscall.Kill(-pid, 0), syscall.ESRCH) }
func fatal(err error)        { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
