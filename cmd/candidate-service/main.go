package main

import (
	"log"
	"net/http"
	"time"

	"github.com/RajChodisetti/incident-trial-lab/internal/candidates"
)

// variant is baked into each candidate binary by the local build script.
var variant string

func main() {
	handler, err := candidates.Handler(variant)
	if err != nil {
		log.Fatal(err)
	}
	server := &http.Server{Addr: ":8080", Handler: handler, ReadHeaderTimeout: 2 * time.Second, ReadTimeout: 3 * time.Second, WriteTimeout: 3 * time.Second, IdleTimeout: 10 * time.Second}
	log.Fatal(server.ListenAndServe())
}
