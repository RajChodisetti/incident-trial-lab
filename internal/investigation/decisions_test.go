package investigation

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
)

type decisionsTransport func(*http.Request) (*http.Response, error)

func (f decisionsTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func decisionsFixture(t *testing.T) SelectionRecord {
	t.Helper()
	data, err := os.ReadFile("../../evidence/selection-context-offline.json")
	if err != nil {
		t.Fatal(err)
	}
	r, err := DecodeSelectionRecord(data)
	if err != nil {
		t.Fatal(err)
	}
	return r
}
func choiceResponse(choice string) string {
	return `{"answers":[{"type":"choice","name":"candidate_selection","choice":"` + choice + `","probabilities":[{"value":"option-01","probability":0.1},{"value":"option-02","probability":0.2},{"value":"option-03","probability":0.3},{"value":"abstain","probability":0.4}],"confidence":0.8}]}`
}
func responseBody(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}
}
func TestDecisionsHTTPContract(t *testing.T) {
	fixture := decisionsFixture(t)
	for _, choice := range []string{"option-01", "abstain"} {
		t.Run(choice, func(t *testing.T) {
			count := 0
			s, err := NewDecisionsSelector("offline-test-key", decisionsTransport(func(r *http.Request) (*http.Response, error) {
				count++
				if r.Method != "POST" || r.URL.String() != decisionsEndpoint || r.Header.Get("Authorization") != "Bearer offline-test-key" || r.Header.Get("Content-Type") != "application/json" {
					t.Fatal("HTTP contract")
				}
				if _, ok := r.Context().Deadline(); !ok { // http.Client supplies its timeout deadline.
					t.Fatal("missing deadline")
				}
				data, _ := io.ReadAll(r.Body)
				var q decisionsRequest
				if json.Unmarshal(data, &q) != nil || q.Model != "gpt-6-luna" || len(q.Questions) != 1 || q.Questions[0].Type != "choice" || q.Questions[0].Name != decisionsQuestion || len(q.Questions[0].Choices) != 4 {
					t.Fatal("request contract")
				}
				var input SelectionRequest
				if json.Unmarshal([]byte(q.Input), &input) != nil || validateNeutralRequest(input) != nil {
					t.Fatal("input contract")
				}
				if strings.Contains(string(data), "owner_aliases") || strings.Contains(string(data), `"fixed"`) || strings.Contains(string(data), "heldout") {
					t.Fatal("owner/answer leakage")
				}
				return responseBody(200, choiceResponse(choice)), nil
			}))
			if err != nil {
				t.Fatal(err)
			}
			result, err := s.Choose(context.Background(), fixture.Request.View, fixture.Request.Choices)
			if err != nil || count != 1 || len(result.RequestSHA256) != 64 || result.Confidence != 0.8 || len(result.Probabilities) != 4 {
				t.Fatalf("result %+v: %v", result, err)
			}
			if choice == "abstain" {
				if result.Decision != (Decision{Abstain: true}) {
					t.Fatal(result)
				}
			} else if result.Decision.Candidate != choice {
				t.Fatal(result)
			}
			// Chosen value need not be argmax; confidence need not equal its probability.
		})
	}
}
func TestDecisionsRejectMalformedAnswers(t *testing.T) {
	fixture := decisionsFixture(t)
	valid := choiceResponse("option-01")
	tests := map[string]string{
		"name":                 strings.Replace(valid, "candidate_selection", "unexpected", 1),
		"type":                 strings.Replace(valid, `"type":"choice"`, `"type":"boolean"`, 1),
		"unknown choice":       choiceResponse("fixed"),
		"missing distribution": strings.Replace(valid, `"probabilities":`, `"unexpected":`, 1),
		"duplicate value":      strings.Replace(valid, `"value":"option-02"`, `"value":"option-01"`, 1),
		"unknown value":        strings.Replace(valid, `"value":"option-02"`, `"value":"fixed"`, 1),
		"out of range":         strings.Replace(valid, `"probability":0.1`, `"probability":1.1`, 1),
		"sum":                  strings.Replace(valid, `"probability":0.1`, `"probability":0.11`, 1),
		"missing probability":  strings.Replace(valid, `"probability":0.1`, `"probability":null`, 1),
		"missing confidence":   strings.Replace(valid, `"confidence":0.8`, `"confidence":null`, 1),
		"confidence range":     strings.Replace(valid, `"confidence":0.8`, `"confidence":-0.1`, 1),
		"duplicate key":        strings.Replace(valid, `"confidence":0.8`, `"confidence":0.8,"confidence":0.9`, 1),
		"trailing":             valid + `{}`, "no answers": `{"answers":[]}`, "null": `null`, "oversized": strings.Repeat(" ", maxDecisionsResponse+1),
		"extra answers": `{"answers":[` + strings.TrimSuffix(strings.TrimPrefix(valid, `{"answers":[`), `]}`) + `,` + strings.TrimSuffix(strings.TrimPrefix(valid, `{"answers":[`), `]}`) + `]}`,
	}
	for name, body := range tests {
		t.Run(name, func(t *testing.T) {
			s, _ := NewDecisionsSelector("offline", decisionsTransport(func(*http.Request) (*http.Response, error) { return responseBody(200, body), nil }))
			d, err := s.Select(context.Background(), fixture.Request.View, fixture.Request.Choices)
			if !errors.Is(err, ErrSelection) || d != (Decision{}) {
				t.Fatalf("%+v %v", d, err)
			}
		})
	}
}
func TestDecisionsRefusesInvalidInputBeforeTransport(t *testing.T) {
	for _, change := range []func(*SelectionRequest){
		func(r *SelectionRequest) { r.Choices[0] = "fixed" },
		func(r *SelectionRequest) { r.View.BusinessIncident = nil },
		func(r *SelectionRequest) { r.View.CandidateChanges[0].Logic = "arbitrary code" },
		func(r *SelectionRequest) { r.View.RemainingAttempts = 4 },
		func(r *SelectionRequest) { r.View.LatestProof.SHA256 = "bad" },
		func(r *SelectionRequest) { r.View.History = []Feedback{{Variant: r.Choices[0]}} },
	} {
		fixture := decisionsFixture(t)
		change(&fixture.Request)
		calls := 0
		s, _ := NewDecisionsSelector("offline", decisionsTransport(func(*http.Request) (*http.Response, error) { calls++; return nil, errors.New("must not call") }))
		_, err := s.Select(context.Background(), fixture.Request.View, fixture.Request.Choices)
		if err == nil || calls != 0 {
			t.Fatal("invalid input sent")
		}
	}
}
func TestDecisionsTransportFailureCancellationAndRedirect(t *testing.T) {
	fixture := decisionsFixture(t)
	for _, status := range []int{301, 401, 429, 500} {
		calls := 0
		s, _ := NewDecisionsSelector("offline", decisionsTransport(func(*http.Request) (*http.Response, error) {
			calls++
			r := responseBody(status, "private provider details")
			r.Header.Set("Location", "https://other.invalid")
			return r, nil
		}))
		_, err := s.Select(context.Background(), fixture.Request.View, fixture.Request.Choices)
		if err != ErrSelection || calls != 1 {
			t.Fatal("status retried or followed")
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	calls := 0
	s, _ := NewDecisionsSelector("offline", decisionsTransport(func(*http.Request) (*http.Response, error) { calls++; return nil, errors.New("private details") }))
	if _, err := s.Select(ctx, fixture.Request.View, fixture.Request.Choices); err != context.Canceled || calls != 0 {
		t.Fatal("canceled request sent")
	}
	if _, err := s.Select(context.Background(), fixture.Request.View, fixture.Request.Choices); err != ErrSelection || strings.Contains(err.Error(), "private") {
		t.Fatal("transport details leaked")
	}
	ctx, cancel = context.WithCancel(context.Background())
	defer cancel()
	s, _ = NewDecisionsSelector("offline", decisionsTransport(func(r *http.Request) (*http.Response, error) { cancel(); return nil, r.Context().Err() }))
	if _, err := s.Select(ctx, fixture.Request.View, fixture.Request.Choices); err != context.Canceled {
		t.Fatal(err)
	}
}
func TestDecisionsMapsThroughNeutralProposer(t *testing.T) {
	fixture := decisionsFixture(t)
	for _, abstain := range []bool{false, true} {
		s, _ := NewDecisionsSelector("offline", decisionsTransport(func(r *http.Request) (*http.Response, error) {
			data, _ := io.ReadAll(r.Body)
			var q decisionsRequest
			_ = json.Unmarshal(data, &q)
			choice := q.Questions[0].Choices[0].Value
			if abstain {
				choice = "abstain"
			}
			return responseBody(200, choiceResponse(choice)), nil
		}))
		neutral, err := NewNeutralSelector(s)
		if err != nil {
			t.Fatal(err)
		}
		view := fixture.Request.View
		view.CandidateChanges = nil
		view.History = nil
		proposal, err := (SelectorProposer{Selector: neutral}).Propose(context.Background(), view)
		if abstain {
			if err != ErrAbstained {
				t.Fatal(err)
			}
		} else if err != nil || proposal.Variant == "" || len(proposal.References) != 2 {
			t.Fatalf("%+v %v", proposal, err)
		}
	}
}
