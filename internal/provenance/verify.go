package provenance

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"

	"github.com/RajChodisetti/incident-trial-lab/internal/evaluation"
	"github.com/RajChodisetti/incident-trial-lab/internal/trial"
)

func hashMustJSON(v any) string { b, _ := json.Marshal(v); return hash(b) }

// verifyRuntime connects the manifest content to the pod's config and image ID.
// The local evidence owner is trusted; these files are not remotely attested.
func verifyRuntime(root, dir, digest string) (string, error) {
	manifest, err := read(root, filepath.Join(dir, "image-manifest.json"))
	if err != nil {
		return "", err
	}
	if "sha256:"+hash(manifest) != digest {
		return "", fmt.Errorf("image manifest digest mismatch")
	}
	var m struct{ Config struct{ Digest string } }
	if json.Unmarshal(manifest, &m) != nil || m.Config.Digest == "" {
		return "", fmt.Errorf("missing image config identity")
	}
	pods, err := read(root, filepath.Join(dir, "pods.json"))
	if err != nil {
		return "", err
	}
	var p struct {
		Items []struct {
			Metadata struct{ UID string }
			Spec     struct {
				Containers []struct{ Name, Image string }
			}
			Status struct {
				ContainerStatuses []struct {
					Name, Image, ImageID string
					Ready                bool
				}
			}
		}
	}
	if json.Unmarshal(pods, &p) != nil || len(p.Items) != 1 || len(p.Items[0].Spec.Containers) != 1 || len(p.Items[0].Status.ContainerStatuses) != 1 {
		return "", fmt.Errorf("require one application pod/container")
	}
	c := p.Items[0].Spec.Containers[0]
	o := p.Items[0].Status.ContainerStatuses[0]
	parts := strings.Split(o.ImageID, "@")
	if p.Items[0].Metadata.UID == "" || c.Name != "service" || o.Name != "service" || !strings.HasSuffix(c.Image, "@"+digest) || o.Image != m.Config.Digest || !o.Ready || len(parts) != 2 {
		return "", fmt.Errorf("pod image identity mismatch")
	}
	runtime, err := read(root, filepath.Join(dir, "runtime-image.json"))
	if err != nil {
		return "", err
	}
	if "sha256:"+hash(runtime) != parts[1] {
		return "", fmt.Errorf("runtime image content mismatch")
	}
	if parts[1] != digest {
		var index struct{ Manifests []struct{ Digest string } }
		if json.Unmarshal(runtime, &index) != nil {
			return "", fmt.Errorf("invalid runtime index")
		}
		found := false
		for _, child := range index.Manifests {
			if child.Digest == digest {
				found = true
			}
		}
		if !found {
			return "", fmt.Errorf("runtime index lacks pinned manifest")
		}
	}
	return hash([]byte(hash(manifest) + hash(pods) + hash(runtime))), nil
}

