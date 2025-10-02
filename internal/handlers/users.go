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

const (
	jwtExpirationInSeconds  = 3600
	refreshExpirationInDays = 60
)

func (cfg *APIConfig) HandlerPostUsers(w http.ResponseWriter, r *http.Request) {
	// Set the header
	w.Header().Set("Content-Type", "application/json")

	// Decode the json from the request
	decoder := json.NewDecoder(r.Body)
	params := struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}{}
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
	resp := models.FormatUser(user, "", "")

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

func (cfg *APIConfig) HandlerPutUsers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Get access token
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintf(w, `{"error": "unable to access token: %v"}`, err)
		return
	}

	// Validate the access token
	userID, err := auth.ValidateJWT(token, cfg.JWTSecret)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintf(w, `{"error": "invalid access token: %v"}`, err)
		return
	}

	// Decode the body into parameters
	decoder := json.NewDecoder(r.Body)
	params := struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}{}
	err = decoder.Decode(&params)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `{"error": "invalid format: %v"}`, err)
		return
	}

	// Hash the password
	hashedPassword, err := auth.HashPassword(params.Password)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"error": "failed to hash password: %v"}`, err)
		return
	}

	// Update the user with the provided information
	newUser, err := cfg.DB.UpdateUser(r.Context(), database.UpdateUserParams{
		ID:             userID,
		Email:          params.Email,
		HashedPassword: hashedPassword,
	})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"error": "unable to update user: %v"}`, err)
	}

	// Format a response
	resp := models.FormatUser(newUser, token, "")

	// Pack response
	data, err := json.Marshal(resp)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"error": "unable to marshal data: %v"}`, err)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func (cfg *APIConfig) HandlerLogin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Decode the body into params
	decoder := json.NewDecoder(r.Body)
	params := struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}{}
	err := decoder.Decode(&params)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `{"error": "invalid format: %v"}`, err)
		return
	}

	// Find a dbUser with the specified email
	dbUser, err := cfg.DB.GetUserByEmail(r.Context(), params.Email)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintf(w, `{"error": "unable to validate credentials, invalid email: %v"}`, err)
		return
	}

	// Validate their credentials
	ok, err := auth.CheckPasswordHash(params.Password, dbUser.HashedPassword)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"error": "hashing failed: %v"}`, err)
		return
	}
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintf(w, `{"error": "unable to validate credentials, invalid password: %v"}`, err)
		return
	}

	// Generate access token
	expiresAt := time.Duration(jwtExpirationInSeconds) * time.Second
	jwtToken, err := auth.MakeJWT(dbUser.ID, cfg.JWTSecret, expiresAt)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"error": "couldn't generate jwt token: %v"}`, err)
		return
	}

	// Generate a refresh token
	refreshToken, err := auth.MakeRefreshToken()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"error": "couldn't generate refresh token %v"}`, err)
		return
	}

	// Add refresh token to database
	expiresAt = time.Duration(refreshExpirationInDays) * time.Hour * 24
	_, err = cfg.DB.CreateRefreshToken(r.Context(), database.CreateRefreshTokenParams{
		Token:     refreshToken,
		UserID:    dbUser.ID,
		ExpiresAt: time.Now().Add(expiresAt),
	})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"error": "couldn't create refresh token in database: %v}`, err)
		return
	}

	// Format a response
	resp := models.FormatUser(dbUser, jwtToken, refreshToken)

	// Pack response
	data, err := json.Marshal(resp)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"error": "unable to marshal data: %v"}`, err)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(data)
}
