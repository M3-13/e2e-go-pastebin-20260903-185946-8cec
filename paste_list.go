package main

import (
	"net/http"
)

// ListHandler lists paste metadata. Implemented in a later ticket; currently a stub.
func ListHandler(s *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusNotImplemented, "not implemented")
	}
}
