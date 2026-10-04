package baseline

import (
	"context"
	"fmt"
	"github.com/RajChodisetti/incident-trial-lab/internal/synthetic"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func fixture(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile("../../scenarios/discount-rounding/workload.json")
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestReproduction(t *testing.T) {
	cases, err := Load(fixture(t))
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		name    string
		handler http.Handler
		want    bool
	}{
		{"seeded baseline", synthetic.Handler(), true},
		{"missing total", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, `{}`) }), false},
		{"unavailable service", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(503) }), false},
		{"unexpected result", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, `{"total_cents":0}`) }), false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(tt.handler)
			defer srv.Close()
			report := Run(context.Background(), srv.Client(), srv.URL, cases)
			if report.BaselineReproduced != tt.want {
				t.Fatalf("%+v", report)
			}
			if len(report.Observations) != 3 {
				t.Fatal("missing evidence")
			}
		})
	}
}

func TestUnavailableEvidence(t *testing.T) {
	cases, _ := Load(fixture(t))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	report := Run(ctx, &http.Client{Timeout: time.Second}, "http://127.0.0.1:1", cases)
	if report.BaselineReproduced || report.HealthError == "" {
		t.Fatal(report)
	}
	for _, o := range report.Observations {
		if o.BusinessCheck != "unknown" || o.Error == "" {
			t.Fatal(o)
		}
	}
}

func TestFixtureContract(t *testing.T) {
	data := string(fixture(t))
	for _, bad := range []string{`[]`, data + ` {}`, strings.Replace(data, `"expected_cents":283`, `"expected_cents":282`, 1), strings.Replace(data, `"known_faulty_cents":282`, `"known_faulty_cents":283`, 1), strings.Replace(data, `"bulk-discount"`, `"single-unit-control"`, 1)} {
		if _, err := Load([]byte(bad)); err == nil {
			t.Fatalf("accepted invalid fixture %s", bad)
		}
	}
}

func TestRunRejectsMissingFailureCase(t *testing.T) {
	cases, _ := Load(fixture(t))
	report := Run(context.Background(), nil, "", cases[:2])
	if report.BaselineReproduced || report.FixtureError == "" {
		t.Fatal(report)
	}
}
