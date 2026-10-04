// Package evaluation conservatively classifies collected trial checks.
// It does not execute trials or decide which checks a scenario requires.
package evaluation

// Status records the outcome of a single collected check.
// Its zero value represents missing evidence.
type Status uint8

const (
	Unknown Status = iota
	Pass
	Fail
)

// Verdict summarizes a set of checks without discarding uncertainty.
type Verdict string

const (
	Inconclusive Verdict = "inconclusive"
	Passed       Verdict = "passed"
	Failed       Verdict = "failed"
)

// Decide reports Passed only for a completed trial with at least one check and
// every check passing. Incomplete or invalid evidence takes precedence over a
// failure: callers must preserve the individual checks alongside the verdict.
// The caller must also supply every check required by its scenario contract.
func Decide(completed bool, checks []Status) Verdict {
	if !completed || len(checks) == 0 {
		return Inconclusive
	}
	failed := false
	for _, check := range checks {
		switch check {
		case Pass:
		case Fail:
			failed = true
		default:
			return Inconclusive
		}
	}
	if failed {
		return Failed
	}
	return Passed
}
