package trial

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/RajChodisetti/incident-trial-lab/internal/candidates"
	"github.com/RajChodisetti/incident-trial-lab/internal/evaluation"
)

func workload(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile("../../scenarios/discount-rounding/comparison-workload.json")
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestCandidates(t *testing.T) {
	cases, err := Load(workload(t))
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		variant                        string
		recovery, regressions, verdict evaluation.Verdict
	}{
		{"fixed", evaluation.Passed, evaluation.Passed, evaluation.Passed},
		{"ineffective", evaluation.Failed, evaluation.Failed, evaluation.Failed},
		{"misleading", evaluation.Passed, evaluation.Failed, evaluation.Failed},
	} {
		t.Run(tt.variant, func(t *testing.T) {
			handler, err := candidates.Handler(tt.variant)
			if err != nil {
				t.Fatal(err)
			}
			srv := httptest.NewServer(handler)
			defer srv.Close()
			report := Run(context.Background(), srv.Client(), srv.URL, cases, 2)
			if report.Recovery != tt.recovery || report.Regressions != tt.regressions || report.BehavioralVerdict != tt.verdict || report.Requests != 12 || len(report.WarmupObservations) != 6 {
				t.Fatalf("%+v", report)
			}
		})
	}
}

func TestIncompleteEvidence(t *testing.T) {
	cases, _ := Load(workload(t))
	for _, body := range []string{`{}`, `{"total_cents":null}`, `{"total_cents":283} {}`, `{`, strings.Repeat("x", 2049)} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, body) }))
		report := Run(context.Background(), srv.Client(), srv.URL, cases, 1)
		srv.Close()
		if report.BehavioralVerdict != evaluation.Inconclusive {
			t.Fatalf("accepted %q: %+v", body, report)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	report := Run(ctx, &http.Client{Timeout: time.Second}, "http://127.0.0.1:1", cases, 1)
	if report.Completed || report.BehavioralVerdict != evaluation.Inconclusive {
		t.Fatal(report)
	}
}

func TestWorkloadValidation(t *testing.T) {
	data := string(workload(t))
	for _, bad := range []string{`[]`, data + `{}`, strings.Replace(data, `"expected_cents":283`, `"expected_cents":282`, 1), strings.Replace(data, `"role":"recovery"`, `"role":"regression"`, 1), strings.Replace(data, `"name":"bulk-discount"`, `"name":"full-discount"`, 1)} {
		if _, err := Load([]byte(bad)); err == nil {
			t.Fatalf("accepted %s", bad)
		}
	}
	cases, _ := Load(workload(t))
	if Run(context.Background(), nil, "", cases, 0).Error == "" {
		t.Fatal("accepted zero cycles")
	}
}
