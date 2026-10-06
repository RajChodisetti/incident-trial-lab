// Package scenario projects only the observed baseline assertion and business rule.
package scenario

import (
	"bytes"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"strings"

	"github.com/RajChodisetti/incident-trial-lab/internal/baseline"
	"github.com/RajChodisetti/incident-trial-lab/internal/synthetic"
)

//go:embed contract.md
var contract []byte

//go:embed workload.json
var workload []byte

const Rule = "calculate an order's subtotal in integer cents, apply its percentage discount once to the subtotal, and truncate fractional cents once."

type Context struct {
	Rule           string          `json:"business_rule"`
	ContractSHA256 string          `json:"contract_sha256"`
	WorkloadSHA256 string          `json:"baseline_workload_sha256"`
	Request        synthetic.Quote `json:"observed_request"`
	RequiredCents  int64           `json:"assertion_required_cents"`
	ObservedCents  int64           `json:"observed_cents"`
	HTTPStatus     int             `json:"observed_http_status"`
}

func digest(data []byte) string { h := sha256.Sum256(data); return hex.EncodeToString(h[:]) }

// Project ignores prose beyond the verified business sentence. It never embeds
// comparison workloads, expected candidate outcomes, response text or case names.
func Project(report baseline.Report) (Context, error) {
	var c Context
	if !strings.Contains(strings.Join(strings.Fields(string(contract)), " "), "Business contract: "+Rule) || !report.BaselineReproduced || report.FixtureError != "" || report.HealthError != "" {
		return c, errors.New("business contract or reproduced baseline unverified")
	}
	cases, err := baseline.Load(workload)
	if err != nil || len(report.Observations) != len(cases) {
		return c, errors.New("baseline observations do not match workload")
	}
	seen := map[string]bool{}
	failures := 0
	for _, o := range report.Observations {
		var match *baseline.Case
		for i := range cases {
			if cases[i].Name == o.Name {
				match = &cases[i]
			}
		}
		if match == nil || seen[o.Name] || o.Error != "" || o.Observed == nil || o.Expected != match.Expected || o.HTTPStatus != 200 {
			return c, errors.New("observed assertion unverified")
		}
		seen[o.Name] = true
		var result synthetic.Total
		dec := json.NewDecoder(bytes.NewBufferString(o.RawResponse))
		dec.DisallowUnknownFields()
		if len(o.RawResponse) > 2048 || dec.Decode(&result) != nil || dec.Decode(new(any)) != io.EOF || result.TotalCents != *o.Observed {
			return c, errors.New("observed numeric response unverified")
		}
		if match.Faulty == nil {
			if o.BusinessCheck != "pass" || *o.Observed != match.Expected {
				return c, errors.New("baseline control assertion unverified")
			}
		} else {
			if o.BusinessCheck != "fail" || *o.Observed != *match.Faulty {
				return c, errors.New("baseline failure unverified")
			}
			failures++
			c = Context{Rule: Rule, ContractSHA256: digest(contract), WorkloadSHA256: digest(workload), Request: match.Request, RequiredCents: o.Expected, ObservedCents: *o.Observed, HTTPStatus: o.HTTPStatus}
		}
	}
	if failures != 1 {
		return Context{}, errors.New("one observed baseline failure required")
	}
	return c, nil
}

// Validate checks the bounded typed projection against the current scenario.
// This is an assertion/context check, not a candidate verdict or image attestation.
func Validate(c Context) error {
	cases, err := baseline.Load(workload)
	if err != nil {
		return errors.New("scenario workload invalid")
	}
	for _, fixture := range cases {
		if fixture.Faulty != nil && c.Rule == Rule && c.ContractSHA256 == digest(contract) && c.WorkloadSHA256 == digest(workload) && c.Request == fixture.Request && c.RequiredCents == fixture.Expected && c.ObservedCents == *fixture.Faulty && c.HTTPStatus == 200 {
			return nil
		}
	}
	return errors.New("invalid business/incident projection")
}
