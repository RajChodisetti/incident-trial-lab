package lifecycle

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testRunID = "0123456789abcdef"

func TestOwnedNames(t *testing.T) {
	names, err := OwnedNames(testRunID)
	if err != nil || len(names) != 4 {
		t.Fatal(names, err)
	}
	for _, id := range []string{"", "../foreign", "default", "0123456789abcdeg"} {
		if _, err := OwnedNames(id); err == nil {
			t.Fatal("invalid ownership ID accepted")
		}
	}
}

func TestCleanupOwnership(t *testing.T) {
	names, _ := OwnedNames(testRunID)
	for _, tt := range []struct {
		name                  string
		badLabel, deleteFails bool
		want                  bool
	}{{"owned only", false, false, true}, {"label mismatch", true, false, false}, {"delete failure", false, true, false}} {
		t.Run(tt.name, func(t *testing.T) {
			remaining := true
			deletions := 0
			run := func(ctx context.Context, args []string) ([]byte, error) {
				if args[0] == "docker" && args[1] == "ps" {
					filter := args[len(args)-1]
					allowed := false
					for _, name := range names {
						if filter == "label=io.x-k8s.kind.cluster="+name {
							allowed = true
						}
					}
					if !allowed {
						t.Fatalf("unscoped inventory: %v", args)
					}
					if strings.HasSuffix(filter, names[0]) && remaining {
						return []byte("owned-container\n"), nil
					}
					return nil, nil
				}
				if args[0] == "docker" && args[1] == "inspect" {
					label := names[0]
					if tt.badLabel {
						label = "unrelated-project"
					}
					return []byte(fmt.Sprintf(`{"io.x-k8s.kind.cluster":%q}`, label)), nil
				}
				if args[0] == "kind" {
					deletions++
					if args[4] != names[0] {
						t.Fatalf("foreign deletion: %v", args)
					}
					if tt.deleteFails {
						return nil, fmt.Errorf("injected failure")
					}
					remaining = false
					return nil, nil
				}
				t.Fatalf("unexpected command: %v", args)
				return nil, nil
			}
			result := CleanResources(context.Background(), testRunID, "kind", run)
			if result.Complete != tt.want {
				t.Fatal(result)
			}
			if tt.badLabel && deletions != 0 {
				t.Fatal("deleted label-mismatched container")
			}
		})
	}
	result := CleanResources(context.Background(), testRunID, "kind", func(context.Context, []string) ([]byte, error) { return nil, fmt.Errorf("Docker unavailable") })
	if result.Complete {
		t.Fatal("unavailable inventory reported clean")
	}
}

func TestRemoveOnlyOwnedLocalState(t *testing.T) {
	root := t.TempDir()
	owned := filepath.Join(root, ".build", "comparison-"+testRunID+".lock")
	foreign := filepath.Join(root, ".build", "comparison-foreign.lock")
	evidence := filepath.Join(root, "evidence", "lifecycle-"+testRunID, "trial")
	if err := os.MkdirAll(owned, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(foreign, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(evidence, 0700); err != nil {
		t.Fatal(err)
	}
	config := filepath.Join(evidence, "kubeconfig")
	if err := os.WriteFile(config, []byte("local fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(root, "foreign-kubeconfig")
	if err := os.WriteFile(outside, []byte("preserve"), 0600); err != nil {
		t.Fatal(err)
	}
	linkDir := filepath.Join(evidence, "linked")
	if err := os.Mkdir(linkDir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(linkDir, "kubeconfig")); err != nil {
		t.Fatal(err)
	}
	if errors := RemoveLocalState(root, testRunID); len(errors) != 0 {
		t.Fatal(errors)
	}
	if _, err := os.Stat(config); !os.IsNotExist(err) {
		t.Fatal("owned credentials retained")
	}
	for _, path := range []string{foreign, outside, filepath.Join(linkDir, "kubeconfig")} {
		if _, err := os.Lstat(path); err != nil {
			t.Fatalf("foreign/symlink state changed: %v", err)
		}
	}
}
