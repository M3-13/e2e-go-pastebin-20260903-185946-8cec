package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func doCreate(t *testing.T, body []byte) *http.Response {
	t.Helper()
	store := NewStore()
	srv := httptest.NewServer(newRouter(store))
	t.Cleanup(srv.Close)

	req, err := http.NewRequest(http.MethodPost, srv.URL+"/pastes", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("NewRequest failed: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST /pastes failed: %v", err)
	}
	t.Cleanup(func() { resp.Body.Close() })
	return resp
}

func TestCreateReturns201With32HexID(t *testing.T) {
	resp := doCreate(t, []byte(`{"content":"hello world"}`))

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var paste Paste
	if err := json.Unmarshal(body, &paste); err != nil {
		t.Fatalf("invalid JSON response %q: %v", body, err)
	}
	if len(paste.ID) != 32 {
		t.Fatalf("expected 32-char id, got %q (len %d)", paste.ID, len(paste.ID))
	}
	if _, err := hex.DecodeString(paste.ID); err != nil {
		t.Fatalf("id %q is not valid hex: %v", paste.ID, err)
	}
	if paste.Content != "hello world" {
		t.Fatalf("expected content echoed back, got %q", paste.Content)
	}
}

func TestCreateMissingContentReturns400(t *testing.T) {
	resp := doCreate(t, []byte(`{"content":""}`))
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty content, got %d", resp.StatusCode)
	}

	resp = doCreate(t, []byte(`{}`))
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing content, got %d", resp.StatusCode)
	}
}

func TestCreateInvalidJSONReturns400(t *testing.T) {
	resp := doCreate(t, []byte(`{"content": `))
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid JSON, got %d", resp.StatusCode)
	}
}

func TestCreateNegativeExpiryReturns400(t *testing.T) {
	resp := doCreate(t, []byte(`{"content":"x","expires_in_seconds":-1}`))
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for negative expiry, got %d", resp.StatusCode)
	}
}

func TestCreateBodyOverLimitReturns413(t *testing.T) {
	large := `{"content":"` + strings.Repeat("a", maxBodyBytes) + `"}`
	resp := doCreate(t, []byte(large))
	if resp.StatusCode != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413 for oversized body, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var decoded map[string]string
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("expected JSON error body, got %q: %v", body, err)
	}
	if decoded["error"] == "" {
		t.Fatalf("expected non-empty error message, got %q", decoded["error"])
	}
}
