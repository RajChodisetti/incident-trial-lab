package investigation

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"sync"
	"time"

	"github.com/RajChodisetti/incident-trial-lab/internal/candidates"
	scenario "github.com/RajChodisetti/incident-trial-lab/scenarios/discount-rounding"
)

type SelectionRequest struct {
	View    View     `json:"view"`
	Choices []string `json:"choices"`
}
type SelectionRecord struct {
	Version   int    `json:"version"`
	Mode      string `json:"mode"`
	CreatedAt string `json:"created_at"`
	// OwnerAliases is audit data, outside the selector-facing Request.
	OwnerAliases  map[string]string `json:"owner_aliases"`
	Request       SelectionRequest  `json:"request"`
	RequestSHA256 string            `json:"request_sha256"`
	Decision      Decision          `json:"decision"`
	Status        string            `json:"status"`
	RecordSHA256  string            `json:"record_sha256"`
}

// SelectionJournal records at most three offline fake selections in memory.
// It is not an API transport, durable decision worker or budget authorization.
type SelectionJournal struct {
	inner   Selector
	aliases map[string]string
	mu      sync.Mutex
	records []SelectionRecord
}

func NewOfflineSelectionJournal(fake Selector) (*NeutralSelector, *SelectionJournal, error) {
	j := &SelectionJournal{inner: fake}
	if fake == nil {
		return nil, nil, ErrSelection
	}
	n, err := NewNeutralSelector(j)
	if err != nil {
		return nil, nil, err
	}
	j.aliases = map[string]string{}
	for alias, id := range n.aliasToID {
		j.aliases[alias] = id
	}
	return n, j, nil
}

func hashJSON(value any) string {
	data, _ := json.Marshal(value)
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}
func recordDigest(r SelectionRecord) string { r.RecordSHA256 = ""; return hashJSON(r) }

func validateRequest(request SelectionRequest, mapping map[string]string) error {
	v := request.View
	if v.ScenarioID != "discount-rounding" || v.BusinessIncident == nil || scenario.Validate(*v.BusinessIncident) != nil || validateLatestReferences(v) != nil || !recordSignalValid(v.Signal) || v.RemainingAttempts < 1 || v.RemainingAttempts > 3 || len(v.History) > 3 || len(mapping) != 3 || len(request.Choices) < 1 || len(request.Choices) > 3 || len(v.CandidateChanges) != len(request.Choices) {
		return errors.New("selection input contract invalid")
	}
	seenIDs := map[string]bool{}
	for _, alias := range []string{"option-01", "option-02", "option-03"} {
		id := mapping[alias]
		if !candidates.ValidID(id) || seenIDs[id] {
			return errors.New("selection alias mapping invalid")
		}
		seenIDs[id] = true
	}
	changes, err := candidates.Changes()
	if err != nil {
		return err
	}
	seenAliases := map[string]bool{}
	for i, alias := range request.Choices {
		id, ok := mapping[alias]
		c := v.CandidateChanges[i]
		actual := changes[id]
		if !ok || seenAliases[alias] || c.Option != alias || c.SourceSHA256 != actual.SourceSHA256 || c.InitialLogic != actual.InitialLogic || c.Logic != actual.Logic || c.LogicSHA256 != actual.LogicSHA256 {
			return errors.New("selection code context or choices invalid")
		}
		seenAliases[alias] = true
	}
	seenHistory := map[string]bool{}
	for _, h := range v.History {
		if _, ok := mapping[h.Variant]; !ok || seenAliases[h.Variant] || seenHistory[h.Variant] {
			return errors.New("selection history invalid")
		}
		if h.Outcome != "no_improvement" && h.Outcome != "regression" && h.Outcome != "budget_or_behavior_failure" {
			return errors.New("unobserved or unsupported history outcome")
		}
		if !recordSignalValid(h.Signal) || validateLatestReferences(View{LatestBehavior: h.Behavior, LatestProof: h.Proof}) != nil {
			return errors.New("observed history evidence invalid")
		}
		seenHistory[h.Variant] = true
	}
	data, _ := json.Marshal(request)
	if len(data) > 16384 {
		return errors.New("selection request exceeds bound")
	}
	return nil
}

func recordSignalValid(s Signal) bool {
	verdict := func(value string) bool { return value == "passed" || value == "failed" || value == "inconclusive" }
	return verdict(string(s.Recovery)) && verdict(string(s.Regressions)) && s.Failed >= 0 && s.Failed <= 1000 && s.Unknown >= 0 && s.Unknown <= 1000
}

func decisionStatus(d Decision, choices []string) string {
	if d.Abstain {
		if d.Candidate == "" {
			return "abstained"
		}
		return "invalid"
	}
	for _, alias := range choices {
		if d.Candidate == alias {
			return "selected"
		}
	}
	return "invalid"
}

