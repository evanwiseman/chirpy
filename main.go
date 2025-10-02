package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"sync/atomic"

	"github.com/evanwiseman/chirpy/internal/database"
	"github.com/evanwiseman/chirpy/internal/handlers"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

const (
	serverPort     = "8080"
	fileServerPath = "."
)

func main() {
	// Load .env
	godotenv.Load()

	// Load postgres database
	dbURL := os.Getenv("DB_URL")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("failed to open database %v", dbURL)
	}

	// Create API Config
	apiCfg := handlers.APIConfig{
		DB:             database.New(db),
		FileServerHits: atomic.Int32{},
		Platform:       os.Getenv("PLATFORM"),
		JWTSecret:      os.Getenv("JWT_SECRET"),
	}

	serveMux := http.NewServeMux()

	// Create file server handlers
	fileServerHandler := http.FileServer((http.Dir(fileServerPath)))
	appHandler := http.StripPrefix("/app", fileServerHandler)

	// Attach handlers to the serve mux
	serveMux.Handle("/app/", apiCfg.MiddlewareMetricsInc(appHandler))

	serveMux.HandleFunc("GET /admin/metrics", apiCfg.HandlerGetMetrics)
	serveMux.HandleFunc("POST /admin/reset", apiCfg.HandlerPostReset)

	serveMux.HandleFunc("GET /api/healthz", handlers.HandlerHealthz)

	serveMux.HandleFunc("POST /api/users", apiCfg.HandlerPostUsers)
	serveMux.HandleFunc("POST /api/login", apiCfg.HandlerLogin)

	serveMux.HandleFunc("POST /api/chirps", apiCfg.HandlerPostChirps)
	serveMux.HandleFunc("GET /api/chirps", apiCfg.HandlerGetChirps)
	serveMux.HandleFunc("GET /api/chirps/{chirpID}", apiCfg.HandlerGetChripByID)

	// Create the server at the desired port and attach the serve mux
	server := http.Server{
		Handler: serveMux,
		Addr:    ":" + serverPort,
	}

	// Start the server
	log.Printf("Serving files from %s on port: %s\n", fileServerPath, serverPort)
	log.Fatal(server.ListenAndServe())
}
