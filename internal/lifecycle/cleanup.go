package lifecycle

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var runIDPattern = regexp.MustCompile(`^[0-9a-f]{16}$`)

// OwnedNames returns only the allowlisted names derived from a unique run ID.
func OwnedNames(runID string) ([]string, error) {
	if !runIDPattern.MatchString(runID) {
		return nil, fmt.Errorf("invalid run ID")
	}
	names := []string{"baseline", "candidate-fixed", "candidate-ineffective", "candidate-misleading"}
	for i, name := range names {
		names[i] = "itl-" + name + "-" + runID
	}
	return names, nil
}

type CommandFunc func(context.Context, []string) ([]byte, error)

// CleanResources refuses broad selectors and independently checks cluster removal.
// Unique run-derived names identify resources created by this invocation. This
// is local cooperative ownership, not protection against a malicious Docker peer.
func CleanResources(ctx context.Context, runID, kindPath string, command CommandFunc) CleanupResult {
	result := CleanupResult{Complete: true}
	names, err := OwnedNames(runID)
	if err != nil {
		result.Complete = false
		result.Errors = append(result.Errors, err.Error())
		return result
	}
	configPath, removeConfig, configErr := temporaryCleanupConfig()
	if configErr != nil {
		result.Complete = false
		result.Errors = append(result.Errors, configErr.Error())
		return result
	}
	defer removeConfig()
	for _, name := range names {
		cluster := ClusterCleanup{Name: name}
		list := []string{"docker", "ps", "-aq", "--filter", "label=io.x-k8s.kind.cluster=" + name}
		raw, err := command(ctx, list)
		if err != nil {
			cluster.Error = "ownership inventory unavailable: " + err.Error()
			result.Complete = false
			result.Clusters = append(result.Clusters, cluster)
			continue
		}
		cluster.Found = strings.Fields(string(raw))
		if len(cluster.Found) > 0 {
			// Recheck each ID's exact label before authorizing deletion by name.
			verified := true
			for _, id := range cluster.Found {
				raw, err := command(ctx, []string{"docker", "inspect", "--format", "{{json .Config.Labels}}", id})
				var labels map[string]string
				if err != nil || json.Unmarshal(raw, &labels) != nil || labels["io.x-k8s.kind.cluster"] != name {
					verified = false
					cluster.Error = "container ownership could not be verified"
					break
				}
			}
			if verified {
				if _, err := command(ctx, []string{kindPath, "delete", "cluster", "--name", name, "--kubeconfig", configPath}); err != nil {
					cluster.Error = "cluster deletion failed: " + err.Error()
				}
			}
		}
		// Failed deletion is not reported as success merely because a command returned.
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

// RemoveLocalState only removes empty run-derived locks and regular kubeconfig
// files under this invocation's evidence root. It never follows symlinks.
func RemoveLocalState(root, runID string) []string {
	if !runIDPattern.MatchString(runID) {
		return []string{"invalid run ID"}
	}
	var problems []string
	locks := []string{"comparison", "baseline-run", "candidate-fixed", "candidate-ineffective", "candidate-misleading"}
	for _, name := range locks {
		path := filepath.Join(root, ".build", name+"-"+runID+".lock")
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			problems = append(problems, err.Error())
		}
	}
	evidence := filepath.Join(root, "evidence", "lifecycle-"+runID)
	err := filepath.WalkDir(evidence, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Name() == "kubeconfig" && d.Type().IsRegular() {
			return os.Remove(path)
		}
		return nil
	})
	if err != nil && !os.IsNotExist(err) {
		problems = append(problems, err.Error())
	}
	return problems
}

// kind mutates its kubeconfig during deletion; special devices are not valid targets.
// This file is newly allocated, owned by this cleanup invocation, and never points
// at the user's default kubeconfig.
func temporaryCleanupConfig() (string, func(), error) {
	f, err := os.CreateTemp("", "incident-trial-lab-cleanup-*.kubeconfig")
	if err != nil {
		return "", nil, err
	}
	path := f.Name()
	if err = f.Close(); err != nil {
		os.Remove(path)
		return "", nil, err
	}
	return path, func() { os.Remove(path) }, nil
}
