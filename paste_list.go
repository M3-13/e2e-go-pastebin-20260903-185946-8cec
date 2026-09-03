package main

import (
	"net/http"
)

// ListHandler responds to GET /pastes with the metadata of every non-expired
// paste as a JSON array (an empty list as [], never null).
func ListHandler(s *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, s.List())
	}
}
