package investigation

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/RajChodisetti/incident-trial-lab/internal/baseline"
	"github.com/RajChodisetti/incident-trial-lab/internal/provenance"
	"github.com/RajChodisetti/incident-trial-lab/internal/synthetic"
	"github.com/RajChodisetti/incident-trial-lab/internal/toolserver"
	scenario "github.com/RajChodisetti/incident-trial-lab/scenarios/discount-rounding"
)

func recordedCaptures(t *testing.T) (Capture, Capture) {
	t.Helper()
	data, err := os.ReadFile("../../scenarios/discount-rounding/workload.json")
	if err != nil {
		t.Fatal(err)
	}
	cases, err := baseline.Load(data)
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
		raw, _ := json.Marshal(synthetic.Total{TotalCents: value})
		r.Observations = append(r.Observations, baseline.Observation{Name: c.Name, Expected: c.Expected, Observed: &value, HTTPStatus: 200, BusinessCheck: status, RawResponse: string(raw)})
	}
	incident, err := scenario.Project(r)
	if err != nil {
		t.Fatal(err)
	}
	v := selectorView()
	s := toolserver.Snapshot{TrialID: v.LatestBehavior.TrialID, ScenarioID: toolserver.Scenario, Kind: "baseline", State: "completed", ExperimentID: expID, CleanupVerified: true, ProvenanceVerified: true}
	b := Capture{Reference: v.LatestBehavior, Evidence: toolserver.Evidence{Trial: s, Baseline: &r}}
	p := Capture{Reference: v.LatestProof, Evidence: toolserver.Evidence{Trial: s, Experiment: &provenance.Experiment{ID: expID, Spec: provenance.Spec{BaselineFixture: incident.WorkloadSHA256}}}}
	b.Reference.SHA256 = hashJSON(b.Evidence)
	p.Reference.SHA256 = hashJSON(p.Evidence)
	return b, p
}

func TestHistoricalIncidentProjectionValidatesCaptureHashes(t *testing.T) {
	b, p := recordedCaptures(t)
	v, err := HistoricalIncidentView(b, p)
	if err != nil || v.BusinessIncident == nil || v.BusinessIncident.ObservedCents != 282 {
		t.Fatalf("view=%+v error=%v", v, err)
	}
	p.Evidence.Experiment.Spec.BaselineFixture = strings.Repeat("0", 64)
	p.Reference.SHA256 = hashJSON(p.Evidence)
	if _, err = HistoricalIncidentView(b, p); err == nil {
		t.Fatal("wrong baseline fixture binding accepted")
	}
	b, p = recordedCaptures(t)
	b.Evidence.Baseline.Observations[0].Expected++
	if _, err = HistoricalIncidentView(b, p); err == nil {
		t.Fatal("changed capture accepted")
	}
}

func makeRecord(t *testing.T, d Decision, selectorError error) SelectionRecord {
	t.Helper()
	b, p := recordedCaptures(t)
	v, err := HistoricalIncidentView(b, p)
	if err != nil {
		t.Fatal(err)
	}
	inner := selectorFunc(func(_ context.Context, input View, choices []string) (Decision, error) {
		data, _ := json.Marshal(SelectionRequest{View: input, Choices: choices})
		for _, label := range []string{"fixed", "misleading", "ineffective", "expected_verdict", "owner_aliases", "RawResponse"} {
			if strings.Contains(string(data), label) {
				t.Fatalf("selector input leaked %s", label)
			}
		}
		if input.BusinessIncident.RequiredCents != 283 || input.BusinessIncident.ObservedCents != 282 {
			t.Fatal("observed assertion lost")
		}
		input.CandidateChanges[0].Logic = "untrusted mutation"
		input.BusinessIncident.ObservedCents = 0
		choices[0] = "untrusted mutation"
		return d, selectorError
	})
	n, j, err := NewOfflineSelectionJournal(inner)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = n.Select(context.Background(), v, []string{"fixed", "ineffective", "misleading"})
	rows := j.Records()
	if len(rows) != 1 {
		t.Fatal("record missing")
	}
	if v.BusinessIncident.ObservedCents != 282 || strings.Contains(rows[0].Request.View.CandidateChanges[0].Logic, "untrusted") {
		t.Fatal("audit or owner view mutated")
	}
	rows[0].OwnerAliases["option-01"] = "owner mutation"
	return j.Records()[0]
}

