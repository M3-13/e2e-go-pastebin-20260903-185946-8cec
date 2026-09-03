package main

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"
)

// maxBodyBytes caps the request body before it is fully read.
const maxBodyBytes = 1 << 20

// createRequest is the JSON body accepted by POST /pastes.
type createRequest struct {
	Content          string `json:"content"`
	Language         string `json:"language"`
	ExpiresInSeconds int64  `json:"expires_in_seconds"`
}

// CreateHandler returns a handler for POST /pastes.
func CreateHandler(s *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

		body, err := io.ReadAll(r.Body)
		if err != nil {
			var maxErr *http.MaxBytesError
			if errors.As(err, &maxErr) {
				writeError(w, http.StatusRequestEntityTooLarge, "request body too large")
				return
			}
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		var req createRequest
		if err := json.Unmarshal(body, &req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON")
			return
		}

		if req.Content == "" {
			writeError(w, http.StatusBadRequest, "content is required")
			return
		}

		if req.ExpiresInSeconds < 0 {
			writeError(w, http.StatusBadRequest, "expires_in_seconds must not be negative")
			return
		}

		paste, err := s.Create(req.Content, req.Language, time.Duration(req.ExpiresInSeconds)*time.Second)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to create paste")
			return
		}

		writeJSON(w, http.StatusCreated, paste)
	}
}
