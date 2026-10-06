// Package provenance binds a local experiment to evidence; it is not signed attestation.
package provenance

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/RajChodisetti/incident-trial-lab/internal/artifact"
	"github.com/RajChodisetti/incident-trial-lab/internal/baseline"
	"github.com/RajChodisetti/incident-trial-lab/internal/evaluation"
	"github.com/RajChodisetti/incident-trial-lab/internal/lifecycle"
	"github.com/RajChodisetti/incident-trial-lab/internal/trial"
)

var idPattern = regexp.MustCompile(`^[0-9a-f]{16}$`)

const Lifetime = 30 * time.Minute

type Spec struct {
	CandidateLock      string            `json:"candidate_lock_sha256"`
	BaselineFixture    string            `json:"baseline_fixture_sha256"`
	BaselineAssertions string            `json:"baseline_assertions_sha256"`
	Workload           string            `json:"workload_sha256"`
	Assertions         string            `json:"assertions_sha256"`
	Contract           string            `json:"contract_sha256"`
	Manifest           string            `json:"deployment_sha256"`
	BaselineImage      string            `json:"baseline_image_digest"`
	Candidates         map[string]string `json:"candidate_image_digests"`
}
type Experiment struct {
	ID                   string    `json:"experiment_id"`
	CreatedAt            time.Time `json:"created_at"`
	ExpiresAt            time.Time `json:"expires_at"`
	BaselineRunID        string    `json:"baseline_run_id"`
	BaselineFinishedAt   time.Time `json:"baseline_finished_at"`
	BaselineEvidenceHash string    `json:"baseline_evidence_sha256"`
	Spec                 Spec      `json:"spec"`
}
type Attempt struct {
	ID           string    `json:"attempt_id"`
	ExperimentID string    `json:"experiment_id"`
	Variant      string    `json:"variant"`
	CreatedAt    time.Time `json:"created_at"`
}
type Receipt struct {
	ExperimentID          string             `json:"experiment_id"`
	AttemptID             string             `json:"attempt_id"`
	BaselineRunID         string             `json:"baseline_run_id"`
	CandidateRunID        string             `json:"candidate_run_id"`
	Variant               string             `json:"variant"`
	Verdict               evaluation.Verdict `json:"verdict"`
	BaselineEvidenceHash  string             `json:"baseline_evidence_sha256"`
	CandidateEvidenceHash string             `json:"candidate_evidence_sha256"`
	Spec                  Spec               `json:"spec"`
	SourceBinding         *artifact.Binding  `json:"source_binding,omitempty"`
}

func hash(data []byte) string { return fmt.Sprintf("%x", sha256.Sum256(data)) }
func newID() (string, error) {
	b := make([]byte, 8)
	_, err := rand.Read(b)
	return hex.EncodeToString(b), err
}
func read(root, relative string) ([]byte, error) {
	path := filepath.Join(root, relative)
	for p := path; p != filepath.Clean(root); p = filepath.Dir(p) {
		s, err := os.Lstat(p)
		if err != nil {
			return nil, err
		}
		if s.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("symlink evidence refused")
		}
		if filepath.Dir(p) == p {
			return nil, fmt.Errorf("path outside project")
		}
	}
	return os.ReadFile(path)
}
func writeExclusive(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("evidence already bound or unavailable: %w", err)
	}
	defer f.Close()
	if _, err = f.Write(append(data, '\n')); err != nil {
		return err
	}
	if err = f.Sync(); err != nil {
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	d, err := os.Open(filepath.Dir(path))
	if err != nil {
		return err
	}
	defer d.Close()
	return d.Sync()
}
func runDir(id string) (string, error) {
	if !idPattern.MatchString(id) {
		return "", fmt.Errorf("invalid run ID")
	}
	return filepath.Join("evidence", "lifecycle-"+id), nil
}
func experimentDir(id string) (string, error) {
	if !idPattern.MatchString(id) {
		return "", fmt.Errorf("invalid experiment ID")
	}
	return filepath.Join("evidence", "experiment-"+id), nil
}

