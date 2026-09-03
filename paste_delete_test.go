package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func deleteRequest(t *testing.T, url string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		t.Fatalf("failed to build DELETE request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("DELETE %s failed: %v", url, err)
	}
	return resp
}

func assertErrorJSON(t *testing.T, resp *http.Response, status int) {
	t.Helper()
	if resp.StatusCode != status {
		t.Fatalf("expected status %d, got %d", status, resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
		t.Fatalf("expected application/json content-type, got %q", ct)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read body: %v", err)
	}
	var decoded map[string]string
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("invalid JSON body %q: %v", body, err)
	}
	if decoded["error"] == "" {
		t.Fatalf("expected non-empty error message, got %q", body)
	}
}

func TestDeleteHandlerExistingReturns204(t *testing.T) {
	store := NewStore()
	p, _ := store.Create("hello", "text", time.Hour)
	srv := httptest.NewServer(newRouter(store))
	defer srv.Close()

	resp := deleteRequest(t, srv.URL+"/pastes/"+p.ID)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read body: %v", err)
	}
	if len(body) != 0 {
		t.Fatalf("expected empty body, got %q", body)
	}
}

func TestDeleteHandlerSecondDeleteReturns404(t *testing.T) {
	store := NewStore()
	p, _ := store.Create("hello", "text", time.Hour)
	srv := httptest.NewServer(newRouter(store))
	defer srv.Close()

	resp := deleteRequest(t, srv.URL+"/pastes/"+p.ID)
	resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("first delete: expected 204, got %d", resp.StatusCode)
	}

	resp = deleteRequest(t, srv.URL+"/pastes/"+p.ID)
	defer resp.Body.Close()
	assertErrorJSON(t, resp, http.StatusNotFound)
}

func TestDeleteHandlerUnknownReturns404(t *testing.T) {
	store := NewStore()
	srv := httptest.NewServer(newRouter(store))
	defer srv.Close()

	resp := deleteRequest(t, srv.URL+"/pastes/does-not-exist")
	defer resp.Body.Close()
	assertErrorJSON(t, resp, http.StatusNotFound)
}

func TestDeleteHandlerExpiredReturns404(t *testing.T) {
	store := NewStore()
	p, _ := store.Create("hello", "text", time.Second)
	srv := httptest.NewServer(newRouter(store))
	defer srv.Close()

	time.Sleep(1100 * time.Millisecond)

	resp := deleteRequest(t, srv.URL+"/pastes/"+p.ID)
	defer resp.Body.Close()
	assertErrorJSON(t, resp, http.StatusNotFound)
}
