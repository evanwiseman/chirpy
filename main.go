package main

import (
	"encoding/json"
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

func (cfg *apiConfig) metricsHandler(w http.ResponseWriter, r *http.Request) {
	hits := cfg.fileServerHits.Load()

	html := fmt.Sprintf(`<html>
	<body>
		<h1>Welcome, Chirpy Admin</h1>
		<p>Chirpy has been visited %d times!</p>
	</body>
</html>`,
		hits,
	)

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, html)
}

func (cfg *apiConfig) metricsReset(w http.ResponseWriter, r *http.Request) {
	cfg.fileServerHits.Store(0)
	hits := cfg.fileServerHits.Load()

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Reset Metrics. Hits: %v", hits)
}

func validateChirpHandler(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Body string `json:"body"`
	}
	type returnVals struct {
		Error string `json:"error"`
		Valid bool   `json:"valid"`
	}

	// Set the header
	w.Header().Set("Content-Type", "application/json")

	// Prepare the response & status code
	respBody := returnVals{}
	status := http.StatusOK

	// Decode the json from the request into Body
	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)

	if err != nil {
		respBody.Error = "Invalid JSON"
		status = http.StatusBadRequest
	} else if len(params.Body) > 140 {
		respBody.Error = "Chirp is too long"
		status = http.StatusBadRequest
	} else {
		respBody.Valid = true
	}

	data, err := json.Marshal(respBody)
	if err != nil {
		log.Printf("JSON marshal failed: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, `{"error":"internal server error"}`)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(data)
}

func main() {
	apiCfg := apiConfig{}

	serveMux := http.NewServeMux()

	// Create file server handlers
	fileServerHandler := http.FileServer((http.Dir(fileServerPath)))
	appHandler := http.StripPrefix("/app", fileServerHandler)

	// Attach handlers to the serve mux
	serveMux.Handle("/app/", apiCfg.middlewareMetricsInc(appHandler))
	serveMux.HandleFunc("GET /api/healthz", healthzHandler)
	serveMux.HandleFunc("GET /admin/metrics", apiCfg.metricsHandler)
	serveMux.HandleFunc("POST /admin/reset", apiCfg.metricsReset)
	serveMux.HandleFunc("POST /api/validate_chirp", validateChirpHandler)

	// Create the server at the desired port and attach the serve mux
	server := http.Server{
		Handler: serveMux,
		Addr:    ":" + serverPort,
	}

	// Start the server
	log.Printf("Serving files from %s on port: %s\n", fileServerPath, serverPort)
	log.Fatal(server.ListenAndServe())
}
