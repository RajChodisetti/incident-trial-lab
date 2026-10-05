package candidates

import "testing"

func TestFiniteCatalogIsNotMutableByCaller(t *testing.T) {
	ids := IDs()
	ids[0] = "unknown"
	if ValidID("unknown") || !ValidID("fixed") || len(IDs()) != 3 {
		t.Fatal("caller changed candidate policy")
	}
	for _, id := range []string{"", "Fixed", "fixed ", "shell"} {
		if ValidID(id) {
			t.Fatalf("invalid ID accepted: %q", id)
		}
	}
}