func CurrentSpec(root string) (Spec, error) {
	var s Spec
	files := []string{"scenarios/discount-rounding/workload.json", "scenarios/discount-rounding/comparison-workload.json", "scenarios/discount-rounding/comparison-contract.md", "deploy/local/baseline.json", "deploy/local/candidates.json", "scenarios/discount-rounding/contract.md", "deploy/local/kind.yaml"}
	data := make([][]byte, len(files))
	for i, path := range files {
		b, err := read(root, path)
		if err != nil {
			return s, err
		}
		data[i] = b
	}
	bc, err := baseline.Load(data[0])
	if err != nil {
		return s, err
	}
	cc, err := trial.Load(data[1])
	if err != nil {
		return s, err
	}
	for _, b := range bc {
		found := false
		for _, c := range cc {
			role := "regression"
			if b.Faulty != nil {
				role = "recovery"
			}
			if c.Name == b.Name && c.Role == role && c.Request == b.Request && c.Expected == b.Expected {
				found = true
			}
		}
		if !found {
			return s, fmt.Errorf("comparison assertions no longer cover baseline case %q", b.Name)
		}
	}
	baselineAssertions, _ := json.Marshal(bc)
	type assertion struct {
		Name, Role string
		Expected   int64
	}
	var assertions []assertion
	for _, c := range cc {
		assertions = append(assertions, assertion{c.Name, c.Role, c.Expected})
	}
	a, _ := json.Marshal(assertions)
	var manifest struct {
		Items []struct {
			Kind string
			Spec struct {
				Template struct {
					Spec struct{ Containers []struct{ Image string } }
				}
			}
		}
	}
	if err = json.Unmarshal(data[3], &manifest); err != nil {
		return s, err
	}
	for _, item := range manifest.Items {
		if item.Kind == "Deployment" && len(item.Spec.Template.Spec.Containers) == 1 {
			image := item.Spec.Template.Spec.Containers[0].Image
			parts := strings.Split(image, "@")
			if len(parts) == 2 {
				s.BaselineImage = parts[1]
			}
		}
	}
	if !regexp.MustCompile(`^sha256:[0-9a-f]{64}$`).MatchString(s.BaselineImage) {
		return s, fmt.Errorf("baseline is not pinned")
	}
	if err = json.Unmarshal(data[4], &s.Candidates); err != nil {
		return s, err
	}
	if len(s.Candidates) != 3 {
		return s, fmt.Errorf("require exactly three registered candidates")
	}
	for _, v := range []string{"fixed", "ineffective", "misleading"} {
		if !regexp.MustCompile(`^sha256:[0-9a-f]{64}$`).MatchString(s.Candidates[v]) {
			return s, fmt.Errorf("candidate is not pinned")
		}
	}
	s.CandidateLock = hash(data[4])
	s.Manifest = hash(data[3])
	s.BaselineFixture = hash(data[0])
	s.Workload = hash(data[1])
	s.Contract = hash(append(append(append([]byte{}, data[2]...), data[5]...), data[6]...))
	s.BaselineAssertions = hash(baselineAssertions)
	s.Assertions = hash(a)
	return s, nil
}

func bundle(root, id, phase string) (string, lifecycle.Record, error) {
	var r lifecycle.Record
	dir, err := runDir(id)
	if err != nil {
		return "", r, err
	}
	b, err := read(root, filepath.Join(dir, "record.json"))
	if err != nil {
		return "", r, err
	}
	if err = json.Unmarshal(b, &r); err != nil {
		return "", r, err
	}
	if r.Version != 1 || r.State != "finished" || r.RunID != id || r.Mode != phase || r.Result == nil || r.Result.Outcome != lifecycle.Completed || !r.Result.Cleanup.Complete || r.Result.ExitCode != 0 || r.Result.Error != "" || r.Result.ForcedKill {
		return "", r, fmt.Errorf("run is not completed with verified cleanup")
	}
	matches, err := filepath.Glob(filepath.Join(root, dir, phase+"-*"))
	if err != nil || len(matches) != 1 {
		return "", r, fmt.Errorf("require one phase evidence bundle")
	}
	relative, err := filepath.Rel(root, matches[0])
	if err == nil {
		_, err = read(root, filepath.Join(relative, "workload.json"))
	}
	return relative, r, err
}

