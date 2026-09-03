package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func getRequest(t *testing.T, url string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("failed to build GET request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET %s failed: %v", url, err)
	}
	return resp
}

func TestGetHandlerExistingReturns200(t *testing.T) {
	store := NewStore()
	p, _ := store.Create("hello world", "go", time.Hour)
	srv := httptest.NewServer(newRouter(store))
	defer srv.Close()

	resp := getRequest(t, srv.URL+"/pastes/"+p.ID)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read body: %v", err)
	}

	var got Paste
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("invalid JSON response %q: %v", body, err)
	}

	if got.ID != p.ID {
		t.Fatalf("expected id %q, got %q", p.ID, got.ID)
	}
	if got.Content != "hello world" {
		t.Fatalf("expected content %q, got %q", "hello world", got.Content)
	}
	if got.Language != "go" {
		t.Fatalf("expected language %q, got %q", "go", got.Language)
	}
	if got.CreatedAt.IsZero() {
		t.Fatalf("expected non-zero created_at")
	}
	if got.ExpiresAt.IsZero() {
		t.Fatalf("expected non-zero expires_at")
	}
}

func TestGetHandlerUnknownReturns404(t *testing.T) {
	store := NewStore()
	srv := httptest.NewServer(newRouter(store))
	defer srv.Close()

	resp := getRequest(t, srv.URL+"/pastes/does-not-exist")
	defer resp.Body.Close()
	assertErrorJSON(t, resp, http.StatusNotFound)
}

func TestGetHandlerExpiredReturns404(t *testing.T) {
	store := NewStore()
	p, _ := store.Create("hello", "text", time.Second)
	srv := httptest.NewServer(newRouter(store))
	defer srv.Close()

	time.Sleep(1100 * time.Millisecond)

	resp := getRequest(t, srv.URL+"/pastes/"+p.ID)
	defer resp.Body.Close()
	assertErrorJSON(t, resp, http.StatusNotFound)
}
