package provenance

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/RajChodisetti/incident-trial-lab/internal/baseline"
	"github.com/RajChodisetti/incident-trial-lab/internal/evaluation"
	"github.com/RajChodisetti/incident-trial-lab/internal/lifecycle"
	"github.com/RajChodisetti/incident-trial-lab/internal/trial"
)

const baselineID = "1111111111111111"
const candidateID = "2222222222222222"

type fixture struct {
	root, bd, cd string
	now          time.Time
	e            Experiment
	a            Attempt
	cases        []trial.Case
	r            trial.Report
	manifest     []byte
	digest       string
}

func put(t *testing.T, root, path string, v any) {
	t.Helper()
	var b []byte
	switch x := v.(type) {
	case []byte:
		b = x
	case string:
		b = []byte(x)
	default:
		var err error
		b, err = json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
	}
	full := filepath.Join(root, path)
	if err := os.MkdirAll(filepath.Dir(full), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, b, 0600); err != nil {
		t.Fatal(err)
	}
}
func runtimeFiles(t *testing.T, f fixture, dir string) {
	put(t, f.root, filepath.Join(dir, "image-manifest.json"), f.manifest)
	put(t, f.root, filepath.Join(dir, "runtime-image.json"), f.manifest)
	put(t, f.root, filepath.Join(dir, "image-digest.txt"), f.digest+"\n")
	put(t, f.root, filepath.Join(dir, "pods.json"), fmt.Sprintf(`{"items":[{"metadata":{"uid":"pod-unit"},"spec":{"containers":[{"name":"service","image":"test.invalid/service@%s"}]},"status":{"containerStatuses":[{"name":"service","image":"sha256:%s","imageID":"test.invalid/service@%s","ready":true}]}}]}`, f.digest, strings.Repeat("a", 64), f.digest))
}
func newFixture(t *testing.T) fixture {
	t.Helper()
	f := fixture{root: t.TempDir(), now: time.Now().UTC(), bd: "evidence/lifecycle-" + baselineID + "/baseline-unit", cd: "evidence/lifecycle-" + candidateID + "/candidate-fixed-unit"}
	f.manifest = []byte(fmt.Sprintf(`{"config":{"digest":"sha256:%s"}}`, strings.Repeat("a", 64)))
	f.digest = "sha256:" + hash(f.manifest)
	for _, p := range []string{"scenarios/discount-rounding/workload.json", "scenarios/discount-rounding/comparison-workload.json", "scenarios/discount-rounding/contract.md", "scenarios/discount-rounding/comparison-contract.md", "deploy/local/kind.yaml", "deploy/local/baseline.json"} {
		b, err := os.ReadFile(filepath.Join("../..", p))
		if err != nil {
			t.Fatal(err)
		}
		if p == "deploy/local/baseline.json" {
			s := strings.Split(string(b), "@")
			if len(s) != 2 {
				t.Fatal("test manifest needs one image")
			}
			old := strings.Split(s[1], `"`)[0]
			b = []byte(strings.ReplaceAll(string(b), old, f.digest))
		}
		put(t, f.root, p, b)
	}
	put(t, f.root, "deploy/local/candidates.json", map[string]string{"fixed": f.digest, "ineffective": f.digest, "misleading": f.digest})
	bf, _ := read(f.root, "scenarios/discount-rounding/workload.json")
	bc, _ := baseline.Load(bf)
	report := baseline.Report{BaselineReproduced: true}
	for _, c := range bc {
		v := c.Expected
		status := "pass"
		if c.Faulty != nil {
			v = *c.Faulty
			status = "fail"
		}
		report.Observations = append(report.Observations, baseline.Observation{Name: c.Name, Expected: c.Expected, Observed: &v, HTTPStatus: 200, RawResponse: fmt.Sprintf(`{"total_cents":%d}`, v), BusinessCheck: status})
	}
	put(t, f.root, filepath.Join(f.bd, "check.json"), struct {
		FixtureHash string `json:"fixture_sha256"`
		baseline.Report
	}{hash(bf), report})
	put(t, f.root, filepath.Join(f.bd, "workload.json"), bf)
	runtimeFiles(t, f, f.bd)
	deployment, _ := read(f.root, "deploy/local/baseline.json")
	put(t, f.root, filepath.Join(f.bd, "baseline.json"), deployment)
	putRecord(t, f, baselineID, "baseline", f.now.Add(-2*time.Minute), f.now.Add(-time.Minute), lifecycle.Completed)
	var err error
	f.e, err = Prepare(f.root, baselineID, f.now)
	if err != nil {
		t.Fatal(err)
	}
	f.a, err = Reserve(f.root, f.e.ID, "fixed", f.now)
	if err != nil {
		t.Fatal(err)
	}
	wf, _ := read(f.root, "scenarios/discount-rounding/comparison-workload.json")
	f.cases, _ = trial.Load(wf)
	put(t, f.root, filepath.Join(f.cd, "workload.json"), wf)
	proof, _ := read(f.root, filepath.Join(f.bd, "check.json"))
	put(t, f.root, filepath.Join(f.cd, "baseline-proof.json"), proof)
	runtimeFiles(t, f, f.cd)
	candidateManifest := strings.ReplaceAll(string(deployment), "incident-trial-lab.invalid/synthetic-service@", "incident-trial-lab.invalid/candidate-fixed@")
	put(t, f.root, filepath.Join(f.cd, "manifest.json"), candidateManifest)
	f.r = trial.Report{StartedAt: f.now.Add(time.Second).Format(time.RFC3339Nano), FinishedAt: f.now.Add(2 * time.Second).Format(time.RFC3339Nano), Completed: true, Cycles: 20, Requests: 20 * len(f.cases), Recovery: evaluation.Passed, Regressions: evaluation.Passed, Warmup: evaluation.Passed, BehavioralVerdict: evaluation.Passed, MedianMicros: 1, P95Micros: 1}
	for cycle := 0; cycle <= 20; cycle++ {
		for _, c := range f.cases {
			v := c.Expected
			o := trial.Observation{Cycle: cycle, Name: c.Name, Role: c.Role, Status: "pass", Expected: c.Expected, Observed: &v, HTTPStatus: 200, RawResponse: fmt.Sprintf(`{"total_cents":%d}`, v), DurationMicros: 1}
			if cycle == 0 {
				f.r.WarmupObservations = append(f.r.WarmupObservations, o)
			} else {
				f.r.Observations = append(f.r.Observations, o)
			}
		}
	}
	sample := func(tm time.Time, cpu int) string {
		return fmt.Sprintf(`{"pods":[{"podRef":{"namespace":"incident-trial-lab","uid":"pod-unit"},"containers":[{"name":"service","cpu":{"time":%q,"usageCoreNanoSeconds":%d},"memory":{"time":%q,"workingSetBytes":1024}}]}]}`, tm.Format(time.RFC3339Nano), cpu, tm.Format(time.RFC3339Nano))
	}
	put(t, f.root, filepath.Join(f.cd, "resources-before.json"), sample(f.now, 1))
	put(t, f.root, filepath.Join(f.cd, "resources-after.json"), sample(f.now.Add(3*time.Second), 100))
	putRecord(t, f, candidateID, "candidate", f.now, f.now.Add(4*time.Second), lifecycle.Completed)
	saveBehavior(t, f)
	return f
}
func putRecord(t *testing.T, f fixture, id, mode string, start, end time.Time, outcome lifecycle.Outcome) {
	put(t, f.root, "evidence/lifecycle-"+id+"/record.json", lifecycle.Record{Version: 1, RunID: id, Mode: mode, State: "finished", Result: &lifecycle.Result{Outcome: outcome, StartedAt: start.Format(time.RFC3339Nano), FinishedAt: end.Format(time.RFC3339Nano), Cleanup: lifecycle.CleanupResult{Complete: true}}})
}
func saveBehavior(t *testing.T, f fixture) {
	put(t, f.root, filepath.Join(f.cd, "behavior.json"), struct {
		FixtureHash string `json:"fixture_sha256"`
		trial.Report
	}{f.e.Spec.Workload, f.r})
	b, _ := read(f.root, filepath.Join(f.cd, "resources-before.json"))
	a, _ := read(f.root, filepath.Join(f.cd, "resources-after.json"))
	before, _ := trial.ParseSample(b)
	after, _ := trial.ParseSample(a)
	budget := trial.AssessBudget(before, after)
	if err := trial.CoversWorkload(before, after, f.r.StartedAt, f.r.FinishedAt); err != nil {
		budget.Status = evaluation.Inconclusive
		budget.Error = err.Error()
	}
	put(t, f.root, filepath.Join(f.cd, "result.json"), struct {
		Verdict          evaluation.Verdict `json:"verdict"`
		BaselineVerified bool               `json:"baseline_verified"`
		Budget           trial.Budget       `json:"resource_budget"`
		Behavior         trial.Report       `json:"behavior"`
	}{trial.Decide(true, f.r.BehavioralVerdict, budget), true, budget, f.r})
}
func bind(f fixture) (Receipt, error) {
	return Bind(f.root, f.a, candidateID, f.now.Add(5*time.Second))
}