func baselineEvidence(root, id string, s Spec) (string, time.Time, string, error) {
	dir, r, err := bundle(root, id, "baseline")
	if err != nil {
		return "", time.Time{}, "", err
	}
	ended, err := time.Parse(time.RFC3339Nano, r.Result.FinishedAt)
	if err != nil {
		return "", ended, "", err
	}
	started, startErr := time.Parse(time.RFC3339Nano, r.Result.StartedAt)
	if startErr != nil || ended.Before(started) {
		return "", ended, "", fmt.Errorf("invalid baseline timestamps")
	}
	report, err := read(root, filepath.Join(dir, "check.json"))
	if err != nil {
		return "", ended, "", err
	}
	fixture, err := read(root, filepath.Join(dir, "workload.json"))
	if err != nil || hash(fixture) != s.BaselineFixture {
		return "", ended, "", fmt.Errorf("baseline fixture mismatch")
	}
	image, err := read(root, filepath.Join(dir, "image-digest.txt"))
	if err != nil || strings.TrimSpace(string(image)) != s.BaselineImage {
		return "", ended, "", fmt.Errorf("baseline image mismatch")
	}
	if err = verifyDeployment(root, dir, s, ""); err != nil {
		return "", ended, "", err
	}
	runtimeHash, err := verifyRuntime(root, dir, s.BaselineImage)
	if err != nil {
		return "", ended, "", err
	}
	cases, err := baseline.Load(fixture)
	if err != nil {
		return "", ended, "", err
	}
	var proof struct {
		FixtureHash string `json:"fixture_sha256"`
		baseline.Report
	}
	if err = json.Unmarshal(report, &proof); err != nil {
		return "", ended, "", err
	}
	if !proof.BaselineReproduced || proof.FixtureHash != s.BaselineFixture || proof.HealthError != "" || proof.FixtureError != "" || len(proof.Observations) != len(cases) {
		return "", ended, "", fmt.Errorf("baseline proof incomplete")
	}
	for i, c := range cases {
		o := proof.Observations[i]
		required := c.Expected
		status := "pass"
		if c.Faulty != nil {
			required = *c.Faulty
			status = "fail"
		}
		if err := verifyBaselineResponse(o.RawResponse, required); err != nil {
			return "", ended, "", err
		}
		if o.Name != c.Name || o.Expected != c.Expected || o.Observed == nil || *o.Observed != required || o.BusinessCheck != status || o.HTTPStatus != 200 || o.Error != "" {
			return "", ended, "", fmt.Errorf("baseline observations contradict proof")
		}
	}
	return dir, ended, hash([]byte(hash(report) + hash(fixture) + hash(image) + runtimeHash + hashMustJSON(r))), nil
}

// Prepare only accepts a fresh, semantically verified baseline once.
func Prepare(root, baselineID string, now time.Time) (Experiment, error) {
	var e Experiment
	s, err := CurrentSpec(root)
	if err != nil {
		return e, err
	}
	_, err = runDir(baselineID)
	if err != nil {
		return e, err
	}
	validatedDir, err := lifecycle.RecordDir(root, baselineID)
	if err != nil {
		return e, err
	}
	lock, err := lifecycle.LockRecord(validatedDir)
	if err != nil {
		return e, err
	}
	defer lock.Close()
	dir, ended, evidenceHash, err := baselineEvidence(root, baselineID, s)
	if err != nil {
		return e, err
	}
	if ended.After(now) || now.Sub(ended) > 10*time.Minute {
		return e, fmt.Errorf("baseline is stale or future-dated")
	}
	if _, err = os.Lstat(filepath.Join(root, dir, "experiment-binding.json")); !os.IsNotExist(err) {
		return e, fmt.Errorf("baseline already bound or unavailable")
	}
	id, err := newID()
	if err != nil {
		return e, err
	}
	e = Experiment{id, now, now.Add(Lifetime), baselineID, ended, evidenceHash, s}
	expDir, _ := experimentDir(id)
	if err = os.Mkdir(filepath.Join(root, expDir), 0700); err != nil {
		return e, err
	}
	if err = writeExclusive(filepath.Join(root, dir, "experiment-binding.json"), e); err != nil {
		return e, err
	}
	if err = writeExclusive(filepath.Join(root, expDir, "experiment.json"), e); err != nil {
		return e, err
	}
	return e, nil
}

