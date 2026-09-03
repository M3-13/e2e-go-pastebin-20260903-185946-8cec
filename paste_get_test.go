package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func getPaste(t *testing.T, srv *httptest.Server, id string) (*http.Response, []byte) {
	t.Helper()
	resp, err := http.Get(srv.URL + "/pastes/" + id)
	if err != nil {
		t.Fatalf("GET /pastes/%s failed: %v", id, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("reading body failed: %v", err)
	}
	return resp, body
}

func TestGetPasteReturnsFullPaste(t *testing.T) {
	store := NewStore()
	p, err := store.Create("print(\"hi\")", "python", 24*time.Hour)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	srv := httptest.NewServer(newRouter(store))
	defer srv.Close()

	resp, body := getPaste(t, srv, p.ID)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d (body %q)", resp.StatusCode, body)
	}

	var got Paste
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("invalid JSON body %q: %v", body, err)
	}
	if got.ID != p.ID {
		t.Errorf("expected id %q, got %q", p.ID, got.ID)
	}
	if got.Content != "print(\"hi\")" {
		t.Errorf("expected content %q, got %q", "print(\"hi\")", got.Content)
	}
	if got.Language != "python" {
		t.Errorf("expected language %q, got %q", "python", got.Language)
	}
	if got.CreatedAt.IsZero() {
		t.Errorf("expected non-zero CreatedAt")
	}
	if got.ExpiresAt.IsZero() {
		t.Errorf("expected non-zero ExpiresAt")
	}
}

func TestGetPasteUnknownID(t *testing.T) {
	store := NewStore()
	srv := httptest.NewServer(newRouter(store))
	defer srv.Close()

	resp, body := getPaste(t, srv, "does-not-exist")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d (body %q)", resp.StatusCode, body)
	}

	var errResp map[string]string
	if err := json.Unmarshal(body, &errResp); err != nil {
		t.Fatalf("invalid JSON error body %q: %v", body, err)
	}
	if _, ok := errResp["error"]; !ok {
		t.Errorf("expected error field in body, got %q", body)
	}
}

func TestGetPasteExpired(t *testing.T) {
	store := NewStore()
	p, err := store.Create("temporary", "", 1*time.Second)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	srv := httptest.NewServer(newRouter(store))
	defer srv.Close()

	time.Sleep(1100 * time.Millisecond)

	resp, body := getPaste(t, srv, p.ID)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 for expired paste, got %d (body %q)", resp.StatusCode, body)
	}

	var errResp map[string]string
	if err := json.Unmarshal(body, &errResp); err != nil {
		t.Fatalf("invalid JSON error body %q: %v", body, err)
	}
	if _, ok := errResp["error"]; !ok {
		t.Errorf("expected error field in body, got %q", body)
	}
}
