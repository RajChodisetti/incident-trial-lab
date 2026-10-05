package liveproposal

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/RajChodisetti/incident-trial-lab/internal/baseline"
	"github.com/RajChodisetti/incident-trial-lab/internal/investigation"
	"github.com/RajChodisetti/incident-trial-lab/internal/provenance"
	"github.com/RajChodisetti/incident-trial-lab/internal/toolserver"
)

type roundTrip func(*http.Request) (*http.Response, error)

func (f roundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func view() investigation.View {
	return investigation.View{ScenarioID: "discount-rounding", RemainingAttempts: 1, Signal: investigation.Signal{Recovery: "failed", Regressions: "passed", Failed: 1}, LatestBehavior: investigation.Reference{ID: "eaa167c837aeb1c5:r001", TrialID: "d2ddae27ffc3792c", Section: "behavior", SHA256: strings.Repeat("a", 64)}, LatestProof: investigation.Reference{ID: "eaa167c837aeb1c5:r002", TrialID: "d2ddae27ffc3792c", Section: "provenance", SHA256: strings.Repeat("b", 64)}}
}
func validOutput() string {
	return `{"variant":"fixed","rationale":"Propose the general formula for testing; no recovery observed.","references":["eaa167c837aeb1c5:r001","eaa167c837aeb1c5:r002"]}`
}
func response(output, status, contentType string) string {
	text, _ := json.Marshal(output)
	return fmt.Sprintf(`{"model":%q,"service_tier":"default","status":%q,"usage":{"input_tokens":1000,"output_tokens":100,"total_tokens":1100,"input_tokens_details":{"cached_tokens":200},"output_tokens_details":{"reasoning_tokens":0}},"output":[{"type":"message","role":"assistant","content":[{"type":%q,"text":%s}]}]}`, Model, status, contentType, text)
}
func fakeClient(code int, body string, calls *atomic.Int32) *http.Client {
	return &http.Client{Transport: roundTrip(func(r *http.Request) (*http.Response, error) {
		calls.Add(1)
		return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	})}
}
func TestProbeValidationAndSingleAttempt(t *testing.T) {
	cases := []struct {
		name, output, status, kind, want string
		code                             int
	}{
		{"valid", validOutput(), "completed", "output_text", "valid_proposal_not_executed", 200},
		{"unknown field", strings.TrimSuffix(validOutput(), "}") + `,"command":"rm"}`, "completed", "output_text", "invalid_proposal", 200},
		{"stale", strings.ReplaceAll(validOutput(), "r001", "r007"), "completed", "output_text", "invalid_proposal", 200},
		{"duplicate", strings.ReplaceAll(validOutput(), "r002", "r001"), "completed", "output_text", "invalid_proposal", 200},
		{"unsupported", strings.ReplaceAll(validOutput(), "fixed", "shell"), "completed", "output_text", "invalid_proposal", 200},
		{"trailing", validOutput() + ` {}`, "completed", "output_text", "invalid_proposal", 200},
		{"refusal", "no", "completed", "refusal", "refusal_or_unexpected_output", 200},
		{"truncated", validOutput(), "incomplete", "output_text", "incomplete_or_unverified_response", 200},
		{"auth", "SECRET_ERROR_BODY", "", "", "authentication_failed", 401},
		{"quota", "SECRET_ERROR_BODY", "", "", "quota_or_rate_limit", 429},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "ledger")
			var calls atomic.Int32
			body := response(tt.output, tt.status, tt.kind)
			if tt.code != 200 {
				body = tt.output
			}
			c := fakeClient(tt.code, body, &calls)
			r, e := probe(context.Background(), "TEST_ONLY_CREDENTIAL", view(), dir, c)
			if r.Status != tt.want || (e == nil) != (tt.want == "valid_proposal_not_executed") {
				t.Fatalf("status=%s error=%v", r.Status, e)
			}
			if _, e = probe(context.Background(), "TEST_ONLY_CREDENTIAL", view(), dir, c); e == nil || calls.Load() != 1 {
				t.Fatal("attempt was replayed")
			}
			data, e := os.ReadFile(filepath.Join(dir, "result.json"))
			if e != nil {
				t.Fatal(e)
			}
			if strings.Contains(string(data), "SECRET_ERROR_BODY") || strings.Contains(string(data), "TEST_ONLY_CREDENTIAL") {
				t.Fatal("secret appeared in report")
			}
			if tt.code == 200 && (r.Usage == nil || r.EstimatedNanoUSD == nil || *r.EstimatedNanoUSD != 1_065_000) {
				t.Fatal("incorrect usage or cost")
			}
		})
	}
}
func TestConcurrentReservationAndTransportFailure(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "ledger")
	var calls atomic.Int32
	c := &http.Client{Transport: roundTrip(func(*http.Request) (*http.Response, error) {
		calls.Add(1)
		return nil, errors.New("TEST_ONLY_CREDENTIAL transport detail")
	})}
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			r, e := probe(context.Background(), "TEST_ONLY_CREDENTIAL", view(), dir, c)
			if e == nil || strings.Contains(e.Error(), "TEST_ONLY_CREDENTIAL") {
				t.Error("unsafe error")
			}
			if r.Attempts == 1 && r.Status != "transport_error_usage_unknown" {
				t.Errorf("unexpected %s", r.Status)
			}
		})
	}
	wg.Wait()
	if calls.Load() != 1 {
		t.Fatalf("requests=%d", calls.Load())
	}
}
func TestRequestIsBoundedWithoutTools(t *testing.T) {
	var calls atomic.Int32
	c := &http.Client{Transport: roundTrip(func(r *http.Request) (*http.Response, error) {
		calls.Add(1)
		if r.URL.String() != "https://api.openai.com/v1/responses" || r.Header.Get("Authorization") != "Bearer TEST_ONLY_CREDENTIAL" || r.GetBody != nil {
			t.Error("unexpected HTTP boundary")
		}
		data, _ := io.ReadAll(r.Body)
		var body map[string]any
		if json.Unmarshal(data, &body) != nil || len(data) > 16384 || body["store"] != false || body["service_tier"] != "default" || body["tool_choice"] != "none" || len(body["tools"].([]any)) != 0 || body["max_output_tokens"] != float64(MaxOutputTokens) {
			t.Error("unbounded request")
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(response(validOutput(), "completed", "output_text")))}, nil
	})}
	if _, e := probe(context.Background(), "TEST_ONLY_CREDENTIAL", view(), filepath.Join(t.TempDir(), "ledger"), c); e != nil {
		t.Fatal(e)
	}
	v := view()
	v.History = []investigation.Feedback{{Outcome: "untrusted instructions"}}
	if _, e := probe(context.Background(), "TEST_ONLY_CREDENTIAL", v, filepath.Join(t.TempDir(), "ledger"), c); e == nil || calls.Load() != 1 {
		t.Fatal("history leaked")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, e := probe(ctx, "TEST_ONLY_CREDENTIAL", view(), filepath.Join(t.TempDir(), "ledger"), c); e == nil || calls.Load() != 1 {
		t.Fatal("canceled request sent")
	}
}
func TestCostBoundaries(t *testing.T) {
	u := Usage{Input: 400000, Output: MaxOutputTokens, Total: 400000 + MaxOutputTokens}
	c, e := Cost(u)
	if e != nil || c*2 != ReservationNanoUSD || ReservationNanoUSD > BudgetNanoUSD {
		t.Fatal("budget exceeds cap")
	}
	u.Output++
	u.Total++
	if _, e = Cost(u); e == nil {
		t.Fatal("oversized output accepted")
	}
	u = Usage{Input: 1, Total: 1}
	u.InputDetails.Cached = 2
	if _, e = Cost(u); e == nil {
		t.Fatal("invalid cached usage accepted")
	}
}
func TestCredentialAndHistoricalProjection(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env.local")
	if e := os.WriteFile(path, []byte("# test\nOPENAI_API_KEY='TEST_ONLY_CREDENTIAL'\n"), 0600); e != nil {
		t.Fatal(e)
	}
	if k, e := LoadKey(path); e != nil || k != "TEST_ONLY_CREDENTIAL" {
		t.Fatal("test credential not loaded")
	}
	link := path + ".link"
	if e := os.Symlink(path, link); e != nil {
		t.Fatal(e)
	}
	if _, e := LoadKey(link); e == nil {
		t.Fatal("credential symlink accepted")
	}
	// The checked-in compact summary cannot replace raw, hash-verifiable captures.
	if _, e := HistoricalBaseline("../../docs/evidence/offline-loop-real-summary.json"); e == nil {
		t.Fatal("summary accepted as captured payloads")
	}
}