func Load(root, id string, now time.Time) (Experiment, error) {
	var e Experiment
	dir, err := experimentDir(id)
	if err != nil {
		return e, err
	}
	b, err := read(root, filepath.Join(dir, "experiment.json"))
	if err != nil {
		return e, err
	}
	if err = json.Unmarshal(b, &e); err != nil {
		return e, err
	}
	if e.ID != id || now.Before(e.CreatedAt) || !now.Before(e.ExpiresAt) || e.ExpiresAt.Sub(e.CreatedAt) != Lifetime {
		return e, fmt.Errorf("experiment expired, future-dated, or mismatched")
	}
	current, err := CurrentSpec(root)
	if err != nil {
		return e, err
	}
	a, _ := json.Marshal(current)
	b, _ = json.Marshal(e.Spec)
	if string(a) != string(b) {
		return e, fmt.Errorf("scenario/workload/assertions/image identity changed")
	}
	baselineDir, ended, h, err := baselineEvidence(root, e.BaselineRunID, e.Spec)
	if err != nil || !ended.Equal(e.BaselineFinishedAt) || h != e.BaselineEvidenceHash {
		return e, fmt.Errorf("bound baseline evidence changed: %v", err)
	}
	marker, err := read(root, filepath.Join(baselineDir, "experiment-binding.json"))
	var bound Experiment
	if err != nil || json.Unmarshal(marker, &bound) != nil || hashMustJSON(bound) != hashMustJSON(e) {
		return e, fmt.Errorf("baseline binding mismatch")
	}
	return e, nil
}

func Reserve(root, id, variant string, now time.Time) (Attempt, error) {
	e, err := Load(root, id, now)
	if err != nil {
		return Attempt{}, err
	}
	if _, ok := e.Spec.Candidates[variant]; !ok {
		return Attempt{}, fmt.Errorf("candidate not registered")
	}
	nonce, err := newID()
	if err != nil {
		return Attempt{}, err
	}
	a := Attempt{nonce, id, variant, now}
	dir, _ := experimentDir(id)
	return a, writeExclusive(filepath.Join(root, dir, "attempt-"+nonce+".json"), a)
}

// Bind consumes a new candidate run/attempt exactly once; no arbitrary boolean proof.
func Bind(root string, a Attempt, candidateID string, now time.Time) (Receipt, error) {
	return validateAndBind(root, a, candidateID, now, true)
}

