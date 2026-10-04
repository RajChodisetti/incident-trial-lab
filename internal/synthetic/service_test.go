package synthetic

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSeededDefectAndControls(t *testing.T) {
	for _, tt := range []struct{ name, body, want string }{
		{"no discount", `{"unit_cents":105,"quantity":3,"discount_percent":0}`, `{"total_cents":315}`},
		{"single unit", `{"unit_cents":105,"quantity":1,"discount_percent":10}`, `{"total_cents":94}`},
		{"known seeded defect", `{"unit_cents":105,"quantity":3,"discount_percent":10}`, `{"total_cents":282}`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			Handler().ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/quote", strings.NewReader(tt.body)))
			if rr.Code != 200 || strings.TrimSpace(rr.Body.String()) != tt.want {
				t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
			}
		})
	}
}

func TestRejectInvalidRequests(t *testing.T) {
	for _, body := range []string{`{`, `{}`, `{"unit_cents":105,"quantity":0}`, `{"unit_cents":-1,"quantity":3}`, `{"unit_cents":105,"quantity":3,"discount_percent":101}`, `{"unit_cents":105,"quantity":3,"extra":true}`, `{"unit_cents":105,"quantity":3} {}`, strings.Repeat(" ", 2049)} {
		rr := httptest.NewRecorder()
		Handler().ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/quote", strings.NewReader(body)))
		if rr.Code != 400 {
			t.Fatalf("body=%q status=%d", body, rr.Code)
		}
	}
}

func TestRoutes(t *testing.T) {
	for _, tt := range []struct {
		method, path string
		code         int
	}{{"GET", "/healthz", 200}, {"GET", "/quote", 405}, {"GET", "/missing", 404}} {
		rr := httptest.NewRecorder()
		Handler().ServeHTTP(rr, httptest.NewRequest(tt.method, tt.path, nil))
		if rr.Code != tt.code {
			t.Fatalf("%s %s status=%d", tt.method, tt.path, rr.Code)
		}
	}
}
