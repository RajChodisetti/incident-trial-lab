package artifact

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
)

func TestFingerprintUsesActualBinaryBytesAndCodeProjection(t *testing.T) {
	b, err := ForBinary("fixed", strings.NewReader("fixture binary bytes"))
	if err != nil {
		t.Fatal(err)
	}
	h := sha256.Sum256([]byte("fixture binary bytes"))
	if b.BinarySHA256 != hex.EncodeToString(h[:]) || Validate(b, "fixed", b.SourceSHA256, b.LogicSHA256, b.Labels()) != nil {
		t.Fatal("binding lacks byte/source fingerprint")
	}
	for _, mutate := range []func(*Binding){func(b *Binding) { b.Version = 0 }, func(b *Binding) { b.Variant = "misleading" }, func(b *Binding) { b.BinarySHA256 = "" }, func(b *Binding) { b.SourceSHA256 = strings.Repeat("0", 64) }, func(b *Binding) { b.LogicSHA256 = strings.Repeat("0", 64) }} {
		changed := b
		mutate(&changed)
		if Validate(changed, "fixed", b.SourceSHA256, b.LogicSHA256, b.Labels()) == nil {
			t.Fatal("mismatched binding accepted")
		}
	}
	if Validate(b, "fixed", b.SourceSHA256, b.LogicSHA256, nil) == nil {
		t.Fatal("unlabeled image accepted")
	}
	if _, err = ForBinary("unknown", strings.NewReader("binary")); err == nil {
		t.Fatal("unknown candidate accepted")
	}
	if _, err = ForBinary("fixed", strings.NewReader("")); err == nil {
		t.Fatal("empty binary accepted")
	}
}
