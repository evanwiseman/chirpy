package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/evanwiseman/chirpy/internal/auth"
)

func (cfg *APIConfig) HandlerRefresh(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Get the refresh token
	refreshToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `{"error": "invalid header: %v}`, err)
		return
	}

	// Find the refresh token in the database
	dbRefreshToken, err := cfg.DB.GetRefreshToken(r.Context(), refreshToken)
	// Token doesn't exist
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintf(w, `{"error": "unauthorized access, token doesn't exist: %v"}`, err)
		return
	}
	// Token has expired
	if time.Now().Compare(dbRefreshToken.ExpiresAt) > 0 {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintf(w, `{"error": "unauthorized access, token has expired: %v"}`, err)
		return
	}
	// Token is revoked
	if dbRefreshToken.RevokedAt.Valid && time.Now().Compare(dbRefreshToken.RevokedAt.Time) > 0 {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintf(w, `{"error": "unauthorized access, token has been revoked: %v"}`, err)
		return
	}

	// Get the expiration duration
	expiresAt := dbRefreshToken.ExpiresAt.Sub(dbRefreshToken.CreatedAt)
	// Generate a new access token
	token, err := auth.MakeJWT(dbRefreshToken.UserID, cfg.JWTSecret, expiresAt)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"error": "unable to get access token: %v"}`, err)
		return
	}

	resp := struct {
		Token string `json:"token"`
	}{
		Token: token,
	}

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

func (cfg *APIConfig) HandlerRevoke(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Get the refresh token
	refreshToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `{"error": "invalid header: %v}`, err)
		return
	}

	// Revoke the token
	err = cfg.DB.RevokeRefreshToken(r.Context(), refreshToken)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, `{"error": "unable to revoke token, not found: %v"}`, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
