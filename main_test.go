package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthEndpoint(t *testing.T) {
	store := NewStore()
	srv := httptest.NewServer(newRouter(store))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/health")
	if err != nil {
		t.Fatalf("GET /health failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var decoded map[string]string
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("invalid JSON body %q: %v", body, err)
	}
	if decoded["status"] != "ok" {
		t.Fatalf("expected status ok, got %q", decoded["status"])
	}
}

func TestPasteRoutesAreWired(t *testing.T) {
	store := NewStore()
	srv := httptest.NewServer(newRouter(store))
	defer srv.Close()

	// The paste routes are declared and reachable. A registered route always
	// answers through its handler, whose errors are JSON ({"error": ...}) even
	// when it legitimately answers 404 (e.g. an unknown id). An unregistered
	// route falls through to the mux's plain-text 404, so a non-JSON 404 is
	// what proves a route is missing.
	for _, tc := range []struct {
		method string
		path   string
	}{
		{"POST", "/pastes"},
		{"GET", "/pastes"},
		{"GET", "/pastes/abc"},
		{"DELETE", "/pastes/abc"},
	} {
		req, _ := http.NewRequest(tc.method, srv.URL+tc.path, nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("%s %s failed: %v", tc.method, tc.path, err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode == http.StatusNotFound && resp.Header.Get("Content-Type") != "application/json" {
			t.Fatalf("%s %s returned a plain 404 (route not registered): %q", tc.method, tc.path, body)
		}
	}
}
