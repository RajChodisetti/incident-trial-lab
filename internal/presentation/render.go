package presentation

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"

	"github.com/RajChodisetti/incident-trial-lab/internal/provenance"
)

//go:embed page.html
var page string

type row struct {
	Name, Run, Recovery, Regressions, Outcome, Image, Source, Logic, Binary, CPU, Memory string
	Failed                                                                               int
}
type caseRow struct {
	Name, Role string
	Required   int64
	Values     []int64
}
type view struct {
	Page, Title, Loop, Started, Finished, Audited, RefusalLoop string
	Required, Observed                                         int64
	Rows                                                       []row
	Cases                                                      []caseRow
	Proof                                                      string
}

func makeView(b Bundle) (view, error) {
	s := b.Replay
	v := view{Loop: s.Trace.LoopID, Started: s.Trace.StartedAt, Finished: s.Trace.FinishedAt, Audited: s.RevalidatedAt.UTC().Format("2006-01-02 15:04:05 UTC"), RefusalLoop: b.Refusal.Trace.LoopID}
	behaviors := map[string]int{}
	for i, c := range b.Captures.Captures {
		if c.Evidence.Baseline != nil {
			for _, o := range c.Evidence.Baseline.Observations {
				if o.BusinessCheck == "fail" && o.Observed != nil {
					v.Required = o.Expected
					v.Observed = *o.Observed
				}
			}
		}
		if c.Evidence.Behavior != nil {
			behaviors[c.Evidence.Trial.Variant] = i
		}
	}
	if v.Required != 283 || v.Observed != 282 || len(behaviors) != 3 {
		return v, fmt.Errorf("display evidence incomplete")
	}
	for _, a := range s.Trace.Attempts {
		id := a.Proposal.Variant
		r := s.Receipts[id]
		binding := r.SourceBinding
		budget := s.Budgets[id]
		v.Rows = append(v.Rows, row{Name: id, Run: a.Trial.RunID, Recovery: string(a.Signal.Recovery), Regressions: string(a.Signal.Regressions), Outcome: a.Outcome, Failed: a.Signal.Failed, Image: r.Spec.Candidates[id], Source: binding.SourceSHA256, Logic: binding.LogicSHA256, Binary: binding.BinarySHA256, CPU: fmt.Sprintf("%.3f", float64(budget.CPUDeltaNanos)/1e9), Memory: fmt.Sprintf("%.2f", float64(budget.MaxObservedWorkingSetBytes)/(1<<20))})
	}
	for i, o := range b.Captures.Captures[behaviors["fixed"]].Evidence.Behavior.Observations[:6] {
		c := caseRow{Name: strings.ReplaceAll(o.Name, "-", " "), Role: o.Role, Required: o.Expected}
		for _, id := range []string{"ineffective", "misleading", "fixed"} {
			observed := b.Captures.Captures[behaviors[id]].Evidence.Behavior.Observations[i]
			if observed.Cycle != 1 || observed.Name != o.Name || observed.Expected != o.Expected || observed.Observed == nil || observed.HTTPStatus != 200 {
				return v, fmt.Errorf("case evidence incomplete")
			}
			c.Values = append(c.Values, *observed.Observed)
		}
		v.Cases = append(v.Cases, c)
	}
	proof, _ := json.MarshalIndent(struct {
		Loop     string `json:"loop"`
		Audit    string `json:"audited_at"`
		Receipts any    `json:"receipts"`
		Hashes   any    `json:"capture_hashes"`
	}{v.Loop, v.Audited, s.Receipts, s.CaptureHashes}, "", "  ")
	v.Proof = string(proof)
	return v, nil
}

const failurePage = `<!doctype html><html lang="en"><meta charset="utf-8"><title>Evidence unavailable</title><style>body{font:20px system-ui;margin:80px;max-width:700px}h1{color:#922}</style><h1>Evidence unavailable</h1><p>The recorded bundle is missing, changed or inconsistent. No recovery result is displayed.</p><p>Restore and validate the reviewed evidence before generating this local demo. This page starts no trial.</p></html>`

// Generate also refuses scenario/pin drift before rendering.
func Generate(root string) error {
	spec, err := provenance.CurrentSpec(root)
	if err != nil {
		return unavailable(root, err)
	}
	return Export(root, spec)
}

func unavailable(root string, cause error) error {
	dir := filepath.Join(root, ".build", "demo")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	for _, name := range []string{"index.html", "comparison.html", "provenance.html"} {
		if e := os.WriteFile(filepath.Join(dir, name), []byte(failurePage), 0600); e != nil {
			return e
		}
	}
	if e := os.Remove(filepath.Join(dir, "evidence.json")); e != nil && !os.IsNotExist(e) {
		return e
	}
	return cause
}

// Export writes three static pages. Invalid input replaces every old result page
// with a neutral error page, returns an error and removes the prior JSON export.
func Export(root string, spec provenance.Spec) error {
	dir := filepath.Join(root, ".build", "demo")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	b, err := Load(root, spec)
	var v view
	if err == nil {
		v, err = makeView(b)
	}
	if err != nil {
		return unavailable(root, err)
	}
	t, err := template.New("demo").Funcs(template.FuncMap{"short": func(s string) string {
		s = strings.TrimPrefix(s, "sha256:")
		if len(s) > 12 {
			return s[:12]
		}
		return s
	}}).Parse(page)
	if err != nil {
		return err
	}
	for _, p := range []struct{ name, title string }{{"index", "A one-cent failure. A verifiable recovery."}, {"comparison", "Passing the incident is only half the test."}, {"provenance", "Trace the result back to the running artifact."}} {
		v.Page = p.name
		v.Title = p.title
		var out bytes.Buffer
		if err = t.Execute(&out, v); err != nil {
			return err
		}
		if err = os.WriteFile(filepath.Join(dir, p.name+".html"), out.Bytes(), 0600); err != nil {
			return err
		}
	}
	data, _ := json.MarshalIndent(b.Replay, "", "  ")
	return os.WriteFile(filepath.Join(dir, "evidence.json"), append(data, '\n'), 0600)
}
