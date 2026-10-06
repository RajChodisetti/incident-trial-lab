package investigation

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"net/http/httptest"

	"github.com/RajChodisetti/incident-trial-lab/internal/candidates"
	"github.com/RajChodisetti/incident-trial-lab/internal/synthetic"
)

// Evaluator-only fixtures never enter View or Selector.Select. They are held out
// from selector input, not a secret/unseen benchmark or independent incident.
//
//go:embed testdata/heldout-discount-v1.json
var heldoutDataset []byte

type heldoutCase struct {
	Name    string          `json:"name"`
	Request synthetic.Quote `json:"request"`
	Status  int             `json:"status"`
	Cents   int64           `json:"cents"`
}

type HeldOutRun struct {
	OwnerOrder    []string `json:"owner_order"`
	RequestSHA256 string   `json:"request_sha256"`
	Status        string   `json:"status"`
	Candidate     string   `json:"candidate,omitempty"`
	Passed        int      `json:"passed"`
	Failed        []string `json:"failed_cases,omitempty"`
}

type HeldOutReport struct {
	Mode             string       `json:"mode"`
	DatasetSHA256    string       `json:"dataset_sha256"`
	SeedRecordSHA256 string       `json:"seed_record_sha256"`
	Cases            int          `json:"cases"`
	Selected         int          `json:"selected"`
	AllCasesPassed   int          `json:"all_cases_passed"`
	Abstained        int          `json:"abstained"`
	Invalid          int          `json:"invalid"`
	SelectorErrors   int          `json:"selector_errors"`
	Canceled         int          `json:"canceled"`
	Runs             []HeldOutRun `json:"runs"`
}

type observedSelector struct {
	inner   Selector
	request SelectionRequest
	err     error
}

func (s *observedSelector) Select(ctx context.Context, v View, choices []string) (Decision, error) {
	// Capture exact input before a selector can mutate its owned copy.
	data, _ := json.Marshal(SelectionRequest{View: v, Choices: choices})
	_ = json.Unmarshal(data, &s.request)
	d, err := s.inner.Select(ctx, v, choices)
	s.err = err
	return d, err
}

// EvaluateHeldOut makes at most six offline calls, one for each alias assignment.
// Use only injected offline selectors: this function grants no spending authority.
// It evaluates Go handlers in process, not images, Kubernetes, budgets or receipts.
func EvaluateHeldOut(ctx context.Context, seed []byte, offline Selector) (HeldOutReport, error) {
	r := HeldOutReport{Mode: "offline_handler_controls_v1", DatasetSHA256: hashJSON(json.RawMessage(heldoutDataset))}
	if offline == nil {
		return r, errors.New("offline selector required")
	}
	record, err := DecodeSelectionRecord(seed)
	if err != nil {
		return r, err
	}
	if len(record.Request.Choices) != 3 || len(record.Request.View.History) != 0 {
		return r, errors.New("evaluation requires initial three-option context")
	}
	r.SeedRecordSHA256 = record.RecordSHA256
	var cases []heldoutCase
	if err = json.Unmarshal(heldoutDataset, &cases); err != nil || len(cases) != 18 {
		return r, errors.New("evaluation dataset invalid")
	}
	r.Cases = len(cases)
	changes, err := candidates.Changes()
	if err != nil {
		return r, err
	}
	orders := [][]string{{"fixed", "ineffective", "misleading"}, {"fixed", "misleading", "ineffective"}, {"ineffective", "fixed", "misleading"}, {"ineffective", "misleading", "fixed"}, {"misleading", "fixed", "ineffective"}, {"misleading", "ineffective", "fixed"}}
	for _, order := range orders {
		if err := ctx.Err(); err != nil {
			return r, err
		}
		s := &observedSelector{inner: offline}
		n := neutralWithOrder(s, order)
		n.changes = changes
		v := record.Request.View
		// NeutralSelector rebuilds candidate facts and copies context before calling.
		d, selectErr := n.Select(ctx, v, candidates.IDs())
		run := HeldOutRun{OwnerOrder: append([]string{}, order...), RequestSHA256: hashJSON(s.request)}
		switch {
		case ctx.Err() != nil:
			run.Status = "canceled"
			r.Canceled++
		case s.err != nil:
			run.Status = "selector_error"
			r.SelectorErrors++
		case selectErr != nil:
			run.Status = "invalid"
			r.Invalid++
		case d.Abstain:
			run.Status = "abstained"
			r.Abstained++
		default:
			run.Status = "selected"
			run.Candidate = d.Candidate
			r.Selected++
			h, err := candidates.Handler(d.Candidate)
			if err != nil {
				return r, err
			}
			for _, c := range cases {
				body, _ := json.Marshal(c.Request)
				response := httptest.NewRecorder()
				h.ServeHTTP(response, httptest.NewRequest("POST", "/quote", bytes.NewReader(body)))
				ok := response.Code == c.Status
				if c.Status == 200 {
					var total synthetic.Total
					ok = ok && json.Unmarshal(response.Body.Bytes(), &total) == nil && total.TotalCents == c.Cents
				}
				if ok {
					run.Passed++
				} else {
					run.Failed = append(run.Failed, c.Name)
				}
			}
			if run.Passed == len(cases) {
				r.AllCasesPassed++
			}
		}
		r.Runs = append(r.Runs, run)
		if ctx.Err() != nil {
			return r, ctx.Err()
		}
	}
	return r, nil
}