func validateAndBind(root string, a Attempt, candidateID string, now time.Time, consume bool) (Receipt, error) {
	var receipt Receipt
	if !idPattern.MatchString(a.ID) {
		return receipt, fmt.Errorf("invalid attempt ID")
	}
	e, err := Load(root, a.ExperimentID, now)
	if err != nil {
		return receipt, err
	}
	expDir, _ := experimentDir(e.ID)
	lock, err := lifecycle.LockRecord(filepath.Join(root, expDir))
	if err != nil {
		return receipt, err
	}
	defer lock.Close()
	if _, err = os.Lstat(filepath.Join(root, expDir, "receipt-"+a.ID+".json")); consume && !os.IsNotExist(err) {
		return receipt, fmt.Errorf("attempt already consumed or unavailable")
	}
	runRoot, err := runDir(candidateID)
	if err != nil {
		return receipt, err
	}
	validatedDir, err := lifecycle.RecordDir(root, candidateID)
	if err != nil {
		return receipt, err
	}
	runLock, err := lifecycle.LockRecord(validatedDir)
	if err != nil {
		return receipt, err
	}
	defer runLock.Close()
	stored, err := read(root, filepath.Join(expDir, "attempt-"+a.ID+".json"))
	if err != nil {
		return receipt, err
	}
	var registered Attempt
	if json.Unmarshal(stored, &registered) != nil || registered != a || !idPattern.MatchString(a.ID) {
		return receipt, fmt.Errorf("unregistered/mismatched attempt")
	}
	dir, r, err := bundle(root, candidateID, "candidate")
	if err != nil {
		return receipt, err
	}
	started, err := time.Parse(time.RFC3339Nano, r.Result.StartedAt)
	if err != nil || started.Before(a.CreatedAt) || a.CreatedAt.Before(e.CreatedAt) {
		return receipt, fmt.Errorf("candidate predates reserved attempt")
	}
	ended, err := time.Parse(time.RFC3339Nano, r.Result.FinishedAt)
	if err != nil || ended.Before(started) || ended.After(now) {
		return receipt, fmt.Errorf("invalid candidate timestamps")
	}
	workload, err := read(root, filepath.Join(dir, "workload.json"))
	if err != nil || hash(workload) != e.Spec.Workload {
		return receipt, fmt.Errorf("candidate workload mismatch")
	}
	image, err := read(root, filepath.Join(dir, "image-digest.txt"))
	if err != nil || strings.TrimSpace(string(image)) != e.Spec.Candidates[a.Variant] {
		return receipt, fmt.Errorf("candidate image mismatch")
	}
	if err = verifyDeployment(root, dir, e.Spec, a.Variant); err != nil {
		return receipt, err
	}
	runtimeHash, err := verifyRuntime(root, dir, e.Spec.Candidates[a.Variant])
	if err != nil {
		return receipt, err
	}
	sourceBinding, err := VerifySourceBinding(root, dir, a.Variant, e.Spec.Candidates[a.Variant])
	if err != nil {
		return receipt, err
	}
	proof, err := read(root, filepath.Join(dir, "baseline-proof.json"))
	baselineDir, _, _, _ := baselineEvidence(root, e.BaselineRunID, e.Spec)
	original, originalErr := read(root, filepath.Join(baselineDir, "check.json"))
	if err != nil || originalErr != nil || hash(proof) != hash(original) {
		return receipt, fmt.Errorf("candidate used different baseline evidence")
	}
	behaviorBytes, err := read(root, filepath.Join(dir, "behavior.json"))
	if err != nil {
		return receipt, err
	}
	var behavior struct {
		FixtureHash string `json:"fixture_sha256"`
		trial.Report
	}
	if json.Unmarshal(behaviorBytes, &behavior) != nil || behavior.FixtureHash != e.Spec.Workload {
		return receipt, fmt.Errorf("candidate behavior fixture mismatch")
	}
	bs, bse := time.Parse(time.RFC3339Nano, behavior.StartedAt)
	be, bee := time.Parse(time.RFC3339Nano, behavior.FinishedAt)
	if bse != nil || bee != nil || bs.Before(started) || be.After(ended) || be.Before(bs) {
		return receipt, fmt.Errorf("behavior timestamps outside run")
	}
	cases, _ := trial.Load(workload)
	if err = ValidateObservations(behavior.Report, cases); err != nil {
		return receipt, err
	}
	resultBytes, err := read(root, filepath.Join(dir, "result.json"))
	if err != nil {
		return receipt, err
	}
	var result struct {
		Verdict          evaluation.Verdict
		BaselineVerified bool         `json:"baseline_verified"`
		Budget           trial.Budget `json:"resource_budget"`
		Behavior         trial.Report `json:"behavior"`
	}
	if json.Unmarshal(resultBytes, &result) != nil || !result.BaselineVerified {
		return receipt, fmt.Errorf("candidate result incomplete")
	}
	b1, _ := json.Marshal(result.Behavior)
	b2, _ := json.Marshal(behavior.Report)
	if string(b1) != string(b2) {
		return receipt, fmt.Errorf("result behavior differs from observations")
	}
	beforeBytes, err := read(root, filepath.Join(dir, "resources-before.json"))
	if err != nil {
		return receipt, err
	}
	afterBytes, err := read(root, filepath.Join(dir, "resources-after.json"))
	if err != nil {
		return receipt, err
	}
	before, e1 := trial.ParseSample(beforeBytes)
	after, e2 := trial.ParseSample(afterBytes)
	if e1 != nil || e2 != nil {
		return receipt, fmt.Errorf("resource evidence cannot be verified")
	}
	podUID, err := runtimePodUID(root, dir)
	if err != nil || before.PodUID != podUID || after.PodUID != podUID {
		return receipt, fmt.Errorf("resource samples belong to different pod")
	}
	budget := trial.AssessBudget(before, after)
	if err = trial.CoversWorkload(before, after, behavior.StartedAt, behavior.FinishedAt); err != nil {
		budget.Status = evaluation.Inconclusive
		budget.Error = err.Error()
	}
	b1, _ = json.Marshal(budget)
	b2, _ = json.Marshal(result.Budget)
	if string(b1) != string(b2) || trial.Decide(true, behavior.BehavioralVerdict, budget) != result.Verdict {
		return receipt, fmt.Errorf("result contradicts recomputed budget/verdict")
	}
	receipt = Receipt{e.ID, a.ID, e.BaselineRunID, candidateID, a.Variant, result.Verdict, e.BaselineEvidenceHash, hash([]byte(hash(behaviorBytes) + hash(resultBytes) + hash(image) + hash(workload) + hash(proof) + hash(beforeBytes) + hash(afterBytes) + runtimeHash + hashMustJSON(r) + hashMustJSON(sourceBinding))), e.Spec, &sourceBinding}
	if !consume {
		stored, err := read(root, filepath.Join(expDir, "receipt-"+a.ID+".json"))
		var saved Receipt
		if err != nil || json.Unmarshal(stored, &saved) != nil || hashMustJSON(saved) != hashMustJSON(receipt) {
			return Receipt{}, fmt.Errorf("stored receipt no longer matches evidence")
		}
		marker, err := read(root, filepath.Join(runRoot, "provenance-binding.json"))
		var bound Receipt
		if err != nil || json.Unmarshal(marker, &bound) != nil || hashMustJSON(bound) != hashMustJSON(receipt) {
			return Receipt{}, fmt.Errorf("candidate binding marker mismatch")
		}
		return receipt, nil
	}
	if err = writeExclusive(filepath.Join(root, runRoot, "provenance-binding.json"), receipt); err != nil {
		return receipt, err
	}
	if err = writeExclusive(filepath.Join(root, expDir, "receipt-"+a.ID+".json"), receipt); err != nil {
		return receipt, err
	}
	return receipt, nil
}

