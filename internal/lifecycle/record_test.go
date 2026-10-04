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

func TestRecordedDeletionRequiresExactIdentity(t *testing.T) {
	names, _ := OwnedNames(testRunID)
	owned := map[string]string{names[0]: "original-id"}
	for _, badLabel := range []bool{false, true} {
		remaining, deletions := true, 0
		run := func(ctx context.Context, args []string) ([]byte, error) {
			if args[0] == "docker" && args[1] == "ps" {
				if remaining && strings.HasSuffix(args[len(args)-1], names[0]) {
					return []byte("original-id\n"), nil
				}
				return nil, nil
			}
			if args[0] == "docker" && args[1] == "inspect" {
				cluster := names[0]
				if badLabel {
					cluster = "foreign"
				}
				data, _ := json.Marshal(NodeOwnership{"original-id", "/" + names[0] + "-control-plane", cluster})
				return data, nil
			}
			if args[0] == "kind" {
				deletions++
				remaining = false
				return nil, nil
			}
			return nil, fmt.Errorf("unexpected command")
		}
		result := CleanRecordedResources(context.Background(), testRunID, "kind", owned, run)
		if result.Complete == badLabel || (badLabel && deletions != 0) || (!badLabel && deletions != 1) {
			t.Fatal(result, deletions)
		}
	}
}

func TestSnapshotAndRecordPathRejectSymlinks(t *testing.T) {
	root := t.TempDir()
	evidence := filepath.Join(root, "evidence")
	if err := os.Mkdir(evidence, 0700); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(evidence, "lifecycle-"+testRunID)
	if err := os.Symlink(t.TempDir(), dir); err != nil {
		t.Fatal(err)
	}
	if _, err := RecordDir(root, testRunID); err == nil {
		t.Fatal("symlink run directory accepted")
	}
	snapshotDir := t.TempDir()
	target := filepath.Join(t.TempDir(), "node.json")
	if err := os.WriteFile(target, []byte(`{}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(snapshotDir, "ownership.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := OwnershipSnapshots(snapshotDir, testRunID); err == nil {
		t.Fatal("symlink snapshot accepted")
	}
}
