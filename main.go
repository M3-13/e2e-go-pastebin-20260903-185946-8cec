package main

import (
	"log"
	"net/http"
	"os"
)

// healthHandler responds to GET /health with {"status":"ok"}.
func healthHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// newRouter builds the HTTP handler with all registered routes.
func newRouter(s *Store) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", healthHandler)
	mux.HandleFunc("POST /pastes", CreateHandler(s))
	mux.HandleFunc("GET /pastes", ListHandler(s))
	mux.HandleFunc("GET /pastes/{id}", GetHandler(s))
	mux.HandleFunc("DELETE /pastes/{id}", DeleteHandler(s))
	return mux
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	store := NewStore()
	router := newRouter(store)

	addr := ":" + port
	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatal(err)
	}
}
