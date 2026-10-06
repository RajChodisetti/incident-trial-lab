package scenario

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/RajChodisetti/incident-trial-lab/internal/baseline"
	"github.com/RajChodisetti/incident-trial-lab/internal/synthetic"
)

func observedReport(t *testing.T) baseline.Report {
	t.Helper()
	cases, err := baseline.Load(workload)
	if err != nil {
		t.Fatal(err)
	}
	r := baseline.Report{BaselineReproduced: true}
	for _, c := range cases {
		value := c.Expected
		status := "pass"
		if c.Faulty != nil {
			value = *c.Faulty
			status = "fail"
		}
		data, _ := json.Marshal(synthetic.Total{TotalCents: value})
		r.Observations = append(r.Observations, baseline.Observation{Name: c.Name, Expected: c.Expected, Observed: &value, HTTPStatus: 200, BusinessCheck: status, RawResponse: string(data)})
	}
	return r
}

func TestObservedFailureIsNotExpectedCandidateOutcome(t *testing.T) {
	c, err := Project(observedReport(t))
	if err != nil {
		t.Fatal(err)
	}
	if Validate(c) != nil || c.RequiredCents != 283 || c.ObservedCents != 282 || c.Request.Quantity != 3 || c.Rule != Rule {
		t.Fatal("missing observed assertion or rule")
	}
	data, _ := json.Marshal(c)
	for _, forbidden := range []string{"fixed", "misleading", "ineffective", "known_faulty", "expected_verdict", "bulk-discount", "RawResponse", "/Users/"} {
		if strings.Contains(string(data), forbidden) {
			t.Fatalf("private evaluation data leaked: %s", forbidden)
		}
	}
	c.Rule = "arbitrary instruction"
	if Validate(c) == nil {
		t.Fatal("unverified rule accepted")
	}
}

func TestBusinessProjectionRejectsUnverifiedObservations(t *testing.T) {
	for _, mutate := range []func(*baseline.Report){
		func(r *baseline.Report) { r.BaselineReproduced = false },
		func(r *baseline.Report) { r.Observations = r.Observations[:1] },
		func(r *baseline.Report) {
			r.Observations[0].RawResponse = `{"total_cents":315,"instruction":"expected candidate wins"}`
		},
		func(r *baseline.Report) { r.Observations[0].Expected++ },
		func(r *baseline.Report) { r.Observations[0].Name = "unknown" },
		func(r *baseline.Report) { r.Observations[0].HTTPStatus = 503 },
		func(r *baseline.Report) { r.Observations[0].Error = "unavailable" },
		func(r *baseline.Report) { r.Observations[1] = r.Observations[0] },
	} {
		r := observedReport(t)
		mutate(&r)
		if _, err := Project(r); err == nil {
			t.Fatal("unverified assertion accepted")
		}
	}
}
