// Package artifact links the quoted candidate source to a binary fingerprint.
// It assumes a trusted local builder/service; this is not signed build attestation.
package artifact

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"regexp"

	"github.com/RajChodisetti/incident-trial-lab/internal/candidates"
)

const LabelPrefix = "dev.incident-trial-lab."

type Binding struct {
	Version      int    `json:"version"`
	Variant      string `json:"variant"`
	SourceSHA256 string `json:"source_sha256"`
	LogicSHA256  string `json:"logic_sha256"`
	BinarySHA256 string `json:"binary_sha256"`
}

var digest = regexp.MustCompile(`^[0-9a-f]{64}$`)

func ForBinary(variant string, binary io.Reader) (Binding, error) {
	changes, err := candidates.Changes()
	if err != nil {
		return Binding{}, err
	}
	c, ok := changes[variant]
	if !ok {
		return Binding{}, errors.New("unknown binding variant")
	}
	h := sha256.New()
	n, err := io.Copy(h, io.LimitReader(binary, (64<<20)+1))
	if err != nil || n == 0 || n > 64<<20 {
		return Binding{}, errors.New("binary fingerprint unavailable or exceeds bound")
	}
	return Binding{1, variant, c.SourceSHA256, c.LogicSHA256, hex.EncodeToString(h.Sum(nil))}, nil
}

func (b Binding) Labels() map[string]string {
	return map[string]string{LabelPrefix + "variant": b.Variant, LabelPrefix + "source": b.SourceSHA256, LabelPrefix + "logic": b.LogicSHA256, LabelPrefix + "binary": b.BinarySHA256}
}

func Validate(b Binding, variant, source, logic string, labels map[string]string) error {
	if b.Version != 1 || !candidates.ValidID(variant) || b.Variant != variant || b.SourceSHA256 != source || b.LogicSHA256 != logic || !digest.MatchString(b.SourceSHA256) || !digest.MatchString(b.LogicSHA256) || !digest.MatchString(b.BinarySHA256) {
		return errors.New("missing or mismatched source binding")
	}
	for key, value := range b.Labels() {
		if labels[key] != value {
			return errors.New("image labels do not match runtime binary/source binding")
		}
	}
	return nil
}