func TestBindAndReplay(t *testing.T) {
	f := newFixture(t)
	r, err := bind(f)
	if err != nil || r.Verdict != evaluation.Passed || r.Spec.Workload != f.e.Spec.Workload || r.CandidateEvidenceHash == "" {
		t.Fatalf("receipt=%+v error=%v", r, err)
	}
	if _, err = bind(f); err == nil {
		t.Fatal("replayed attempt accepted")
	}
	a, err := Reserve(f.root, f.e.ID, "fixed", f.now.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	f.a = a
	if _, err = bind(f); err == nil {
		t.Fatal("replayed candidate accepted")
	}
	if _, err = Prepare(f.root, baselineID, f.now); err == nil {
		t.Fatal("replayed baseline accepted")
	}
}
func TestConcurrentConsumption(t *testing.T) {
	f := newFixture(t)
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for range 2 {
		wg.Add(1)
		go func() { defer wg.Done(); _, err := bind(f); results <- err }()
	}
	wg.Wait()
	close(results)
	accepted := 0
	for err := range results {
		if err == nil {
			accepted++
		}
	}
	if accepted != 1 {
		t.Fatalf("accepted %d concurrent binders", accepted)
	}
}
func TestFreshness(t *testing.T) {
	f := newFixture(t)
	if _, err := Load(f.root, f.e.ID, f.e.ExpiresAt); err == nil {
		t.Fatal("expired experiment")
	}
	if _, err := Load(f.root, f.e.ID, f.now.Add(-time.Second)); err == nil {
		t.Fatal("future experiment")
	}
	putRecord(t, f, baselineID, "baseline", f.now.Add(-12*time.Minute), f.now.Add(-11*time.Minute), lifecycle.Completed)
	if _, err := Prepare(f.root, baselineID, f.now); err == nil {
		t.Fatal("stale baseline")
	}
	putRecord(t, f, baselineID, "baseline", f.now, f.now.Add(time.Minute), lifecycle.Completed)
	if _, err := Prepare(f.root, baselineID, f.now); err == nil {
		t.Fatal("future baseline")
	}
}
func TestRejectChangedScenario(t *testing.T) {
	for _, path := range []string{"scenarios/discount-rounding/workload.json", "scenarios/discount-rounding/comparison-workload.json", "scenarios/discount-rounding/contract.md", "scenarios/discount-rounding/comparison-contract.md", "deploy/local/baseline.json", "deploy/local/candidates.json", "deploy/local/kind.yaml"} {
		t.Run(path, func(t *testing.T) {
			f := newFixture(t)
			b, _ := read(f.root, path)
			put(t, f.root, path, append(b, ' '))
			if _, err := Load(f.root, f.e.ID, f.now); err == nil {
				t.Fatal("changed scenario accepted")
			}
		})
	}
}
func TestRejectCandidateMismatch(t *testing.T) {
	tests := map[string]func(*testing.T, fixture){
		"workload": func(t *testing.T, f fixture) { put(t, f.root, filepath.Join(f.cd, "workload.json"), "[]") },
		"image": func(t *testing.T, f fixture) {
			put(t, f.root, filepath.Join(f.cd, "image-digest.txt"), "sha256:"+strings.Repeat("b", 64))
		},
		"manifest": func(t *testing.T, f fixture) { put(t, f.root, filepath.Join(f.cd, "image-manifest.json"), "{}") },
		"runtime":  func(t *testing.T, f fixture) { put(t, f.root, filepath.Join(f.cd, "runtime-image.json"), "{}") },
		"pod": func(t *testing.T, f fixture) {
			b, _ := read(f.root, filepath.Join(f.cd, "pods.json"))
			put(t, f.root, filepath.Join(f.cd, "pods.json"), strings.ReplaceAll(string(b), "service@", "different@sha256:"))
		},
		"proof": func(t *testing.T, f fixture) {
			put(t, f.root, filepath.Join(f.cd, "baseline-proof.json"), `{"baseline_reproduced":true}`)
		},
		"raw response": func(t *testing.T, f fixture) {
			f.r.Observations[0].RawResponse = `{"total_cents":0}`
			saveBehavior(t, f)
		},
		"missing observation": func(t *testing.T, f fixture) { f.r.Observations = f.r.Observations[1:]; saveBehavior(t, f) },
		"wrong role":          func(t *testing.T, f fixture) { f.r.Observations[0].Role = "regression"; saveBehavior(t, f) },
		"wrong cycle":         func(t *testing.T, f fixture) { f.r.Observations[0].Cycle = 2; saveBehavior(t, f) },
		"forged verdict":      func(t *testing.T, f fixture) { f.r.Recovery = evaluation.Failed; saveBehavior(t, f) },
		"old behavior": func(t *testing.T, f fixture) {
			f.r.StartedAt = f.now.Add(-time.Minute).Format(time.RFC3339Nano)
			saveBehavior(t, f)
		},
		"old run": func(t *testing.T, f fixture) {
			putRecord(t, f, candidateID, "candidate", f.now.Add(-time.Second), f.now.Add(4*time.Second), lifecycle.Completed)
		},
		"canceled": func(t *testing.T, f fixture) {
			putRecord(t, f, candidateID, "candidate", f.now, f.now.Add(4*time.Second), lifecycle.Canceled)
		},
		"failed": func(t *testing.T, f fixture) {
			putRecord(t, f, candidateID, "candidate", f.now, f.now.Add(4*time.Second), lifecycle.Failed)
		},
		"timed out": func(t *testing.T, f fixture) {
			putRecord(t, f, candidateID, "candidate", f.now, f.now.Add(4*time.Second), lifecycle.TimedOut)
		},
		"changed baseline": func(t *testing.T, f fixture) {
			b, _ := read(f.root, filepath.Join(f.bd, "check.json"))
			put(t, f.root, filepath.Join(f.bd, "check.json"), append(b, ' '))
		},
		"resource pod": func(t *testing.T, f fixture) {
			b, _ := read(f.root, filepath.Join(f.cd, "resources-before.json"))
			put(t, f.root, filepath.Join(f.cd, "resources-before.json"), strings.ReplaceAll(string(b), "pod-unit", "other-pod"))
			saveBehavior(t, f)
		},
		"symlink": func(t *testing.T, f fixture) {
			p := filepath.Join(f.root, f.cd, "workload.json")
			if err := os.Remove(p); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(filepath.Join(f.root, "scenarios/discount-rounding/comparison-workload.json"), p); err != nil {
				t.Fatal(err)
			}
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			f := newFixture(t)
			mutate(t, f)
			if _, err := bind(f); err == nil {
				t.Fatal("mismatched evidence accepted")
			}
		})
	}
}
func TestPreserveFailedAndUnknown(t *testing.T) {
	for _, unknown := range []bool{false, true} {
		t.Run(fmt.Sprint(unknown), func(t *testing.T) {
			f := newFixture(t)
			o := &f.r.Observations[0]
			v := int64(0)
			o.Observed = &v
			o.RawResponse = `{"total_cents":0}`
			o.Status = "fail"
			f.r.Recovery = evaluation.Failed
			f.r.BehavioralVerdict = evaluation.Failed
			want := evaluation.Failed
			if unknown {
				o.Observed = nil
				o.RawResponse = ""
				o.HTTPStatus = 0
				o.Status = "unknown"
				o.Error = "connection refused"
				f.r.Recovery = evaluation.Inconclusive
				f.r.BehavioralVerdict = evaluation.Inconclusive
				want = evaluation.Inconclusive
			}
			saveBehavior(t, f)
			r, err := bind(f)
			if err != nil || r.Verdict != want {
				t.Fatalf("verdict %s error %v", r.Verdict, err)
			}
		})
	}
}
func TestIDs(t *testing.T) {
	f := newFixture(t)
	if _, err := Load(f.root, "../other", f.now); err == nil {
		t.Fatal("invalid experiment ID")
	}
	if _, err := LoadAttempt(f.root, f.e.ID, "../../other"); err == nil {
		t.Fatal("invalid attempt ID")
	}
	if _, err := Reserve(f.root, f.e.ID, "arbitrary", f.now); err == nil {
		t.Fatal("unknown candidate")
	}
}

