package main

import (
	"fmt"
	"log"
	"net/http"
	"sync/atomic"
)

const (
	serverPort     = "8080"
	fileServerPath = "."
)

type apiConfig struct {
	fileServerHits atomic.Int32
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileServerHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func healthzHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "OK")
}

func (cfg *apiConfig) Handler(w http.ResponseWriter, r *http.Request) {
	hits := cfg.fileServerHits.Load()

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Hits: %v", hits)
}

func (cfg *apiConfig) Reset(w http.ResponseWriter, r *http.Request) {
	cfg.fileServerHits.Store(0)
	hits := cfg.fileServerHits.Load()

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Reset Metrics. Hits: %v", hits)
}

func main() {
	apiCfg := apiConfig{}

	serveMux := http.NewServeMux()

	// Create file server handlers
	fileServerHandler := http.FileServer((http.Dir(fileServerPath)))
	appHandler := http.StripPrefix("/app", fileServerHandler)

	// Attach handlers to the serve mux
	serveMux.Handle("/app/", apiCfg.middlewareMetricsInc(appHandler))
	serveMux.HandleFunc("/healthz", healthzHandler)
	serveMux.HandleFunc("/metrics", apiCfg.Handler)
	serveMux.HandleFunc("/reset", apiCfg.Reset)

	// Create the server at the desired port and attach the serve mux
	server := http.Server{
		Handler: serveMux,
		Addr:    ":" + serverPort,
	}

	// Start the server
	log.Printf("Serving files from %s on port: %s\n", fileServerPath, serverPort)
	log.Fatal(server.ListenAndServe())
}
