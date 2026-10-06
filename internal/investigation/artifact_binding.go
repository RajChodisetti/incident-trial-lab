package investigation

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/RajChodisetti/incident-trial-lab/internal/provenance"
)

// MatchSelectionSource links the selected source facts to a revalidated receipt.
// The caller MUST first use provenance.Verify; a supplied struct is not attestation.
func MatchSelectionSource(record SelectionRecord, receipt provenance.Receipt) error {
	data, _ := json.Marshal(record)
	if _, err := DecodeSelectionRecord(data); err != nil {
		return err
	}
	if record.Status != "selected" || receipt.SourceBinding == nil {
		return errors.New("selection lacks selected/bound artifact evidence")
	}
	id := record.OwnerAliases[record.Decision.Candidate]
	image := receipt.Spec.Candidates[id]
	if receipt.Variant != id || receipt.SourceBinding.Variant != id || !strings.HasPrefix(image, "sha256:") || !referenceHash.MatchString(strings.TrimPrefix(image, "sha256:")) || !referenceHash.MatchString(receipt.SourceBinding.BinarySHA256) {
		return errors.New("selected artifact identity mismatch")
	}
	for _, c := range record.Request.View.CandidateChanges {
		if c.Option == record.Decision.Candidate && c.SourceSHA256 == receipt.SourceBinding.SourceSHA256 && c.LogicSHA256 == receipt.SourceBinding.LogicSHA256 && receipt.SourceBinding.Version == 1 {
			return nil
		}
	}
	return errors.New("selected source/logic differs from bound artifact")
}