func (j *SelectionJournal) Select(ctx context.Context, v View, choices []string) (Decision, error) {
	if err := ctx.Err(); err != nil {
		return Decision{}, err
	}
	if j == nil || j.inner == nil {
		return Decision{}, ErrSelection
	}
	request := SelectionRequest{View: v, Choices: choices}
	if err := validateRequest(request, j.aliases); err != nil {
		return Decision{}, err
	}
	// Snapshot before invoking an untrusted selector; mutation cannot change audit.
	data, _ := json.Marshal(request)
	var snapshot SelectionRequest
	_ = json.Unmarshal(data, &snapshot)
	request = snapshot
	mapping := map[string]string{}
	for k, value := range j.aliases {
		mapping[k] = value
	}
	r := SelectionRecord{Version: 1, Mode: "offline_fake_selection", CreatedAt: time.Now().UTC().Format(time.RFC3339Nano), OwnerAliases: mapping, Request: request, RequestSHA256: hashJSON(request), Status: "pending"}
	j.mu.Lock()
	if len(j.records) >= 3 {
		j.mu.Unlock()
		return Decision{}, errors.New("offline journal attempt limit reached")
	}
	index := len(j.records)
	j.records = append(j.records, r)
	j.mu.Unlock()
	d, err := j.inner.Select(ctx, v, choices)
	r.Decision = d
	r.Status = decisionStatus(d, request.Choices)
	if r.Status == "invalid" {
		r.Decision = Decision{}
	}
	if ctx.Err() != nil {
		r.Decision = Decision{}
		r.Status = "canceled"
		err = ctx.Err()
	} else if err != nil {
		r.Decision = Decision{}
		r.Status = "selector_error"
		err = ErrSelection
	}
	r.RecordSHA256 = recordDigest(r)
	j.mu.Lock()
	j.records[index] = r
	j.mu.Unlock()
	return d, err
}

func (j *SelectionJournal) Records() []SelectionRecord {
	j.mu.Lock()
	defer j.mu.Unlock()
	data, _ := json.Marshal(j.records)
	var records []SelectionRecord
	_ = json.Unmarshal(data, &records)
	return records
}

// DecodeSelectionRecord reconstructs a bounded record under current compiled
// scenario/source contracts. It never invokes a selector, trial or apply action.
func DecodeSelectionRecord(data []byte) (SelectionRecord, error) {
	var r SelectionRecord
	if len(data) > 24576 {
		return r, errors.New("selection record exceeds bound")
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if dec.Decode(&r) != nil || dec.Decode(new(any)) != io.EOF || r.Version != 1 || r.Mode != "offline_fake_selection" || r.RecordSHA256 != recordDigest(r) || r.RequestSHA256 != hashJSON(r.Request) || validateRequest(r.Request, r.OwnerAliases) != nil {
		return SelectionRecord{}, errors.New("selection record validation failed")
	}
	if _, err := time.Parse(time.RFC3339Nano, r.CreatedAt); err != nil {
		return SelectionRecord{}, errors.New("record timestamp invalid")
	}
	if r.Status == "selected" || r.Status == "abstained" || r.Status == "invalid" {
		if r.Status != decisionStatus(r.Decision, r.Request.Choices) {
			return SelectionRecord{}, errors.New("record decision/status mismatch")
		}
	} else if r.Status != "selector_error" && r.Status != "canceled" {
		return SelectionRecord{}, errors.New("record outcome incomplete")
	} else if r.Decision != (Decision{}) {
		return SelectionRecord{}, errors.New("failed record contains a decision")
	}
	return r, nil
}

// HistoricalIncidentView verifies captured payload linkage before projecting the
// observed failed assertion. Saved eligibility never grants current trial authority.
func HistoricalIncidentView(b, p Capture) (View, error) {
	var v View
	for _, c := range []Capture{b, p} {
		if hashJSON(c.Evidence) != c.Reference.SHA256 || c.Evidence.Trial.TrialID != c.Reference.TrialID || c.Evidence.Trial.ScenarioID != "discount-rounding" || c.Evidence.Trial.Kind != "baseline" || c.Evidence.Trial.State != "completed" || !c.Evidence.Trial.CleanupVerified || !c.Evidence.Trial.ProvenanceVerified {
			return v, errors.New("historical baseline capture unverified")
		}
	}
	if b.Evidence.Baseline == nil || p.Evidence.Experiment == nil {
		return v, errors.New("historical baseline sections missing")
	}
	incident, err := scenario.Project(*b.Evidence.Baseline)
	if err != nil {
		return v, err
	}
	if p.Evidence.Experiment.Spec.BaselineFixture != incident.WorkloadSHA256 || p.Evidence.Experiment.ID != p.Evidence.Trial.ExperimentID || b.Evidence.Trial.ExperimentID != p.Evidence.Trial.ExperimentID {
		return v, errors.New("historical baseline fixture/provenance mismatch")
	}
	v = View{ScenarioID: "discount-rounding", LatestBehavior: b.Reference, LatestProof: p.Reference, Signal: Signal{Recovery: "failed", Regressions: "passed", Failed: 1}, RemainingAttempts: 1, History: []Feedback{}, BusinessIncident: &incident}
	return v, validateLatestReferences(v)
}
