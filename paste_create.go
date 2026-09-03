package main

import (
	"net/http"
)

// CreateHandler creates a paste. Implemented in a later ticket; currently a stub.
func CreateHandler(s *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusNotImplemented, "not implemented")
	}
}
