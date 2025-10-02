package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/evanwiseman/chirpy/internal/auth"
	"github.com/evanwiseman/chirpy/internal/database"
	"github.com/evanwiseman/chirpy/internal/models"
)

func (cfg *APIConfig) HandlerPostUsers(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Email    string `json:"email"`
		Password string `json:"password"`
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

	// Hash the password
	hashed_password, err := auth.HashPassword(params.Password)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"error": "unable to hash password: %v"}`, err)
		return
	}

	// Create a user in the database with the email
	user, err := cfg.DB.CreateUser(r.Context(), database.CreateUserParams{
		Email:          params.Email,
		HashedPassword: hashed_password,
	})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"error": "unable to create user: %v"}`, err)
		return
	}

	// Format the response
	resp, err := models.FormatUser(user)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"error": "unable to format response: %v"}`, err)
		return
	}

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

func (cfg *APIConfig) HandlerLogin(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Email            string `json:"email"`
		Password         string `json:"password"`
		ExpiresInSeconds int    `json:"expires_in_seconds"`
	}

	decoder := json.NewDecoder(r.Body)
	var params parameters
	err := decoder.Decode(&params)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `{"error": "invalid format: %v"}`, err)
		return
	}

	// If not specified set to 1 hour
	if params.ExpiresInSeconds == 0 {
		params.ExpiresInSeconds = 3600
	}
	// Clamp to 1 hour if greater than 1 hour
	params.ExpiresInSeconds = min(params.ExpiresInSeconds, 3600)

	// Find a user with the specified email
	user, err := cfg.DB.GetUserByEmail(r.Context(), params.Email)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// Validate their credentials
	ok, err := auth.CheckPasswordHash(params.Password, user.HashedPassword)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"error": "hashing failed: %v"}`, err)
		return
	}
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// Generate the token
	expiresIn := time.Duration(params.ExpiresInSeconds) * time.Second
	token, err := auth.MakeJWT(user.ID, cfg.JWTSecret, expiresIn)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"error": "couldn't generate token: %v"}`, err)
	}

	// Format a response with the token
	resp, err := models.FormatUserWithToken(user, token)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"error": "unable to format response: %v"}`, err)
		return
	}

	data, err := json.Marshal(resp)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"error": "hashing failed: %v"}`, err)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(data)
}
