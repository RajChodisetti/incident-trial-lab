// offline-loop runs a deliberately scripted proposer over actual MCP trial tools.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/RajChodisetti/incident-trial-lab/internal/investigation"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	real := flag.Bool("real", false, "explicitly run one actual kind experiment with three scripted presets")
	flag.Parse()
	if !*real {
		return fmt.Errorf("this command requires -real; go test ./internal/investigation evaluates simulated cases without clusters")
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	client, err := investigation.Connect(ctx)
	if err != nil {
		return err
	}
	defer client.Close()
	result := investigation.Run(ctx, client, &investigation.Scripted{Variants: []string{"ineffective", "misleading", "fixed"}}, investigation.Config{MaxAttempts: 3, Timeout: 8 * time.Minute, PollInterval: 250 * time.Millisecond})
	dir := filepath.Join("evidence", "offline-loop-"+result.Trace.LoopID)
	if err = os.Mkdir(dir, 0700); err != nil {
		return err
	}
	full, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	if err = os.WriteFile(filepath.Join(dir, "captures.json"), append(full, '\n'), 0600); err != nil {
		return err
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err = enc.Encode(result.Trace); err != nil {
		return err
	}
	if result.Trace.StopReason != "verified_recovery" {
		return fmt.Errorf("offline loop stopped: %s", result.Trace.StopReason)
	}
	return nil
}
