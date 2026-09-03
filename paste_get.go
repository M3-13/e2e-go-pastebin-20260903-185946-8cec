package main

import (
	"net/http"
)

// GetHandler retrieves a paste by id. It answers 200 with the full paste for a
// known, non-expired id, and 404 with a JSON error otherwise.
func GetHandler(s *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			writeError(w, http.StatusNotFound, "paste not found")
			return
		}
		p, ok := s.Get(id)
		if !ok {
			writeError(w, http.StatusNotFound, "paste not found")
			return
		}
		writeJSON(w, http.StatusOK, p)
	}
}
