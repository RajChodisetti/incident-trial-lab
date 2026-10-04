// Package baseline checks reproduction of the seeded discount-rounding incident.
package baseline

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/RajChodisetti/incident-trial-lab/internal/synthetic"
	"io"
	"net/http"
)

type Case struct {
	Name     string          `json:"name"`
	Request  synthetic.Quote `json:"request"`
	Expected int64           `json:"expected_cents"`
	Faulty   *int64          `json:"known_faulty_cents,omitempty"`
}

type Observation struct {
	Name          string `json:"name"`
	Expected      int64  `json:"expected_cents"`
	Observed      *int64 `json:"observed_cents,omitempty"`
	HTTPStatus    int    `json:"http_status,omitempty"`
	RawResponse   string `json:"raw_response,omitempty"`
	BusinessCheck string `json:"business_check"`
	Error         string `json:"error,omitempty"`
}

type Report struct {
	FixtureError       string        `json:"fixture_error,omitempty"`
	BaselineReproduced bool          `json:"baseline_reproduced"`
	HealthError        string        `json:"health_error,omitempty"`
	Observations       []Observation `json:"observations"`
}

// Load validates the fixture against the independent business formula.
func Load(data []byte) ([]Case, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	var cases []Case
	if err := dec.Decode(&cases); err != nil {
		return nil, err
	}
	if err := dec.Decode(new(any)); err != io.EOF {
		return nil, fmt.Errorf("extra fixture data")
	}
	if err := validate(cases); err != nil {
		return nil, err
	}
	return cases, nil
}

func validate(cases []Case) error {
	names := map[string]bool{}
	faulty, controls := 0, 0
	for _, c := range cases {
		q := c.Request
		if c.Name == "" || names[c.Name] || q.UnitCents < 1 || q.UnitCents > 1000000 || q.Quantity < 1 || q.Quantity > 100 || q.DiscountPercent < 0 || q.DiscountPercent > 100 {
			return fmt.Errorf("invalid fixture case %q", c.Name)
		}
		names[c.Name] = true
		correct := q.UnitCents * q.Quantity * (100 - q.DiscountPercent) / 100
		if c.Expected != correct {
			return fmt.Errorf("fixture %q contradicts business formula", c.Name)
		}
		if c.Faulty != nil {
			faulty++
			seeded := q.UnitCents * (100 - q.DiscountPercent) / 100 * q.Quantity
			if *c.Faulty == correct || *c.Faulty != seeded {
				return fmt.Errorf("invalid seeded value for %q", c.Name)
			}
		} else {
			controls++
		}
	}
	if faulty != 1 || controls < 2 {
		return fmt.Errorf("require one seeded failure and at least two controls")
	}
	return nil
}

// Run succeeds only if controls pass and the designated failure is reproduced.
// Unavailable/malformed evidence cannot count as a business failure.
func Run(ctx context.Context, client *http.Client, baseURL string, cases []Case) Report {
	report := Report{}
	if err := validate(cases); err != nil {
		report.FixtureError = err.Error()
		return report
	}
	report.BaselineReproduced = true
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/healthz", nil)
	if err != nil {
		report.HealthError = err.Error()
	} else {
		res, e := client.Do(req)
		if e != nil {
			report.HealthError = e.Error()
		} else {
			res.Body.Close()
			if res.StatusCode != 200 {
				report.HealthError = fmt.Sprintf("health status %d", res.StatusCode)
			}
		}
	}
	if report.HealthError != "" {
		report.BaselineReproduced = false
	}
	for _, c := range cases {
		obs := Observation{Name: c.Name, Expected: c.Expected, BusinessCheck: "unknown"}
		data, _ := json.Marshal(c.Request)
		req, e := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/quote", bytes.NewReader(data))
		if e == nil {
			req.Header.Set("Content-Type", "application/json")
			var res *http.Response
			res, e = client.Do(req)
			if e == nil {
				obs.HTTPStatus = res.StatusCode
				raw, readErr := io.ReadAll(io.LimitReader(res.Body, 2049))
				res.Body.Close()
				obs.RawResponse = string(raw)
				if readErr != nil {
					e = readErr
				} else if len(raw) > 2048 || res.StatusCode != 200 {
					e = fmt.Errorf("invalid quote response: status %d, bytes %d", res.StatusCode, len(raw))
				} else {
					// Pointer distinguishes a missing field from a legitimate zero total.
					var total struct {
						Value *int64 `json:"total_cents"`
					}
					dec := json.NewDecoder(bytes.NewReader(raw))
					dec.DisallowUnknownFields()
					e = dec.Decode(&total)
					if e == nil {
						if dec.Decode(new(any)) != io.EOF || total.Value == nil {
							e = fmt.Errorf("malformed total")
						} else {
							obs.Observed = total.Value
						}
					}
				}
			}
		}
		if e != nil {
			obs.Error = e.Error()
			report.BaselineReproduced = false
		} else {
			obs.BusinessCheck = "fail"
			if *obs.Observed == c.Expected {
				obs.BusinessCheck = "pass"
			}
			required := c.Expected
			if c.Faulty != nil {
				required = *c.Faulty
			}
			if *obs.Observed != required {
				report.BaselineReproduced = false
			}
		}
		report.Observations = append(report.Observations, obs)
	}
	return report
}
