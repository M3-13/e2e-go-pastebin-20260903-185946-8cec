package main

import (
	"net/http"
)

// DeleteHandler deletes a paste by id. It answers 204 with an empty body when
// the paste existed and had not expired, and 404 with a JSON error when the id
// is unknown or the paste has expired.
func DeleteHandler(s *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if !s.Delete(id) {
			writeError(w, http.StatusNotFound, "paste not found")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