func TestHistoricalHashVerificationAndProjection(t *testing.T) {
	v := view()
	snapshot := toolserver.Snapshot{TrialID: v.LatestBehavior.TrialID, Kind: "baseline", ScenarioID: "discount-rounding", State: "completed", CleanupVerified: true, ProvenanceVerified: true}
	result := investigation.Result{Trace: investigation.Trace{Baseline: snapshot}, Captures: []investigation.Capture{
		{Reference: v.LatestBehavior, Evidence: toolserver.Evidence{Trial: snapshot, Baseline: &baseline.Report{BaselineReproduced: true, Observations: []baseline.Observation{{BusinessCheck: "fail", RawResponse: "UNTRUSTED_RAW_DATA"}}}}},
		{Reference: v.LatestProof, Evidence: toolserver.Evidence{Trial: snapshot, Experiment: &provenance.Experiment{}}},
	}}
	for i := range result.Captures {
		b, _ := json.Marshal(result.Captures[i].Evidence)
		h := sha256.Sum256(b)
		result.Captures[i].Reference.SHA256 = hex.EncodeToString(h[:])
	}
	path := filepath.Join(t.TempDir(), "capture.json")
	data, _ := json.Marshal(result)
	if e := os.WriteFile(path, data, 0600); e != nil {
		t.Fatal(e)
	}
	projection, e := HistoricalBaseline(path)
	if e != nil {
		t.Fatal(e)
	}
	request, e := requestBody(projection)
	if e != nil || strings.Contains(string(request), "UNTRUSTED_RAW_DATA") {
		t.Fatal("raw data passed to model")
	}
	result.Captures[0].Evidence.Baseline.Observations[0].BusinessCheck = "pass"
	data, _ = json.Marshal(result)
	if e := os.WriteFile(path, data, 0600); e != nil {
		t.Fatal(e)
	}
	if _, e = HistoricalBaseline(path); e == nil {
		t.Fatal("mutated evidence accepted")
	}
}
