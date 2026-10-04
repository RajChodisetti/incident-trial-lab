package trial

import (
	"testing"

	"github.com/RajChodisetti/incident-trial-lab/internal/evaluation"
)

func TestBudgetAndGating(t *testing.T) {
	before := Sample{"pod-1", "2026-10-04T06:00:00Z", "2026-10-04T06:00:00Z", 100, 1000}
	after := Sample{"pod-1", "2026-10-04T06:00:10Z", "2026-10-04T06:00:10Z", 1000, 2000}
	budget := AssessBudget(before, after)
	if budget.Status != evaluation.Passed || budget.CPUDeltaNanos != 900 || budget.PeakMemoryMeasured {
		t.Fatal(budget)
	}
	if Decide(true, evaluation.Passed, budget) != evaluation.Passed || Decide(false, evaluation.Passed, budget) != evaluation.Inconclusive {
		t.Fatal("baseline gating failed")
	}
	for _, tt := range []struct {
		name   string
		sample Sample
		want   evaluation.Verdict
	}{
		{"changed pod", Sample{"other", after.CPUTime, after.MemoryTime, 1000, 2000}, evaluation.Inconclusive},
		{"reset CPU", Sample{"pod-1", after.CPUTime, after.MemoryTime, 99, 2000}, evaluation.Inconclusive},
		{"stale", before, evaluation.Inconclusive},
		{"CPU exceeded", Sample{"pod-1", after.CPUTime, after.MemoryTime, 6000000000, 2000}, evaluation.Failed},
		{"memory exceeded", Sample{"pod-1", after.CPUTime, after.MemoryTime, 1000, 33 * 1024 * 1024}, evaluation.Failed},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := AssessBudget(before, tt.sample); got.Status != tt.want {
				t.Fatal(got)
			}
		})
	}
	if Decide(true, evaluation.Failed, budget) != evaluation.Failed || Decide(true, evaluation.Passed, Budget{Status: evaluation.Inconclusive}) != evaluation.Inconclusive {
		t.Fatal("evidence gating failed")
	}
}

func TestSamples(t *testing.T) {
	data := []byte(`{"pods":[{"podRef":{"name":"synthetic-service-123","namespace":"incident-trial-lab","uid":"pod-1"},"containers":[{"name":"service","cpu":{"time":"2026-10-04T06:00:00Z","usageCoreNanoSeconds":10},"memory":{"time":"2026-10-04T06:00:00Z","workingSetBytes":1024}}]}]}`)
	sample, err := ParseSample(data)
	if err != nil || sample.CPUNanos != 10 || sample.WorkingSetBytes != 1024 {
		t.Fatalf("%+v %v", sample, err)
	}
	for _, bad := range []string{`{`, `{}`, `{"pods":[{"podRef":{"namespace":"unrelated"},"containers":[{"name":"service"}]}]}`, `{"pods":[{"podRef":{"namespace":"incident-trial-lab","uid":"pod-1"},"containers":[{"name":"service"}]}]}`} {
		if _, err := ParseSample([]byte(bad)); err == nil {
			t.Fatal("missing sample accepted")
		}
	}
}

func TestSamplingWindow(t *testing.T) {
	before := Sample{CPUTime: "2026-10-04T06:00:00Z", MemoryTime: "2026-10-04T06:00:00Z"}
	after := Sample{CPUTime: "2026-10-04T06:00:10Z", MemoryTime: "2026-10-04T06:00:10Z"}
	if CoversWorkload(before, after, "2026-10-04T06:00:01Z", "2026-10-04T06:00:05Z") != nil {
		t.Fatal("valid window rejected")
	}
	if CoversWorkload(before, after, "2026-10-04T06:00:01Z", "2026-10-04T06:00:11Z") == nil {
		t.Fatal("stale after sample accepted")
	}
	if CoversWorkload(before, after, "bad", "bad") == nil {
		t.Fatal("malformed timestamps accepted")
	}
}
