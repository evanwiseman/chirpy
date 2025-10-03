package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/evanwiseman/chirpy/internal/auth"
	"github.com/evanwiseman/chirpy/internal/database"
	"github.com/evanwiseman/chirpy/internal/models"
	"github.com/google/uuid"
)

func cleanChirpBody(s string) string {
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
func validateChirpBody(body string) error {
	if len(body) > 140 {
		return fmt.Errorf("chirp is too long")
	}
	return nil
}

func (cfg *APIConfig) HandlerPostChirps(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Decode the json from the request
	decoder := json.NewDecoder(r.Body)
	params := struct {
		Body string `json:"body"`
	}{}
	err := decoder.Decode(&params)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `{"error": "invalid format: %v"}`, err)
		return
	}

	// Get the access token
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintf(w, `{"error": "couldn't get bearer token: %v"}`, err)
		return
	}

	// Validate the access token
	userID, err := auth.ValidateJWT(token, cfg.JWTSecret)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintf(w, `{"error": "couldn't validate jwt: %v"}`, err)
		return
	}

	// Validate the chirp
	err = validateChirpBody(params.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `{"error": "invalid chirp: %v"}`, err)
		return
	}

	// Create the chrip in the database
	dbChirp, err := cfg.DB.CreateChirp(r.Context(), database.CreateChirpParams{
		Body:   cleanChirpBody(params.Body), // provide a cleaned chirp
		UserID: userID,
	})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"error": "unable to create chirp: %v"}`, err)
		return
	}

	// Format the response
	resp := models.FormatChirp(dbChirp)

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

func (cfg *APIConfig) HandlerGetChirps(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var (
		dbChirps []database.Chirp
		err      error
	)

	if r.URL.Query().Has("author_id") { // Query by user id
		userID, err := uuid.Parse(r.URL.Query().Get("author_id"))
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, `{"error": "unable to parse author id: %v"}`, err)
			return
		}
		dbChirps, err = cfg.DB.GetChripsByUserID(r.Context(), userID)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, `{"error": "unable to find user %v: %v"}`, userID, err)
			return
		}
	} else { // Get all chirps
		dbChirps, err = cfg.DB.GetChirps(r.Context())
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, `{"error": "unable to get chirps: %v"}`, err)
			return
		}
	}

	// Filter in memory for sort dir asc or desc
	if r.URL.Query().Has("sort") {
		sort := r.URL.Query().Get("sort")
		var direction int
		switch sort {
		case "asc":
			direction = 1
		case "desc":
			direction = -1
		default:
			direction = 0
		}
		slices.SortFunc(dbChirps, func(a database.Chirp, b database.Chirp) int {
			return a.CreatedAt.Compare(b.CreatedAt) * direction
		})
	}
	// Format a response
	var resp []models.Chirp
	for _, dbChirp := range dbChirps {
		resp = append(resp, models.FormatChirp(dbChirp))
	}

	// Pack response
	data, err := json.Marshal(resp)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"error": "uanble to marshal data: %v"}`, err)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func (cfg *APIConfig) HandlerGetChripByID(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Get the chirp
	chirpIDStr := r.PathValue("chirpID")
	chirpID, err := uuid.Parse(chirpIDStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `{"error": "invalid chirpID: %v"}`, err)
		return
	}
	dbChirp, err := cfg.DB.GetChirp(r.Context(), chirpID)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, `{"error": "unable to get chirp '%v': %v}`, chirpID, err)
		return
	}

	// Format a response
	resp := models.FormatChirp(dbChirp)

	// Pack data
	data, err := json.Marshal(resp)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"error": "unable to marshal data: %v"}`, err)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func (cfg *APIConfig) HandlerDeleteChirpByID(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Get the acces token
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintf(w, `{"error": "malformed access token: %v"}`, err)
		return
	}

	// Validate the access token
	userID, err := auth.ValidateJWT(token, cfg.JWTSecret)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintf(w, `{"error": "invalid access token: %v"}`, err)
		return
	}

	// Get the chirp
	chirpIDStr := r.PathValue("chirpID")
	chirpID, err := uuid.Parse(chirpIDStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `{"error": "invalid chirpID: %v"}`, err)
		return
	}
	dbChirp, err := cfg.DB.GetChirp(r.Context(), chirpID)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, `{"error": "unable to get chirp '%v': %v}`, chirpID, err)
		return
	}

	// Check that the requester is the author
	if userID != dbChirp.UserID {
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprintf(w, `{"error": "forbidden, unable to delete chirp: requester is not owner"}`)
		return
	}

	// Delete the chirp
	err = cfg.DB.DeleteChirp(r.Context(), dbChirp.ID)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, `{"error": "chirp not found: %v"}`, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
