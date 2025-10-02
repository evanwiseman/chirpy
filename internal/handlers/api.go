package handlers

import (
	"fmt"
	"net/http"
	"sync/atomic"

	"github.com/evanwiseman/chirpy/internal/database"
)

type APIConfig struct {
	DB             *database.Queries
	FileServerHits atomic.Int32
	Platform       string
	JWTSecret      string
}

func HandlerHealthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "OK")
}
