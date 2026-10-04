// Package worker runs a small in-memory queue of isolated local trial jobs.
package worker

import (
	"context"
	"fmt"
	"sync"
)

type Job struct {
	ID      string `json:"id"`
	Variant string `json:"variant"`
}
type Result struct {
	Job             Job    `json:"job"`
	Status          string `json:"status"`
	RunID           string `json:"run_id,omitempty"`
	Error           string `json:"error,omitempty"`
	BusinessVerdict string `json:"business_verdict"`
	CleanupVerified bool   `json:"cleanup_verified"`
}
type Execute func(context.Context, Job) Result

type entry struct {
	ctx      context.Context
	cancel   context.CancelFunc
	finished bool
}
type Pool struct {
	mu      sync.Mutex
	entries map[string]*entry
	workers int
}

func New(workers int) (*Pool, error) {
	if workers == 0 {
		workers = 1
	}
	if workers < 1 || workers > 2 {
		return nil, fmt.Errorf("workers must be one or two")
	}
	return &Pool{entries: map[string]*entry{}, workers: workers}, nil
}

// Cancel affects only a registered unfinished job in this pool instance.
func (p *Pool) Cancel(id string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	e := p.entries[id]
	if e == nil || e.finished {
		return false
	}
	e.cancel()
	return true
}

// Run is single-use, bounded to eight queued jobs. Results preserve input order.
func (p *Pool) Run(ctx context.Context, jobs []Job, execute Execute) ([]Result, error) {
	p.mu.Lock()
	if execute == nil || p.workers < 1 || p.workers > 2 || len(p.entries) > 0 || len(jobs) == 0 || len(jobs) > 8 {
		p.mu.Unlock()
		return nil, fmt.Errorf("single-use pool requires 1..8 jobs")
	}
	for _, job := range jobs {
		if job.ID == "" || p.entries[job.ID] != nil {
			for _, e := range p.entries {
				e.cancel()
			}
			p.mu.Unlock()
			return nil, fmt.Errorf("duplicate/empty job ID")
		}
		c, cancel := context.WithCancel(ctx)
		p.entries[job.ID] = &entry{ctx: c, cancel: cancel}
	}
	p.mu.Unlock()
	results := make([]Result, len(jobs))
	queue := make(chan int, len(jobs))
	for i := range jobs {
		queue <- i
	}
	close(queue)
	var workers sync.WaitGroup
	for n := 0; n < p.workers; n++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for i := range queue {
				p.mu.Lock()
				e := p.entries[jobs[i].ID]
				p.mu.Unlock()
				result := Result{Job: jobs[i], Status: "canceled_before_start", BusinessVerdict: "inconclusive", CleanupVerified: true}
				if e.ctx.Err() == nil {
					result = execute(e.ctx, jobs[i])
				}
				result.Job = jobs[i]
				result.BusinessVerdict = BusinessVerdict(result.Status, result.CleanupVerified, result.BusinessVerdict)
				p.mu.Lock()
				e.finished = true
				e.cancel()
				p.mu.Unlock()
				results[i] = result
			}
		}()
	}
	workers.Wait()
	return results, nil
}

func (p *Pool) Limit() int { return p.workers }

// BusinessVerdict prevents partial passing observations from accepting canceled
// or cleanup-incomplete trials. Raw observations remain in the trial evidence.
func BusinessVerdict(status string, clean bool, observed string) string {
	if status != "completed" || !clean {
		return "inconclusive"
	}
	switch observed {
	case "passed", "failed", "inconclusive":
		return observed
	default:
		return "inconclusive"
	}
}
