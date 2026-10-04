package lifecycle

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRecordLockAndAtomicRoundTrip(t *testing.T) {
	dir := t.TempDir()
	r := Record{Version: 1, RunID: testRunID, Mode: "candidate", State: "prepared"}
	if err := SaveRecord(dir, r); err != nil {
		t.Fatal(err)
	}
	got, err := LoadRecord(dir, testRunID)
	if err != nil || got.State != "prepared" {
		t.Fatal(got, err)
	}
	f, err := LockRecord(dir)
	if err != nil {
		t.Fatal(err)
	}
	if other, err := LockRecord(dir); err == nil {
		other.Close()
		t.Fatal("live lock accepted")
	}
	f.Close()
	f, err = LockRecord(dir)
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	if _, err := LoadRecord(dir, "fedcba9876543210"); err == nil {
		t.Fatal("ID mismatch accepted")
	}
	if err := os.WriteFile(filepath.Join(dir, "record.json"), []byte("{"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadRecord(dir, testRunID); err == nil {
		t.Fatal("truncated record accepted")
	}
}

func TestOwnershipSnapshotsAndReplacedResource(t *testing.T) {
	dir := t.TempDir()
	names, _ := OwnedNames(testRunID)
	node := NodeOwnership{ID: "original-id", Name: "/" + names[0] + "-control-plane", Cluster: names[0]}
	data, _ := json.Marshal(node)
	if err := os.WriteFile(filepath.Join(dir, "ownership.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
	owned, err := OwnershipSnapshots(dir, testRunID)
	if err != nil || owned[names[0]] != "original-id" {
		t.Fatal(owned, err)
	}
	deletions := 0
	run := func(ctx context.Context, args []string) ([]byte, error) {
		if args[0] == "docker" && args[1] == "ps" {
			if strings.HasSuffix(args[len(args)-1], names[0]) {
				return []byte("replacement-id\n"), nil
			}
			return nil, nil
		}
		deletions++
		return nil, fmt.Errorf("unexpected deletion/inspection")
	}
	result := CleanRecordedResources(context.Background(), testRunID, "kind", owned, run)
	if result.Complete || deletions != 0 {
		t.Fatal(result, deletions)
	}
	result = CleanRecordedResources(context.Background(), testRunID, "kind", nil, run)
	if result.Complete || deletions != 0 {
		t.Fatal("unrecorded node accepted")
	}
	result = CleanRecordedResources(context.Background(), testRunID, "kind", owned, func(context.Context, []string) ([]byte, error) { return nil, nil })
	if !result.Complete {
		t.Fatal("idempotent missing-resource cleanup failed", result)
	}
}

func TestPersistStartFailureStopsWork(t *testing.T) {
	c := config("/bin/sh", "-c", "sleep 30")
	c.Started = func(int) error { return fmt.Errorf("injected journal failure") }
	r := Run(context.Background(), c)
	if r.Outcome != Failed || !strings.Contains(r.Error, "persist process identity") || !r.Cleanup.Complete {
		t.Fatal(r)
	}
}
