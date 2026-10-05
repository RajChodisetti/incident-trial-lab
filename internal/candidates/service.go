// Package candidates contains explicit synthetic candidate variants for comparison.
package candidates

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/RajChodisetti/incident-trial-lab/internal/synthetic"
)

// Handler intentionally keeps candidate code separate from the frozen baseline.
func Handler(variant string) (http.Handler, error) {
	if !ValidID(variant) {
		return nil, fmt.Errorf("unknown candidate %q", variant)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })
	mux.HandleFunc("POST /quote", func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 2048))
		dec.DisallowUnknownFields()
		var q synthetic.Quote
		if dec.Decode(&q) != nil || dec.Decode(new(any)) != io.EOF || q.UnitCents < 1 || q.UnitCents > 1000000 || q.Quantity < 1 || q.Quantity > 100 || q.DiscountPercent < 0 || q.DiscountPercent > 100 {
			http.Error(w, "invalid quote", 400)
			return
		}
		total := q.UnitCents * (100 - q.DiscountPercent) / 100 * q.Quantity
		if variant == "fixed" || (variant == "misleading" && q.UnitCents == 105 && q.Quantity == 3 && q.DiscountPercent == 10) {
			total = q.UnitCents * q.Quantity * (100 - q.DiscountPercent) / 100
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(synthetic.Total{TotalCents: total})
	})
	return mux, nil
}
