package main

import (
	"encoding/hex"
	"regexp"
	"testing"
	"time"
)

func TestCreateGeneratesRandomHexID(t *testing.T) {
	s := NewStore()
	p1, err := s.Create("hello", "text", time.Hour)
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if len(p1.ID) != 32 {
		t.Fatalf("expected 32-char hex id, got %q (len %d)", p1.ID, len(p1.ID))
	}
	if _, err := hex.DecodeString(p1.ID); err != nil {
		t.Fatalf("id is not valid hex: %v", err)
	}

	p2, err := s.Create("hello", "text", time.Hour)
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if p1.ID == p2.ID {
		t.Fatalf("two pastes got the same id %q", p1.ID)
	}
}

func TestCreateSetsExpiry(t *testing.T) {
	s := NewStore()
	before := time.Now()
	p, err := s.Create("hello", "text", 10*time.Minute)
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	expected := before.Add(10 * time.Minute)
	if p.ExpiresAt.Sub(expected) > time.Second {
		t.Fatalf("expected expiry ~%v, got %v", expected, p.ExpiresAt)
	}
}

func TestCreateDefaultExpiry24h(t *testing.T) {
	s := NewStore()
	before := time.Now()
	p, err := s.Create("hello", "", 0)
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	expected := before.Add(24 * time.Hour)
	if p.ExpiresAt.Sub(expected) > time.Second {
		t.Fatalf("expected default 24h expiry ~%v, got %v", expected, p.ExpiresAt)
	}
}

func TestGetReturnsStoredPaste(t *testing.T) {
	s := NewStore()
	p, _ := s.Create("hello", "text", time.Hour)
	got, ok := s.Get(p.ID)
	if !ok {
		t.Fatalf("expected paste %q to be found", p.ID)
	}
	if got.Content != "hello" || got.ID != p.ID {
		t.Fatalf("got unexpected paste: %+v", got)
	}
}

func TestGetMissingIDReturnsFalse(t *testing.T) {
	s := NewStore()
	if _, ok := s.Get("does-not-exist"); ok {
		t.Fatalf("expected missing id to return false")
	}
}

func TestGetExpiredReturnsFalseAndRemoves(t *testing.T) {
	s := NewStore()
	p, _ := s.Create("hello", "text", 20*time.Millisecond)
	time.Sleep(40 * time.Millisecond)

	if _, ok := s.Get(p.ID); ok {
		t.Fatalf("expected expired paste to return false")
	}
	if _, ok := s.Get(p.ID); ok {
		t.Fatalf("expired paste should have been removed")
	}
}

func TestListOnlyNonexpiredNewestFirst(t *testing.T) {
	s := NewStore()
	p1, _ := s.Create("one", "text", time.Hour)
	time.Sleep(2 * time.Millisecond)
	p2, _ := s.Create("two", "text", time.Hour)
	time.Sleep(2 * time.Millisecond)
	expired, _ := s.Create("gone", "text", 20*time.Millisecond)
	time.Sleep(40 * time.Millisecond)

	metas := s.List()
	if len(metas) != 2 {
		t.Fatalf("expected 2 metas, got %d", len(metas))
	}
	if metas[0].ID != p2.ID || metas[1].ID != p1.ID {
		t.Fatalf("expected newest first, got %v", metas)
	}
	for _, m := range metas {
		if m.ID == expired.ID {
			t.Fatalf("expired paste should be excluded from list")
		}
		if m.Language != "text" {
			t.Fatalf("expected language preserved, got %q", m.Language)
		}
	}
}

func TestListEmptyReturnsNonNil(t *testing.T) {
	s := NewStore()
	metas := s.List()
	if metas == nil {
		t.Fatalf("expected non-nil empty slice")
	}
	if len(metas) != 0 {
		t.Fatalf("expected empty slice, got %d", len(metas))
	}
}

func TestDeleteExisting(t *testing.T) {
	s := NewStore()
	p, _ := s.Create("hello", "text", time.Hour)
	if !s.Delete(p.ID) {
		t.Fatalf("expected Delete to return true for existing paste")
	}
	if _, ok := s.Get(p.ID); ok {
		t.Fatalf("expected paste to be gone after Delete")
	}
}

func TestDeleteMissingReturnsFalse(t *testing.T) {
	s := NewStore()
	if s.Delete("does-not-exist") {
		t.Fatalf("expected Delete to return false for missing id")
	}
}

func TestDeleteExpiredReturnsFalse(t *testing.T) {
	s := NewStore()
	p, _ := s.Create("hello", "text", 20*time.Millisecond)
	time.Sleep(40 * time.Millisecond)
	if s.Delete(p.ID) {
		t.Fatalf("expected Delete to return false for expired paste")
	}
}

func TestIDHasEnoughEntropy(t *testing.T) {
	s := NewStore()
	p, _ := s.Create("hello", "text", time.Hour)
	if matched, _ := regexp.MatchString("^[0-9a-f]{32}$", p.ID); !matched {
		t.Fatalf("id %q is not 32 lowercase hex characters", p.ID)
	}
}
