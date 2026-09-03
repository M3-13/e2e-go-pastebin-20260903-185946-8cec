package main

import (
	"net/http"
)

// GetHandler retrieves a paste by id. Implemented in a later ticket; currently a stub.
func GetHandler(s *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusNotImplemented, "not implemented")
	}
}
