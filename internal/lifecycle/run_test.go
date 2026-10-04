package lifecycle

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func config(command ...string) Config {
	return Config{Command: command, Stdout: io.Discard, Stderr: io.Discard, GracePeriod: 40 * time.Millisecond, CleanupTimeout: time.Second, Cleanup: func(ctx context.Context) CleanupResult { return CleanupResult{Complete: ctx.Err() == nil} }}
}

func TestCompletionAndFailure(t *testing.T) {
	for _, tt := range []struct {
		name, script string
		want         Outcome
		exit         int
	}{{"success", "exit 0", Completed, 0}, {"failure", "exit 7", Failed, 7}} {
		t.Run(tt.name, func(t *testing.T) {
			r := Run(context.Background(), config("/bin/sh", "-c", tt.script))
			if r.Outcome != tt.want || r.ExitCode != tt.exit || !r.Cleanup.Complete {
				t.Fatal(r)
			}
		})
	}
	c := config("/nonexistent/trial-command")
	if r := Run(context.Background(), c); r.Outcome != Failed || r.Error == "" || !r.Cleanup.Complete {
		t.Fatal(r)
	}
}

func TestCancellationAndIndependentCleanup(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	c := config("/bin/sh", "-c", "exit 0")
	called := false
	c.Cleanup = func(cleanupCtx context.Context) CleanupResult {
		called = true
		if cleanupCtx.Err() != nil {
			t.Error("cleanup inherited cancellation")
		}
		if _, ok := cleanupCtx.Deadline(); !ok {
			t.Error("cleanup has no deadline")
		}
		return CleanupResult{Complete: true}
	}
	r := Run(ctx, c)
	if r.Outcome != Canceled || !called {
		t.Fatal(r)
	}
	ctx, cancel = context.WithCancel(context.Background())
	defer cancel()
	timer := time.AfterFunc(30*time.Millisecond, cancel)
	defer timer.Stop()
	r = Run(ctx, config("/bin/sh", "-c", "sleep 30"))
	if r.Outcome != Canceled || !r.Cleanup.Complete {
		t.Fatal(r)
	}
}

func TestTimeoutKillsDescendants(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "orphan-marker")
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Millisecond)
	defer cancel()
	// This fixture ignores TERM in the shell and its descendants to force KILL.
	c := config("/bin/sh", "-c", `trap '' TERM; (sleep 0.4; printf leaked > "$1") & wait`, "fixture", marker)
	r := Run(ctx, c)
	if r.Outcome != TimedOut || !r.ForcedKill || !r.Cleanup.Complete {
		t.Fatal(r)
	}
	time.Sleep(450 * time.Millisecond)
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("descendant survived: %v", err)
	}
}

func TestCleanupFailureCannotPass(t *testing.T) {
	c := config("/bin/sh", "-c", "exit 0")
	c.Cleanup = func(ctx context.Context) CleanupResult {
		return CleanupResult{Complete: false, Errors: []string{"injected cleanup failure"}}
	}
	r := Run(context.Background(), c)
	if r.Outcome != Failed || r.Cleanup.Complete || r.Error == "" {
		t.Fatal(r)
	}
	c.CleanupTimeout = 20 * time.Millisecond
	c.Cleanup = func(ctx context.Context) CleanupResult {
		<-ctx.Done()
		return CleanupResult{Complete: false, Errors: []string{ctx.Err().Error()}}
	}
	r = Run(context.Background(), c)
	if r.Outcome != Failed || r.Cleanup.Complete {
		t.Fatal(r)
	}
}

func TestFailedParentDoesNotLeaveChild(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "orphan-marker")
	r := Run(context.Background(), config("/bin/sh", "-c", `(sleep 0.3; printf leaked > "$1") & exit 7`, "fixture", marker))
	if r.Outcome != Failed || !r.Cleanup.Complete {
		t.Fatal(r)
	}
	time.Sleep(350 * time.Millisecond)
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("failure orphan survived: %v", err)
	}
}
