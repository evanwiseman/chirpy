package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/evanwiseman/chirpy/internal/auth"
	"github.com/evanwiseman/chirpy/internal/database"
	"github.com/google/uuid"
)

const (
	userUpgradedString = "user.upgraded"
)

func (cfg *APIConfig) HandlerUpgradeUserChirpyRed(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Validate the api key
	apiKey, err := auth.GetAPIKey(r.Header)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintf(w, `{"error": "unable to validate api key: %v"}`, err)
		return
	}

	if apiKey != cfg.PolkaKey {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintf(w, `{"error": "invalid api key: %v"}`, err)
		return
	}

	// Decode the body
	decoder := json.NewDecoder(r.Body)
	params := struct {
		Data struct {
			UserID uuid.UUID `json:"user_id"`
		} `json:"data"`
		Event string `json:"event"`
	}{}
	err = decoder.Decode(&params)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"error": "invalid format: %v"}`, err)
		return
	}

	// Validate event is user upgrade
	if params.Event != userUpgradedString {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	_, err = cfg.DB.UpgradeUserChirpyRed(r.Context(), database.UpgradeUserChirpyRedParams{
		ID:          params.Data.UserID,
		IsChirpyRed: sql.NullBool{Bool: true, Valid: true},
	})
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, `{"error": "user not found: %v, %v"}`, err, params.Data.UserID)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
