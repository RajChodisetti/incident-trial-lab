// trial-mcp is a client-launched stdio server, with no listener or account setup.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/RajChodisetti/incident-trial-lab/internal/toolserver"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	workers := flag.Int("workers", 1, "owner-selected local worker bound: one or two")
	timeout := flag.Duration("trial-timeout", 3*time.Minute, "owner-selected supervisor deadline: 1m..10m")
	flag.Parse()
	root, err := os.Getwd()
	if err != nil {
		fatal(err)
	}
	backend, err := toolserver.NewLocalBackend(root, *timeout)
	if err != nil {
		fatal(err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	manager, err := toolserver.NewManager(ctx, backend, *workers)
	if err != nil {
		fatal(err)
	}
	err = toolserver.NewServer(manager).Run(ctx, &mcp.StdioTransport{})
	// On EOF, protocol failure or signal, cancel and await owned cleanup. Nothing
	// writes protocol noise to stdout; child output stays in scoped evidence logs.
	manager.Close()
	if err != nil && !errors.Is(err, context.Canceled) {
		fatal(err)
	}
}
func fatal(err error) { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
