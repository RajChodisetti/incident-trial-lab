package lifecycle

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

type NodeOwnership struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Cluster string `json:"cluster"`
}

type Recovery struct {
	OriginalOutcome   string        `json:"original_outcome"`
	BusinessVerdict   string        `json:"business_verdict"`
	ReconciledAt      string        `json:"reconciled_at"`
	ProcessesVerified bool          `json:"processes_verified"`
	Error             string        `json:"error,omitempty"`
	Cleanup           CleanupResult `json:"cleanup"`
}

func ProcessIdentity(ctx context.Context, pid int, command CommandFunc) (string, error) {
	raw, err := command(ctx, []string{"ps", "-p", fmt.Sprint(pid), "-o", "lstart=", "-o", "command="})
	if err != nil && len(strings.TrimSpace(string(raw))) == 0 {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(raw)), nil
}

// OwnershipSnapshots loads only regular atomic node snapshots from the run root.
func OwnershipSnapshots(dir, id string) (map[string]string, error) {
	names, err := OwnedNames(id)
	if err != nil {
		return nil, err
	}
	allowed := map[string]bool{}
	for _, name := range names {
		allowed[name] = true
	}
	owned := map[string]string{}
	err = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Name() != "ownership.json" {
			return nil
		}
		if !d.Type().IsRegular() {
			return fmt.Errorf("ownership snapshot is not regular")
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var node NodeOwnership
		if err = json.Unmarshal(data, &node); err != nil {
			return err
		}
		if !allowed[node.Cluster] || node.ID == "" || node.Name != "/"+node.Cluster+"-control-plane" {
			return fmt.Errorf("ownership snapshot does not match run")
		}
		if old, ok := owned[node.Cluster]; ok && old != node.ID {
			return fmt.Errorf("conflicting ownership snapshots")
		}
		owned[node.Cluster] = node.ID
		return nil
	})
	return owned, err
}

// CleanRecordedResources refuses unrecorded or replaced containers, even if their
// names/labels resemble the original run. Missing containers are idempotently clean.
func CleanRecordedResources(ctx context.Context, id, kind string, owned map[string]string, command CommandFunc) CleanupResult {
	result := CleanupResult{Complete: true}
	names, err := OwnedNames(id)
	if err != nil {
		result.Complete = false
		result.Errors = append(result.Errors, err.Error())
		return result
	}
	for _, name := range names {
		cluster := ClusterCleanup{Name: name}
		list := []string{"docker", "ps", "-aq", "--no-trunc", "--filter", "label=io.x-k8s.kind.cluster=" + name}
		raw, err := command(ctx, list)
		if err != nil {
			cluster.Error = "inventory unavailable: " + err.Error()
		} else {
			cluster.Found = strings.Fields(string(raw))
		}
		if err == nil && len(cluster.Found) > 0 {
			if len(cluster.Found) != 1 || owned[name] != cluster.Found[0] {
				cluster.Error = "unrecorded or replaced container; refusing deletion"
			} else {
				raw, e := command(ctx, []string{"docker", "inspect", "--format", `{"id":{{json .Id}},"name":{{json .Name}},"cluster":{{json (index .Config.Labels "io.x-k8s.kind.cluster")}}}`, owned[name]})
				var node NodeOwnership
				if e != nil || json.Unmarshal(raw, &node) != nil || node.ID != owned[name] || node.Cluster != name || node.Name != "/"+name+"-control-plane" {
					cluster.Error = "recorded identity/label mismatch; refusing deletion"
				} else {
					if _, e = command(ctx, []string{kind, "delete", "cluster", "--name", name, "--kubeconfig", os.DevNull}); e != nil {
						cluster.Error = "deletion failed: " + e.Error()
					}
				}
			}
		}
		raw, err = command(ctx, list)
		if err != nil {
			cluster.Error = "removal inventory unavailable: " + err.Error()
		} else {
			cluster.Remaining = strings.Fields(string(raw))
		}
		if cluster.Error != "" || len(cluster.Remaining) > 0 {
			result.Complete = false
		}
		result.Clusters = append(result.Clusters, cluster)
	}
	return result
}
