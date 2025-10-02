package handlers

import (
	"fmt"
	"net/http"
)

func (cfg *APIConfig) MiddlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.FileServerHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func (cfg *APIConfig) HandlerGetMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")

	hits := cfg.FileServerHits.Load()
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

func (cfg *APIConfig) HandlerPostReset(w http.ResponseWriter, r *http.Request) {
	// Don't reset if not on development database
	if cfg.Platform != "dev" {
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprintf(w, `{"error": "unable to reset when not no dev database"}`)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	cfg.FileServerHits.Store(0)

	// Attempt to reset the user database
	err := cfg.DB.ResetUsers(r.Context())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"error": "failed to reset user database: %v"}`, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}
