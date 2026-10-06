package provenance

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"

	"github.com/RajChodisetti/incident-trial-lab/internal/artifact"
	"github.com/RajChodisetti/incident-trial-lab/internal/candidates"
)

// VerifySourceBinding extends the existing manifest/config/pod identity chain.
// The service reports hashes of its own executable and compiled source projection.
// Local build metadata and captured responses remain trusted-owner evidence.
func VerifySourceBinding(root, dir, variant, image string) (artifact.Binding, error) {
	var b artifact.Binding
	if _, err := verifyRuntime(root, dir, image); err != nil {
		return b, err
	}
	manifest, err := read(root, filepath.Join(dir, "image-manifest.json"))
	if err != nil {
		return b, err
	}
	var m struct{ Config struct{ Digest string } }
	if json.Unmarshal(manifest, &m) != nil {
		return b, fmt.Errorf("source binding manifest invalid")
	}
	config, err := read(root, filepath.Join(dir, "image-config.json"))
	if err != nil || "sha256:"+hash(config) != m.Config.Digest {
		return b, fmt.Errorf("missing or mismatched image config content")
	}
	var c struct {
		Config struct{ Labels map[string]string }
	}
	if json.Unmarshal(config, &c) != nil {
		return b, fmt.Errorf("image config labels invalid")
	}
	body, err := read(root, filepath.Join(dir, "source-binding.json"))
	if err != nil || len(body) > 2048 {
		return b, fmt.Errorf("runtime source binding missing or exceeds bound")
	}
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.DisallowUnknownFields()
	if dec.Decode(&b) != nil || dec.Decode(new(any)) != io.EOF {
		return b, fmt.Errorf("runtime source binding malformed")
	}
	changes, err := candidates.Changes()
	if err != nil {
		return b, err
	}
	change, ok := changes[variant]
	if !ok {
		return b, fmt.Errorf("source binding candidate unknown")
	}
	if err = artifact.Validate(b, variant, change.SourceSHA256, change.LogicSHA256, c.Config.Labels); err != nil {
		return b, err
	}
	return b, nil
}
