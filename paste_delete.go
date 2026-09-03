package main

import (
	"net/http"
)

// DeleteHandler deletes a paste by id. Implemented in a later ticket; currently a stub.
func DeleteHandler(s *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusNotImplemented, "not implemented")
	}
}
