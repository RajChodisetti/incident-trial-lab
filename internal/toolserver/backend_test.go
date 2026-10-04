package toolserver

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestReadJSONBoundary(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "report.json")
	var v any
	for _, data := range []string{`{"state":"ok"}`, `{`, "{} {}"} {
		if err := os.WriteFile(path, []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
		err := readJSON(path, &v)
		if (data == `{"state":"ok"}`) != (err == nil) {
			t.Fatalf("data %q error %v", data, err)
		}
	}
	outside := filepath.Join(root, "outside.json")
	os.WriteFile(outside, []byte("{}"), 0600)
	os.Remove(path)
	os.Symlink(outside, path)
	if err := readJSON(path, &v); err == nil {
		t.Fatal("symlink accepted")
	}
	os.Remove(path)
	os.WriteFile(path, make([]byte, 2*1024*1024+1), 0600)
	if err := readJSON(path, &v); err == nil {
		t.Fatal("oversized evidence accepted")
	}
}
func TestLocalBackendOwnerConfiguration(t *testing.T) {
	for _, timeout := range []time.Duration{0, 30 * time.Second, 11 * time.Minute} {
		if _, err := NewLocalBackend(t.TempDir(), timeout); err == nil {
			t.Fatal("invalid deadline")
		}
	}
	if _, err := NewLocalBackend("relative", time.Minute); err == nil {
		t.Fatal("relative owner root")
	}
	root := t.TempDir()
	if _, err := NewLocalBackend(root, time.Minute); err == nil {
		t.Fatal("unrelated root")
	}
	os.WriteFile(filepath.Join(root, "go.mod"), []byte("module github.com/RajChodisetti/incident-trial-lab\n"), 0600)
	if _, err := NewLocalBackend(root, time.Minute); err == nil {
		t.Fatal("missing trusted supervisor")
	}
	b := &LocalBackend{root: root, logs: root, timeout: time.Minute}
	r := b.Execute(context.Background(), "../../other", "baseline", "", nil)
	if r.State != "start_failed" || !r.CleanupVerified {
		t.Fatalf("invalid job %+v", r)
	}
	r = b.Execute(context.Background(), "1111111111111111", "candidate", "custom", nil)
	if r.State != "start_failed" {
		t.Fatalf("candidate %+v", r)
	}
}
