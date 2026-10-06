package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/RajChodisetti/incident-trial-lab/internal/artifact"
	"github.com/RajChodisetti/incident-trial-lab/internal/candidates"
)

// variant is baked into each candidate binary by the local build script.
var variant string

func main() {
	handler, err := candidates.Handler(variant)
	if err != nil {
		log.Fatal(err)
	}
	path, err := os.Executable()
	if err != nil {
		log.Fatal("binary identity unavailable")
	}
	f, err := os.Open(path)
	if err != nil {
		log.Fatal("binary fingerprint unavailable")
	}
	binding, err := artifact.ForBinary(variant, f)
	_ = f.Close()
	if err != nil {
		log.Fatal(err)
	}
	mux := http.NewServeMux()
	mux.Handle("/", handler)
	mux.HandleFunc("GET /artifact-binding", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(binding)
	})
	server := &http.Server{Addr: ":8080", Handler: mux, ReadHeaderTimeout: 2 * time.Second, ReadTimeout: 3 * time.Second, WriteTimeout: 3 * time.Second, IdleTimeout: 10 * time.Second}
	log.Fatal(server.ListenAndServe())
}
