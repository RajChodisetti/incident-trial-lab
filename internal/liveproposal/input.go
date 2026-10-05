package liveproposal

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"strings"

	"github.com/RajChodisetti/incident-trial-lab/internal/investigation"
)

// LoadKey reads only the approved local file into memory. It does not export,
// copy, execute or log env contents. All errors deliberately omit file contents.
func LoadKey(path string) (string, error) {
	s, e := os.Lstat(path)
	if e != nil || !s.Mode().IsRegular() || s.Size() > 16384 {
		return "", errors.New("approved credential file unavailable")
	}
	data, e := os.ReadFile(path)
	if e != nil {
		return "", errors.New("approved credential file unreadable")
	}
	key := ""
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "OPENAI_API_KEY=") {
			if key != "" {
				return "", errors.New("ambiguous credential entry")
			}
			key = strings.TrimSpace(strings.TrimPrefix(line, "OPENAI_API_KEY="))
			if len(key) >= 2 && (key[0] == '\'' && key[len(key)-1] == '\'' || key[0] == '"' && key[len(key)-1] == '"') {
				key = key[1 : len(key)-1]
			}
		}
	}
	if key == "" || strings.ContainsAny(key, " \t\r\n") {
		return "", errors.New("credential missing or malformed")
	}
	return key, nil
}

// HistoricalBaseline verifies the two captured payload hashes, then projects only
// counters and reference metadata. Raw observations never become API input.
func HistoricalBaseline(path string) (investigation.View, error) {
	var v investigation.View
	s, e := os.Lstat(path)
	if e != nil || !s.Mode().IsRegular() || s.Size() > 2<<20 {
		return v, errors.New("synthetic capture unavailable")
	}
	data, e := os.ReadFile(path)
	if e != nil {
		return v, errors.New("synthetic capture unreadable")
	}
	var result investigation.Result
	if json.Unmarshal(data, &result) != nil || len(result.Captures) < 2 {
		return v, errors.New("invalid synthetic capture")
	}
	b, p := result.Captures[0], result.Captures[1]
	for _, c := range []investigation.Capture{b, p} {
		payload, e := json.Marshal(c.Evidence)
		h := sha256.Sum256(payload)
		if e != nil || hex.EncodeToString(h[:]) != c.Reference.SHA256 {
			return v, errors.New("capture hash mismatch")
		}
	}
	if b.Evidence.Baseline == nil || !b.Evidence.Baseline.BaselineReproduced || b.Evidence.Baseline.FixtureError != "" || b.Evidence.Baseline.HealthError != "" || p.Evidence.Experiment == nil {
		return v, errors.New("reproduced historical baseline and provenance required")
	}
	for _, c := range []investigation.Capture{b, p} {
		if c.Reference.TrialID != result.Trace.Baseline.TrialID || c.Evidence.Trial.TrialID != c.Reference.TrialID || c.Evidence.Trial.Kind != "baseline" || c.Evidence.Trial.State != "completed" || !c.Evidence.Trial.CleanupVerified || !c.Evidence.Trial.ProvenanceVerified {
			return v, errors.New("historical capture identity mismatch")
		}
	}
	v = investigation.View{ScenarioID: result.Trace.Baseline.ScenarioID, LatestBehavior: b.Reference, LatestProof: p.Reference, RemainingAttempts: 1, History: []investigation.Feedback{}}
	v.Signal.Recovery, v.Signal.Regressions = "failed", "passed"
	for _, o := range b.Evidence.Baseline.Observations {
		if o.BusinessCheck == "fail" {
			v.Signal.Failed++
		}
		if o.Error != "" {
			v.Signal.Unknown++
		}
	}
	return v, validateView(v)
}
