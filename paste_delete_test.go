package main

import (
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
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("second delete: expected 404, got %d", resp.StatusCode)
	}
}

func TestDeleteHandlerUnknownReturns404(t *testing.T) {
	store := NewStore()
	srv := httptest.NewServer(newRouter(store))
	defer srv.Close()

	resp := deleteRequest(t, srv.URL+"/pastes/does-not-exist")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

func TestDeleteHandlerExpiredReturns404(t *testing.T) {
	store := NewStore()
	p, _ := store.Create("hello", "text", time.Second)
	srv := httptest.NewServer(newRouter(store))
	defer srv.Close()

	time.Sleep(1100 * time.Millisecond)

	resp := deleteRequest(t, srv.URL+"/pastes/"+p.ID)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 for expired paste, got %d", resp.StatusCode)
	}
}