func TestVerifyRejectsChangedReceiptEvidence(t *testing.T) {
	f := newFixture(t)
	if _, err := bind(f); err != nil {
		t.Fatal(err)
	}
	if _, err := Verify(f.root, f.e.ID, f.a.ID, f.now.Add(5*time.Second)); err != nil {
		t.Fatal(err)
	}
	if _, err := Verify(f.root, f.e.ID, f.a.ID, f.e.ExpiresAt); err == nil {
		t.Fatal("expired receipt accepted")
	}
	f.r.Observations[0].DurationMicros = 2
	saveBehavior(t, f)
	if _, err := Verify(f.root, f.e.ID, f.a.ID, f.now.Add(5*time.Second)); err == nil {
		t.Fatal("changed candidate evidence accepted")
	}
}
func TestAppliedManifestMismatch(t *testing.T) {
	for _, baselinePhase := range []bool{false, true} {
		t.Run(fmt.Sprint(baselinePhase), func(t *testing.T) {
			f := newFixture(t)
			path := filepath.Join(f.cd, "manifest.json")
			if baselinePhase {
				path = filepath.Join(f.bd, "baseline.json")
			}
			put(t, f.root, path, `{"items":[]}`)
			if _, err := bind(f); err == nil {
				t.Fatal("different applied manifest accepted")
			}
		})
	}
}
