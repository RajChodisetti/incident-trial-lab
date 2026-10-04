package evaluation

import "testing"

func TestDecide(t *testing.T) {
	tests := []struct {
		name      string
		completed bool
		checks    []Status
		want      Verdict
	}{
		{"no evidence", true, nil, Inconclusive},
		{"empty evidence", true, []Status{}, Inconclusive},
		{"unfinished passing trial", false, []Status{Pass}, Inconclusive},
		{"unfinished failing trial", false, []Status{Fail}, Inconclusive},
		{"missing check", true, []Status{Pass, Unknown}, Inconclusive},
		{"failure with missing evidence", true, []Status{Fail, Unknown}, Inconclusive},
		{"missing evidence before failure", true, []Status{Unknown, Fail}, Inconclusive},
		{"invalid check", true, []Status{Status(255)}, Inconclusive},
		{"one passing check", true, []Status{Pass}, Passed},
		{"all passing checks", true, []Status{Pass, Pass, Pass}, Passed},
		{"one failing check", true, []Status{Fail}, Failed},
		{"failure among passes", true, []Status{Pass, Fail, Pass}, Failed},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Decide(tt.completed, tt.checks)
			if got != tt.want {
				t.Fatalf("Decide(%v, %v) = %q, want %q", tt.completed, tt.checks, got, tt.want)
			}
		})
	}
}
