package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/RajChodisetti/incident-trial-lab/internal/baseline"
	"net/http"
	"os"
	"time"
)

func main() {
	url := flag.String("url", "http://127.0.0.1:18080", "local service URL")
	fixture := flag.String("fixture", "scenarios/discount-rounding/workload.json", "workload fixture")
	flag.Parse()
	data, err := os.ReadFile(*fixture)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	cases, err := baseline.Load(data)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	report := baseline.Run(ctx, &http.Client{Timeout: 2 * time.Second}, *url, cases)
	output := struct {
		FixtureSHA256 string `json:"fixture_sha256"`
		baseline.Report
	}{fmt.Sprintf("%x", sha256.Sum256(data)), report}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(output); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if !report.BaselineReproduced {
		os.Exit(1)
	}
}
