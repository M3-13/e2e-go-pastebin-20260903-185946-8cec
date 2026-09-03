package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func doList(t *testing.T, srv *httptest.Server) []byte {
	t.Helper()
	resp, err := http.Get(srv.URL + "/pastes")
	if err != nil {
		t.Fatalf("GET /pastes failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body failed: %v", err)
	}
	return body
}

func TestListHandlerEmpty(t *testing.T) {
	store := NewStore()
	srv := httptest.NewServer(newRouter(store))
	defer srv.Close()

	body := doList(t, srv)

	var metas []PasteMeta
	if err := json.Unmarshal(body, &metas); err != nil {
		t.Fatalf("invalid JSON body %q: %v", body, err)
	}
	if metas == nil {
		t.Fatalf("expected an empty array [], got null (%q)", body)
	}
	if len(metas) != 0 {
		t.Fatalf("expected 0 items, got %d", len(metas))
	}
}

func TestListHandlerMultipleAndExpiredOmitted(t *testing.T) {
	store := NewStore()

	// One paste that has already expired: must be omitted from the list.
	// Create() treats <=0 as the default 24h lifetime, so force expiry by
	// rewriting its ExpiresAt directly.
	expired, err := store.Create("secret expired", "", time.Hour)
	if err != nil {
		t.Fatalf("create expired paste failed: %v", err)
	}
	store.mu.Lock()
	p := store.pastes[expired.ID]
	p.ExpiresAt = time.Now().Add(-time.Hour)
	store.pastes[expired.ID] = p
	store.mu.Unlock()

	_, err = store.Create("first", "go", time.Hour)
	if err != nil {
		t.Fatalf("create first paste failed: %v", err)
	}
	_, err = store.Create("second", "python", time.Hour)
	if err != nil {
		t.Fatalf("create second paste failed: %v", err)
	}

	srv := httptest.NewServer(newRouter(store))
	defer srv.Close()

	body := doList(t, srv)

	var metas []PasteMeta
	if err := json.Unmarshal(body, &metas); err != nil {
		t.Fatalf("invalid JSON body %q: %v", body, err)
	}

	if len(metas) != 2 {
		t.Fatalf("expected 2 items (expired omitted), got %d", len(metas))
	}
	for _, m := range metas {
		if m.ID == expired.ID {
			t.Fatalf("expired paste %q should be omitted", expired.ID)
		}
	}

	// Newest first.
	if metas[0].CreatedAt.Before(metas[1].CreatedAt) {
		t.Fatalf("expected newest first, got %v then %v", metas[0].CreatedAt, metas[1].CreatedAt)
	}
}

func TestListHandlerNoContentField(t *testing.T) {
	store := NewStore()
	_, err := store.Create("sensitive body", "go", time.Hour)
	if err != nil {
		t.Fatalf("create paste failed: %v", err)
	}

	srv := httptest.NewServer(newRouter(store))
	defer srv.Close()

	body := doList(t, srv)

	if strings.Contains(string(body), `"content"`) {
		t.Fatalf("response must not contain a content field: %q", body)
	}
}
