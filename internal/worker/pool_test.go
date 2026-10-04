package worker

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestBoundAndIndependentCancellation(t *testing.T) {
	p, err := New(2)
	if err != nil {
		t.Fatal(err)
	}
	started := make(chan string, 3)
	release := make(chan struct{})
	var active, peak atomic.Int32
	done := make(chan []Result, 1)
	go func() {
		results, err := p.Run(context.Background(), []Job{{ID: "one"}, {ID: "two"}, {ID: "three"}}, func(ctx context.Context, j Job) Result {
			a := active.Add(1)
			for {
				old := peak.Load()
				if a <= old || peak.CompareAndSwap(old, a) {
					break
				}
			}
			defer active.Add(-1)
			started <- j.ID
			select {
			case <-ctx.Done():
				return Result{Job: j, Status: "canceled", CleanupVerified: true}
			case <-release:
				return Result{Job: j, Status: "completed", CleanupVerified: true}
			}
		})
		if err != nil {
			t.Error(err)
		}
		done <- results
	}()
	<-started
	<-started
	if peak.Load() != 2 || p.Cancel("unknown") {
		t.Fatal("bound/unknown cancellation failed")
	}
	if !p.Cancel("one") {
		t.Fatal("registered cancellation failed")
	}
	<-started // Queued third job starts only after cancellation frees a slot.
	close(release)
	results := <-done
	if results[0].Status != "canceled" || results[1].Status != "completed" || results[2].Status != "completed" || peak.Load() > 2 {
		t.Fatal(results, peak.Load())
	}
	if p.Cancel("two") {
		t.Fatal("finished job canceled")
	}
}

func TestConservativeDefaultAndCanceledQueue(t *testing.T) {
	p, _ := New(0)
	if p.Limit() != 1 {
		t.Fatal(p.Limit())
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var calls atomic.Int32
	results, err := p.Run(ctx, []Job{{ID: "one"}, {ID: "two"}}, func(context.Context, Job) Result { calls.Add(1); return Result{} })
	if err != nil || calls.Load() != 0 || results[0].Status != "canceled_before_start" {
		t.Fatal(results, err)
	}
	for _, n := range []int{-1, 3} {
		if _, err := New(n); err == nil {
			t.Fatal("unsafe worker count accepted")
		}
	}
	p, _ = New(1)
	if _, err = p.Run(context.Background(), []Job{{ID: "same"}, {ID: "same"}}, nil); err == nil {
		t.Fatal("duplicate jobs accepted")
	}
}

func TestParentCancellation(t *testing.T) {
	p, _ := New(2)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	results, err := p.Run(ctx, []Job{{ID: "one"}, {ID: "two"}}, func(ctx context.Context, j Job) Result { <-ctx.Done(); return Result{Job: j, Status: "canceled"} })
	if err != nil || results[0].Status != "canceled" || results[1].Status != "canceled" {
		t.Fatal(results, err)
	}
}

func TestPartialPassCannotAcceptCanceledTrial(t *testing.T) {
	for _, tt := range []struct {
		status         string
		clean          bool
		observed, want string
	}{
		{"canceled", true, "passed", "inconclusive"},
		{"completed", false, "passed", "inconclusive"},
		{"completed", true, "failed", "failed"},
		{"completed", true, "passed", "passed"},
		{"needs_recovery", false, "passed", "inconclusive"},
	} {
		if got := BusinessVerdict(tt.status, tt.clean, tt.observed); got != tt.want {
			t.Fatal(tt, got)
		}
	}
}
