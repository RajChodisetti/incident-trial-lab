// Package trial evaluates deterministic candidate evidence without applying fixes.
package trial

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"time"

	"github.com/RajChodisetti/incident-trial-lab/internal/evaluation"
	"github.com/RajChodisetti/incident-trial-lab/internal/synthetic"
)

type Case struct {
	Name     string          `json:"name"`
	Role     string          `json:"role"`
	Request  synthetic.Quote `json:"request"`
	Expected int64           `json:"expected_cents"`
}

type Observation struct {
	Cycle          int    `json:"cycle"`
	Name           string `json:"name"`
	Role           string `json:"role"`
	Status         string `json:"status"`
	Expected       int64  `json:"expected_cents"`
	Observed       *int64 `json:"observed_cents,omitempty"`
	HTTPStatus     int    `json:"http_status,omitempty"`
	RawResponse    string `json:"raw_response,omitempty"`
	DurationMicros int64  `json:"duration_micros"`
	Error          string `json:"error,omitempty"`
}

type Report struct {
	StartedAt          string             `json:"started_at"`
	FinishedAt         string             `json:"finished_at"`
	Completed          bool               `json:"completed"`
	Recovery           evaluation.Verdict `json:"recovery"`
	Regressions        evaluation.Verdict `json:"regressions"`
	Warmup             evaluation.Verdict `json:"warmup"`
	BehavioralVerdict  evaluation.Verdict `json:"behavioral_verdict"`
	Cycles             int                `json:"cycles"`
	Requests           int                `json:"requests"`
	MedianMicros       int64              `json:"median_micros"`
	P95Micros          int64              `json:"p95_micros"`
	Error              string             `json:"error,omitempty"`
	Observations       []Observation      `json:"observations"`
	WarmupObservations []Observation      `json:"warmup_observations"`
}

func Load(data []byte) ([]Case, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	var cases []Case
	if err := dec.Decode(&cases); err != nil {
		return nil, err
	}
	if dec.Decode(new(any)) != io.EOF {
		return nil, fmt.Errorf("extra workload data")
	}
	if err := validate(cases); err != nil {
		return nil, err
	}
	return cases, nil
}

func validate(cases []Case) error {
	seen := map[string]bool{}
	recovery, regressions := 0, 0
	for _, c := range cases {
		q := c.Request
		if c.Name == "" || seen[c.Name] || q.UnitCents < 1 || q.UnitCents > 1000000 || q.Quantity < 1 || q.Quantity > 100 || q.DiscountPercent < 0 || q.DiscountPercent > 100 {
			return fmt.Errorf("invalid case %q", c.Name)
		}
		seen[c.Name] = true
		if c.Expected != q.UnitCents*q.Quantity*(100-q.DiscountPercent)/100 {
			return fmt.Errorf("incorrect assertion %q", c.Name)
		}
		switch c.Role {
		case "recovery":
			recovery++
		case "regression":
			regressions++
		default:
			return fmt.Errorf("invalid role %q", c.Role)
		}
	}
	if recovery != 1 || regressions < 2 {
		return fmt.Errorf("require one recovery and at least two regression cases")
	}
	return nil
}

func status(o Observation) evaluation.Status {
	switch o.Status {
	case "pass":
		return evaluation.Pass
	case "fail":
		return evaluation.Fail
	default:
		return evaluation.Unknown
	}
}

func Run(ctx context.Context, client *http.Client, url string, cases []Case, cycles int) Report {
	report := Report{Recovery: evaluation.Inconclusive, Regressions: evaluation.Inconclusive, Warmup: evaluation.Inconclusive, BehavioralVerdict: evaluation.Inconclusive, Cycles: cycles}
	if err := validate(cases); err != nil {
		report.Error = err.Error()
		return report
	}
	if cycles < 1 || cycles > 100 {
		report.Error = "cycles must be between 1 and 100"
		return report
	}
	report.StartedAt = time.Now().UTC().Format(time.RFC3339Nano)
	var recovery, regression, warmup []evaluation.Status
	var durations []int64
	for cycle := 0; cycle <= cycles; cycle++ {
		for _, c := range cases {
			obs := observe(ctx, client, url, c, cycle)
			if cycle == 0 {
				report.WarmupObservations = append(report.WarmupObservations, obs)
				warmup = append(warmup, status(obs))
			} else {
				report.Observations = append(report.Observations, obs)
				durations = append(durations, obs.DurationMicros)
				if c.Role == "recovery" {
					recovery = append(recovery, status(obs))
				} else {
					regression = append(regression, status(obs))
				}
			}
		}
	}
	report.FinishedAt = time.Now().UTC().Format(time.RFC3339Nano)
	report.Completed = ctx.Err() == nil
	report.Requests = len(report.Observations)
	report.Recovery = evaluation.Decide(report.Completed, recovery)
	report.Regressions = evaluation.Decide(report.Completed, regression)
	report.Warmup = evaluation.Decide(report.Completed, warmup)
	report.BehavioralVerdict = evaluation.Decide(report.Completed, append(append(recovery, regression...), warmup...))
	sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })
	if len(durations) > 0 {
		report.MedianMicros = durations[(len(durations)-1)/2]
		report.P95Micros = durations[(95*len(durations)+99)/100-1]
	}
	if ctx.Err() != nil {
		report.Error = ctx.Err().Error()
	}
	return report
}

func observe(ctx context.Context, client *http.Client, url string, c Case, cycle int) Observation {
	obs := Observation{Cycle: cycle, Name: c.Name, Role: c.Role, Expected: c.Expected, Status: "unknown"}
	start := time.Now()
	data, _ := json.Marshal(c.Request)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url+"/quote", bytes.NewReader(data))
	if err == nil {
		req.Header.Set("Content-Type", "application/json")
		var res *http.Response
		res, err = client.Do(req)
		if err == nil {
			obs.HTTPStatus = res.StatusCode
			raw, e := io.ReadAll(io.LimitReader(res.Body, 2049))
			res.Body.Close()
			obs.RawResponse = string(raw)
			if e != nil {
				err = e
			} else if len(raw) > 2048 || res.StatusCode != 200 {
				err = fmt.Errorf("invalid response status %d or size %d", res.StatusCode, len(raw))
			} else {
				var total struct {
					Value *int64 `json:"total_cents"`
				}
				dec := json.NewDecoder(bytes.NewReader(raw))
				dec.DisallowUnknownFields()
				err = dec.Decode(&total)
				if err == nil {
					if dec.Decode(new(any)) != io.EOF || total.Value == nil {
						err = fmt.Errorf("missing/malformed total")
					} else {
						obs.Observed = total.Value
					}
				}
			}
		}
	}
	obs.DurationMicros = time.Since(start).Microseconds()
	if err != nil {
		obs.Error = err.Error()
	} else {
		obs.Status = "fail"
		if *obs.Observed == c.Expected {
			obs.Status = "pass"
		}
	}
	return obs
}