func BaselineProofPath(root string, e Experiment) (string, error) {
	dir, _, _, err := baselineEvidence(root, e.BaselineRunID, e.Spec)
	return filepath.Join(root, dir, "check.json"), err
}

// LoadAttempt accepts only an ID within the selected experiment, never a path.
func LoadAttempt(root, experimentID, attemptID string) (Attempt, error) {
	var a Attempt
	if !idPattern.MatchString(attemptID) {
		return a, fmt.Errorf("invalid attempt ID")
	}
	dir, err := experimentDir(experimentID)
	if err != nil {
		return a, err
	}
	b, err := read(root, filepath.Join(dir, "attempt-"+attemptID+".json"))
	if err != nil {
		return a, err
	}
	if json.Unmarshal(b, &a) != nil || a.ID != attemptID || a.ExperimentID != experimentID {
		return a, fmt.Errorf("attempt identity mismatch")
	}
	return a, nil
}

// Verify revalidates a live receipt against the evidence; it does not consume it
// again or accept an arbitrary candidate run selected by the caller.
func Verify(root, experimentID, attemptID string, now time.Time) (Receipt, error) {
	a, err := LoadAttempt(root, experimentID, attemptID)
	if err != nil {
		return Receipt{}, err
	}
	dir, _ := experimentDir(experimentID)
	b, err := read(root, filepath.Join(dir, "receipt-"+attemptID+".json"))
	if err != nil {
		return Receipt{}, err
	}
	var receipt Receipt
	if json.Unmarshal(b, &receipt) != nil || receipt.ExperimentID != experimentID || receipt.AttemptID != attemptID {
		return Receipt{}, fmt.Errorf("receipt identity mismatch")
	}
	return validateAndBind(root, a, receipt.CandidateRunID, now, false)
}
