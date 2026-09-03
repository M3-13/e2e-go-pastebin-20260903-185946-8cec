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

	// The paste routes are declared and reachable. Every registered handler
	// answers with a JSON body (writeJSON/writeError), whereas an unregistered
	// route falls through to ServeMux's default text/plain 404. So a JSON
	// Content-Type proves the route is wired, regardless of the status code a
	// given handler returns (a stub answers 501, a real handler may answer 404
	// for an unknown id).
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
		resp.Body.Close()
		if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
			t.Fatalf("%s %s is not wired: got Content-Type %q", tc.method, tc.path, ct)
		}
	}
}
