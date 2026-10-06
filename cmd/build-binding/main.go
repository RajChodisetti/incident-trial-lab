// build-binding describes the fixed project-owned candidate binary for OCI labels.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/RajChodisetti/incident-trial-lab/internal/artifact"
	"github.com/RajChodisetti/incident-trial-lab/internal/candidates"
)

func main() {
	if len(os.Args) != 2 || !candidates.ValidID(os.Args[1]) {
		fmt.Fprintln(os.Stderr, "one registered candidate required")
		os.Exit(2)
	}
	path := filepath.Join(".build/candidates", os.Args[1], "synthetic-service")
	s, err := os.Lstat(path)
	if err != nil || !s.Mode().IsRegular() {
		fmt.Fprintln(os.Stderr, "project candidate binary unavailable")
		os.Exit(1)
	}
	f, err := os.Open(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "candidate binary unreadable")
		os.Exit(1)
	}
	defer f.Close()
	b, err := artifact.ForBinary(os.Args[1], f)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	_ = json.NewEncoder(os.Stdout).Encode(b)
}
