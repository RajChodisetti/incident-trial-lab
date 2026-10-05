// proposal-probe makes the sole authorized historical synthetic proposal request.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/RajChodisetti/incident-trial-lab/internal/liveproposal"
)

func main() {
	if len(os.Args) != 2 || os.Args[1] != "-authorized-first-test" {
		fmt.Fprintln(os.Stderr, "requires explicit -authorized-first-test; one attempt, no retry")
		os.Exit(2)
	}
	root, e := os.Getwd()
	if e != nil {
		os.Exit(1)
	}
	ignored := exec.Command("git", "check-ignore", "-q", "--", ".env.local").Run() == nil
	e = exec.Command("git", "ls-files", "--error-unmatch", "--", ".env.local").Run()
	var exit *exec.ExitError
	if !ignored || !errors.As(e, &exit) || exit.ExitCode() != 1 {
		fmt.Fprintln(os.Stderr, "credential must remain ignored and untracked")
		os.Exit(1)
	}
	v, e := liveproposal.HistoricalBaseline(filepath.Join(root, "evidence/offline-loop-eaa167c837aeb1c5/captures.json"))
	if e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
	key, e := liveproposal.LoadKey(filepath.Join(root, ".env.local"))
	if e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Second)
	defer cancel()
	r, e := liveproposal.Probe(ctx, key, v, filepath.Join(root, "evidence/live-first-test"))
	// Structured report contains only allowlisted metadata and validated proposal.
	_ = json.NewEncoder(os.Stdout).Encode(r)
	if e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
}
