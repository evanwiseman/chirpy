package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/evanwiseman/chirpy/internal/database"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

const (
	serverPort     = "8080"
	fileServerPath = "."
)

type apiConfig struct {
	db             *database.Queries
	fileServerHits atomic.Int32
	platform       string
}

type User struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
}

type Chirp struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body      string    `json:"body"`
	UserID    uuid.UUID `json:"user_id"`
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileServerHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func (cfg *apiConfig) handlerGetMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")

	hits := cfg.fileServerHits.Load()
	html := fmt.Sprintf(`<html>
	<body>
		<h1>Welcome, Chirpy Admin</h1>
		<p>Chirpy has been visited %d times!</p>
	</body>
</html>`,
		hits,
	)

	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, html)
}

func (cfg *apiConfig) handlerPostReset(w http.ResponseWriter, r *http.Request) {
	// Don't reset if not on development database
	if cfg.platform != "dev" {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	cfg.fileServerHits.Store(0)

	// Attempt to reset the user database
	err := cfg.db.ResetUsers(r.Context())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func handlerHealthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "OK")
}

func (cfg *apiConfig) handlerPostUsers(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Email string `json:"email"`
	}

	// Set the header
	w.Header().Set("Content-Type", "application/json")

	// Decode the json from the request
	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		log.Printf("unable decode: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Create a user in the database with the email
	user, err := cfg.db.CreateUser(r.Context(), params.Email)
	if err != nil {
		log.Printf("unable create user: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Format the response
	resp := User{
		ID:        user.ID,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
		Email:     user.Email,
	}

	// Pack the data
	data, err := json.Marshal(resp)
	if err != nil {
		log.Printf("unable to marshal user: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write(data)
}

func cleanChirp(s string) string {
	words := strings.Fields(s)

	// Words to censor/clean
	wordBank := map[string]struct{}{
		"kerfuffle": {},
		"sharbert":  {},
		"fornax":    {},
	}
	for i, w := range words {
		if _, ok := wordBank[strings.ToLower(w)]; ok {
			words[i] = "****"
		}
	}
	return strings.Join(words, " ")
}

// Validates
func validateChirp(body string) error {
	if len(body) > 140 {
		return fmt.Errorf("chirp is too long")
	}
	return nil
}

func formatChirp(c database.Chirp) Chirp {
	return Chirp{
		ID:        c.ID,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
		Body:      c.Body,
		UserID:    c.UserID,
	}
}

func (cfg *apiConfig) handlerPostChirps(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Body   string    `json:"body"`
		UserID uuid.UUID `json:"user_id"`
	}

	// Set the header
	w.Header().Set("Content-Type", "application/json")

	// Decode the json from the request
	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `{"error": "invalid format: %v"}`, err)
		return
	}

	// Validate the chirp
	err = validateChirp(params.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `{"error": "invalid chirp: %v"}`, err)
		return
	}

	// Create the chrip in the database
	chirp, err := cfg.db.CreateChirp(r.Context(), database.CreateChirpParams{
		Body:   cleanChirp(params.Body), // provide a cleaned chirp
		UserID: params.UserID,
	})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"error": "unable to create chirp: %v"}`, err)
		return
	}

	// Format the response
	resp := formatChirp(chirp)

	// Pack the data
	data, err := json.Marshal(resp)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"error": "unable to marshal data: %v"}`, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write(data)
}

func (cfg *apiConfig) handlerGetChirps(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	chirps, err := cfg.db.GetChirps(r.Context())
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, `{"error": "unable to get chirps: %v"}`, err)
		return
	}

	var resp []Chirp
	for _, chirp := range chirps {
		resp = append(resp, formatChirp(chirp))
	}

	data, err := json.Marshal(resp)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"error": "uanble to marshal data: %v"}`, err)
	}

	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func (cfg *apiConfig) handlerGetChripByID(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	chirpIDStr := r.PathValue("chirpID")
	chirpID, err := uuid.Parse(chirpIDStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `{"error": "invalid chirpID: %v"}`, err)
		return
	}

	chirp, err := cfg.db.GetChirp(r.Context(), chirpID)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, `{"error": "unable to get chirp '%v': %v}`, chirpID, err)
		return
	}

	resp := formatChirp(chirp)
	data, err := json.Marshal(resp)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"error": "unable to marshal data: %v"}`, err)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

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
	apiCfg := apiConfig{
		db:             database.New(db),
		fileServerHits: atomic.Int32{},
		platform:       os.Getenv("PLATFORM"),
	}

	serveMux := http.NewServeMux()

	// Create file server handlers
	fileServerHandler := http.FileServer((http.Dir(fileServerPath)))
	appHandler := http.StripPrefix("/app", fileServerHandler)

	// Attach handlers to the serve mux
	serveMux.Handle("/app/", apiCfg.middlewareMetricsInc(appHandler))

	serveMux.HandleFunc("GET /admin/metrics", apiCfg.handlerGetMetrics)
	serveMux.HandleFunc("POST /admin/reset", apiCfg.handlerPostReset)

	serveMux.HandleFunc("GET /api/healthz", handlerHealthz)
	serveMux.HandleFunc("POST /api/users", apiCfg.handlerPostUsers)
	serveMux.HandleFunc("POST /api/chirps", apiCfg.handlerPostChirps)
	serveMux.HandleFunc("GET /api/chirps", apiCfg.handlerGetChirps)
	serveMux.HandleFunc("GET /api/chirps/{chirpID}", apiCfg.handlerGetChripByID)

	// Create the server at the desired port and attach the serve mux
	server := http.Server{
		Handler: serveMux,
		Addr:    ":" + serverPort,
	}

	// Start the server
	log.Printf("Serving files from %s on port: %s\n", fileServerPath, serverPort)
	log.Fatal(server.ListenAndServe())
}
