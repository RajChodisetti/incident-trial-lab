package main

import (
	"github.com/RajChodisetti/incident-trial-lab/internal/synthetic"
	"log"
	"net/http"
	"time"
)

func main() {
	server := &http.Server{Addr: ":8080", Handler: synthetic.Handler(), ReadHeaderTimeout: 2 * time.Second, ReadTimeout: 3 * time.Second, WriteTimeout: 3 * time.Second, IdleTimeout: 10 * time.Second}
	log.Fatal(server.ListenAndServe())
}