func TestSelectionRecordsReconstructExactInputsWithoutSelector(t *testing.T) {
	for _, tc := range []struct {
		d      Decision
		err    error
		status string
	}{
		{Decision{Candidate: "option-02"}, nil, "selected"}, {Decision{Abstain: true}, nil, "abstained"},
		{Decision{Candidate: "UNTRUSTED_UNKNOWN_VALUE"}, nil, "invalid"}, {Decision{}, errors.New("UNTRUSTED_ERROR_BODY"), "selector_error"},
	} {
		r := makeRecord(t, tc.d, tc.err)
		data, _ := json.Marshal(r)
		decoded, err := DecodeSelectionRecord(data)
		if err != nil || decoded.Status != tc.status || hashJSON(decoded.Request) != r.RequestSHA256 || decoded.RecordSHA256 != r.RecordSHA256 {
			t.Fatalf("status=%s error=%v", decoded.Status, err)
		}
		if strings.Contains(string(data), "UNTRUSTED_") {
			t.Fatal("raw invalid answer/error saved")
		}
		again, _ := json.Marshal(decoded)
		if string(data) != string(again) {
			t.Fatal("record reconstruction not canonical")
		}
	}
}

func TestSelectionRecordRejectsTamperingAndUnknownFields(t *testing.T) {
	r := makeRecord(t, Decision{Abstain: true}, nil)
	for _, mutate := range []func(*SelectionRecord){
		func(r *SelectionRecord) { r.Request.View.BusinessIncident.ObservedCents++ },
		func(r *SelectionRecord) { r.OwnerAliases["option-01"] = "unknown"; r.RecordSHA256 = recordDigest(*r) },
		func(r *SelectionRecord) {
			r.Request.View.CandidateChanges[0].Logic = "expected answer"
			r.RequestSHA256 = hashJSON(r.Request)
			r.RecordSHA256 = recordDigest(*r)
		},
		func(r *SelectionRecord) { r.Status = "selected"; r.RecordSHA256 = recordDigest(*r) },
	} {
		data, _ := json.Marshal(r)
		var changed SelectionRecord
		_ = json.Unmarshal(data, &changed)
		mutate(&changed)
		data, _ = json.Marshal(changed)
		if _, err := DecodeSelectionRecord(data); err == nil {
			t.Fatal("tampered record accepted")
		}
	}
	data, _ := json.Marshal(r)
	data = append(data[:len(data)-1], []byte(`,"expected_verdict":"passed"}`)...)
	if _, err := DecodeSelectionRecord(data); err == nil {
		t.Fatal("unknown privileged field accepted")
	}
}

func TestJournalBoundAndCanceledInputs(t *testing.T) {
	b, p := recordedCaptures(t)
	v, err := HistoricalIncidentView(b, p)
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	inner := selectorFunc(func(context.Context, View, []string) (Decision, error) { calls++; return Decision{Abstain: true}, nil })
	n, j, err := NewOfflineSelectionJournal(inner)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err = n.Select(ctx, v, []string{"fixed"}); !errors.Is(err, context.Canceled) || calls != 0 {
		t.Fatal("canceled input invoked selector")
	}
	for range 3 {
		_, _ = n.Select(context.Background(), v, []string{"fixed"})
	}
	if _, err = n.Select(context.Background(), v, []string{"fixed"}); err == nil || calls != 3 || len(j.Records()) != 3 {
		t.Fatal("journal limit not enforced")
	}
}

func TestCommittedOfflineRecordReconstructsUnderCurrentSnapshot(t *testing.T) {
	data, err := os.ReadFile("../../docs/evidence/selection-context-offline.json")
	if err != nil {
		t.Fatal(err)
	}
	r, err := DecodeSelectionRecord(data)
	if err != nil || r.Status != "abstained" || r.RequestSHA256 != "16c827b16afa0e59f5af2689c9bf773404d6e47609c3811413955b53c2c90f14" {
		t.Fatalf("record reconstruction failed: %v", err)
	}
}

func TestJournalRecordsCancellationWithoutAClaimedDecision(t *testing.T) {
	b, p := recordedCaptures(t)
	v, err := HistoricalIncidentView(b, p)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	inner := selectorFunc(func(context.Context, View, []string) (Decision, error) {
		cancel()
		return Decision{Candidate: "option-01"}, nil
	})
	n, j, err := NewOfflineSelectionJournal(inner)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = n.Select(ctx, v, []string{"fixed", "ineffective", "misleading"}); !errors.Is(err, context.Canceled) {
		t.Fatal("late cancellation ignored")
	}
	rows := j.Records()
	if len(rows) != 1 || rows[0].Status != "canceled" || rows[0].Decision != (Decision{}) {
		t.Fatal("canceled record claims decision")
	}
	data, _ := json.Marshal(rows[0])
	if _, err = DecodeSelectionRecord(data); err != nil {
		t.Fatal(err)
	}
}
