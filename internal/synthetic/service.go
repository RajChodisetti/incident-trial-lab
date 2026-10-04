// Package synthetic provides a deliberately faulty, synthetic quote service.
package synthetic

import (
	"encoding/json"
	"io"
	"net/http"
)

type Quote struct {
	UnitCents       int64 `json:"unit_cents"`
	Quantity        int64 `json:"quantity"`
	DiscountPercent int64 `json:"discount_percent"`
}

type Total struct {
	TotalCents int64 `json:"total_cents"`
}

func Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	mux.HandleFunc("POST /quote", func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 2048)
		defer r.Body.Close()
		dec := json.NewDecoder(r.Body)
		dec.DisallowUnknownFields()
		var q Quote
		if err := dec.Decode(&q); err != nil {
			http.Error(w, "invalid quote", 400)
			return
		}
		if err := dec.Decode(new(any)); err != io.EOF {
			http.Error(w, "extra data", 400)
			return
		}
		if q.UnitCents < 1 || q.UnitCents > 1000000 || q.Quantity < 1 || q.Quantity > 100 || q.DiscountPercent < 0 || q.DiscountPercent > 100 {
			http.Error(w, "quote out of range", 400)
			return
		}
		// Seeded defect: truncates each discounted unit instead of the order subtotal.
		total := q.UnitCents * (100 - q.DiscountPercent) / 100 * q.Quantity
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(Total{total})
	})
	return mux
}
