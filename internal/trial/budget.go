package trial

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/RajChodisetti/incident-trial-lab/internal/evaluation"
)

// Sample contains application-container counters from kubelet stats/summary.
type Sample struct {
	PodUID          string `json:"pod_uid"`
	CPUTime         string `json:"cpu_time"`
	MemoryTime      string `json:"memory_time"`
	CPUNanos        uint64 `json:"cpu_nanos"`
	WorkingSetBytes uint64 `json:"working_set_bytes"`
}

type Budget struct {
	Status                     evaluation.Verdict `json:"status"`
	Scope                      string             `json:"scope"`
	Error                      string             `json:"error,omitempty"`
	CPUDeltaNanos              uint64             `json:"cpu_delta_nanos"`
	CPULimitNanos              uint64             `json:"cpu_limit_nanos"`
	MaxObservedWorkingSetBytes uint64             `json:"max_observed_working_set_bytes"`
	WorkingSetLimitBytes       uint64             `json:"working_set_limit_bytes"`
	PeakMemoryMeasured         bool               `json:"peak_memory_measured"`
}

func ParseSample(data []byte) (Sample, error) {
	var summary struct {
		Pods []struct {
			PodRef     struct{ Name, Namespace, UID string }
			Containers []struct {
				Name string
				CPU  struct {
					Time                 string
					UsageCoreNanoSeconds *uint64
				}
				Memory struct {
					Time            string
					WorkingSetBytes *uint64
				}
			}
		}
	}
	if err := json.Unmarshal(data, &summary); err != nil {
		return Sample{}, err
	}
	for _, pod := range summary.Pods {
		if pod.PodRef.Namespace != "incident-trial-lab" {
			continue
		}
		for _, c := range pod.Containers {
			if c.Name != "service" {
				continue
			}
			if pod.PodRef.UID == "" || c.CPU.Time == "" || c.Memory.Time == "" || c.CPU.UsageCoreNanoSeconds == nil || c.Memory.WorkingSetBytes == nil {
				return Sample{}, fmt.Errorf("missing application counters")
			}
			return Sample{pod.PodRef.UID, c.CPU.Time, c.Memory.Time, *c.CPU.UsageCoreNanoSeconds, *c.Memory.WorkingSetBytes}, nil
		}
	}
	return Sample{}, fmt.Errorf("application container sample absent")
}

func AssessBudget(before, after Sample) Budget {
	budget := Budget{Status: evaluation.Inconclusive, Scope: "container CPU interval and two endpoint working-set samples; excludes peak memory and throttling", CPULimitNanos: 5000000000, WorkingSetLimitBytes: 32 * 1024 * 1024}
	if before.PodUID == "" || before.PodUID != after.PodUID || before.CPUTime == "" || before.CPUTime == after.CPUTime || before.MemoryTime == "" || before.MemoryTime == after.MemoryTime || after.CPUNanos < before.CPUNanos {
		budget.Error = "missing, stale, changed-pod, or reset counters"
		return budget
	}
	for _, pair := range [][2]string{{before.CPUTime, after.CPUTime}, {before.MemoryTime, after.MemoryTime}} {
		start, e1 := time.Parse(time.RFC3339Nano, pair[0])
		end, e2 := time.Parse(time.RFC3339Nano, pair[1])
		if e1 != nil || e2 != nil || !end.After(start) {
			budget.Error = "invalid or non-monotonic sample timestamps"
			return budget
		}
	}
	budget.CPUDeltaNanos = after.CPUNanos - before.CPUNanos
	budget.MaxObservedWorkingSetBytes = max(before.WorkingSetBytes, after.WorkingSetBytes)
	budget.Status = evaluation.Passed
	if budget.CPUDeltaNanos > budget.CPULimitNanos || budget.MaxObservedWorkingSetBytes > budget.WorkingSetLimitBytes {
		budget.Status = evaluation.Failed
	}
	return budget
}

// Decide requires baseline, behavior, and the explicitly scoped budget evidence.
func Decide(baselineVerified bool, behavior evaluation.Verdict, budget Budget) evaluation.Verdict {
	convert := func(v evaluation.Verdict) evaluation.Status {
		if v == evaluation.Passed {
			return evaluation.Pass
		}
		if v == evaluation.Failed {
			return evaluation.Fail
		}
		return evaluation.Unknown
	}
	baselineStatus := evaluation.Unknown
	if baselineVerified {
		baselineStatus = evaluation.Pass
	}
	return evaluation.Decide(true, []evaluation.Status{baselineStatus, convert(behavior), convert(budget.Status)})
}

// CoversWorkload rejects sampling windows that omit any of the workload.
func CoversWorkload(before, after Sample, started, finished string) error {
	start, e1 := time.Parse(time.RFC3339Nano, started)
	end, e2 := time.Parse(time.RFC3339Nano, finished)
	if e1 != nil || e2 != nil || end.Before(start) {
		return fmt.Errorf("invalid workload timestamps")
	}
	for _, pair := range [][2]string{{before.CPUTime, after.CPUTime}, {before.MemoryTime, after.MemoryTime}} {
		b, e1 := time.Parse(time.RFC3339Nano, pair[0])
		a, e2 := time.Parse(time.RFC3339Nano, pair[1])
		if e1 != nil || e2 != nil || b.After(start) || a.Before(end) {
			return fmt.Errorf("resource snapshots do not bracket workload")
		}
	}
	return nil
}
