package candidates

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestUnknownAndInvalid(t *testing.T) {
	if _, err := Handler("unknown"); err == nil {
		t.Fatal("accepted unknown variant")
	}
	handler, _ := Handler("fixed")
	for _, body := range []string{`{`, `{}`, `{"unit_cents":1,"quantity":101}`, `{"unit_cents":1,"quantity":1,"extra":true}`} {
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, httptest.NewRequest("POST", "/quote", strings.NewReader(body)))
		if rr.Code != 400 {
			t.Fatal(rr.Code)
		}
	}
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, httptest.NewRequest("GET", "/healthz", nil))
	if rr.Code != 200 {
		t.Fatal(rr.Code)
	}
}