// ValidateObservations recomputes assertions from every raw response, including
// warmup. It checks the committed protocol's exact 20 cycles, without retries.
func ValidateObservations(r trial.Report, cases []trial.Case) error {
	if r.Cycles != 20 || r.Requests != 20*len(cases) || len(r.Observations) != r.Requests || len(r.WarmupObservations) != len(cases) {
		return fmt.Errorf("incomplete or mismatched workload coverage")
	}
	var recovery, regression, warmup []evaluation.Status
	durations := make([]int64, 0, r.Requests)
	verify := func(o trial.Observation, c trial.Case, cycle int) (evaluation.Status, error) {
		if o.Cycle != cycle || o.Name != c.Name || o.Role != c.Role || o.Expected != c.Expected || o.DurationMicros < 0 {
			return evaluation.Unknown, fmt.Errorf("observation assertion/identity mismatch")
		}
		if o.Status == "unknown" {
			if o.Error == "" || o.Observed != nil {
				return evaluation.Unknown, fmt.Errorf("unknown observation contradicts response")
			}
			return evaluation.Unknown, nil
		}
		var total struct {
			Value *int64 `json:"total_cents"`
		}
		dec := json.NewDecoder(bytes.NewBufferString(o.RawResponse))
		dec.DisallowUnknownFields()
		if o.Error != "" || o.HTTPStatus != 200 || len(o.RawResponse) > 2048 || dec.Decode(&total) != nil || dec.Decode(new(any)) != io.EOF || total.Value == nil || o.Observed == nil || *total.Value != *o.Observed {
			return evaluation.Unknown, fmt.Errorf("observation lacks matching raw response")
		}
		expectedStatus := "fail"
		status := evaluation.Fail
		if *total.Value == c.Expected {
			expectedStatus = "pass"
			status = evaluation.Pass
		}
		if o.Status != expectedStatus {
			return evaluation.Unknown, fmt.Errorf("observation verdict contradicts assertion")
		}
		return status, nil
	}
	for i, o := range r.WarmupObservations {
		s, err := verify(o, cases[i], 0)
		if err != nil {
			return err
		}
		warmup = append(warmup, s)
	}
	for i, o := range r.Observations {
		c := cases[i%len(cases)]
		s, err := verify(o, c, i/len(cases)+1)
		if err != nil {
			return err
		}
		durations = append(durations, o.DurationMicros)
		if c.Role == "recovery" {
			recovery = append(recovery, s)
		} else {
			regression = append(regression, s)
		}
	}
	all := append(append(append([]evaluation.Status{}, recovery...), regression...), warmup...)
	if r.Recovery != evaluation.Decide(r.Completed, recovery) || r.Regressions != evaluation.Decide(r.Completed, regression) || r.Warmup != evaluation.Decide(r.Completed, warmup) || r.BehavioralVerdict != evaluation.Decide(r.Completed, all) {
		return fmt.Errorf("behavior verdict contradicts observations")
	}
	sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })
	if len(durations) == 0 || r.MedianMicros != durations[(len(durations)-1)/2] || r.P95Micros != durations[(95*len(durations)+99)/100-1] {
		return fmt.Errorf("latency summary contradicts observations")
	}
	return nil
}

func verifyBaselineResponse(raw string, required int64) error {
	var total struct {
		Value *int64 `json:"total_cents"`
	}
	dec := json.NewDecoder(strings.NewReader(raw))
	dec.DisallowUnknownFields()
	if len(raw) > 2048 || dec.Decode(&total) != nil || dec.Decode(new(any)) != io.EOF || total.Value == nil || *total.Value != required {
		return fmt.Errorf("baseline raw response contradicts assertion")
	}
	return nil
}
func runtimePodUID(root, dir string) (string, error) {
	b, err := read(root, filepath.Join(dir, "pods.json"))
	if err != nil {
		return "", err
	}
	var p struct {
		Items []struct{ Metadata struct{ UID string } }
	}
	if json.Unmarshal(b, &p) != nil || len(p.Items) != 1 {
		return "", fmt.Errorf("invalid pod identity")
	}
	return p.Items[0].Metadata.UID, nil
}

// Archived applied manifests must match the experiment specification; candidate
// manifests may replace only the one deployment image, retaining the controls.
func verifyDeployment(root, dir string, s Spec, variant string) error {
	original, err := read(root, "deploy/local/baseline.json")
	if err != nil || hash(original) != s.Manifest {
		return fmt.Errorf("deployment specification changed")
	}
	if variant == "" {
		archived, err := read(root, filepath.Join(dir, "baseline.json"))
		if err != nil || hash(archived) != s.Manifest {
			return fmt.Errorf("baseline applied manifest mismatch")
		}
		return nil
	}
	var manifest struct {
		Items []struct {
			Kind string
			Spec struct {
				Template struct {
					Spec struct{ Containers []struct{ Image string } }
				}
			}
		}
	}
	if json.Unmarshal(original, &manifest) != nil {
		return fmt.Errorf("invalid deployment specification")
	}
	image := ""
	count := 0
	for _, item := range manifest.Items {
		if item.Kind == "Deployment" && len(item.Spec.Template.Spec.Containers) == 1 {
			image = item.Spec.Template.Spec.Containers[0].Image
			count++
		}
	}
	if count != 1 {
		return fmt.Errorf("require one pinned deployment")
	}
	expected := strings.ReplaceAll(string(original), image, "incident-trial-lab.invalid/candidate-"+variant+"@"+s.Candidates[variant])
	archived, err := read(root, filepath.Join(dir, "manifest.json"))
	if err != nil {
		return err
	}
	var want, got any
	if json.Unmarshal([]byte(expected), &want) != nil || json.Unmarshal(archived, &got) != nil || hashMustJSON(want) != hashMustJSON(got) {
		return fmt.Errorf("candidate applied manifest mismatch")
	}
	return nil
}
