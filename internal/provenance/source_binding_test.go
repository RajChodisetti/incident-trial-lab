package provenance

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/RajChodisetti/incident-trial-lab/internal/artifact"
)

func TestSourceBindingChainAndMissingOrMismatchedEvidence(t *testing.T) {
	f := newFixture(t)
	b, err := VerifySourceBinding(f.root, f.cd, "fixed", f.digest)
	if err != nil || b.SourceSHA256 == "" || b.BinarySHA256 == "" {
		t.Fatalf("binding=%+v error=%v", b, err)
	}
	for _, which := range []string{"missing config", "changed config", "missing runtime", "source", "logic", "binary", "variant", "unknown field"} {
		t.Run(which, func(t *testing.T) {
			f := newFixture(t)
			b, _ := artifact.ForBinary("fixed", strings.NewReader("offline fixture binary"))
			switch which {
			case "missing config":
				_ = os.Remove(filepath.Join(f.root, f.cd, "image-config.json"))
			case "changed config":
				put(t, f.root, filepath.Join(f.cd, "image-config.json"), `{"config":{"Labels":{}}}`)
			case "missing runtime":
				_ = os.Remove(filepath.Join(f.root, f.cd, "source-binding.json"))
			case "source":
				b.SourceSHA256 = strings.Repeat("0", 64)
			case "logic":
				b.LogicSHA256 = strings.Repeat("0", 64)
			case "binary":
				b.BinarySHA256 = strings.Repeat("0", 64)
			case "variant":
				b.Variant = "ineffective"
			case "unknown field":
				put(t, f.root, filepath.Join(f.cd, "source-binding.json"), `{"version":1,"unexpected":"value"}`)
			}
			if which == "source" || which == "logic" || which == "binary" || which == "variant" {
				put(t, f.root, filepath.Join(f.cd, "source-binding.json"), b)
			}
			if _, err := Bind(f.root, f.a, candidateID, f.now.Add(5*time.Second)); err == nil {
				t.Fatal("unbound/mismatched artifact received a receipt")
			}
		})
	}
}

func TestSourceProofRemainsRequiredOnReceiptRevalidation(t *testing.T) {
	f := newFixture(t)
	r, err := Bind(f.root, f.a, candidateID, f.now.Add(5*time.Second))
	if err != nil || r.SourceBinding == nil {
		t.Fatal("bound receipt missing source proof")
	}
	_ = os.Remove(filepath.Join(f.root, f.cd, "source-binding.json"))
	if _, err = Verify(f.root, f.e.ID, f.a.ID, f.now.Add(5*time.Second)); err == nil {
		t.Fatal("receipt survived loss of source proof")
	}
}
