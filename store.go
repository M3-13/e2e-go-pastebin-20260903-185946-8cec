package main

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sort"
	"sync"
	"time"
)

// defaultExpiry is the fallback lifetime applied when expiresIn <= 0.
const defaultExpiry = 24 * time.Hour

// Paste is a single stored paste, including its content.
type Paste struct {
	ID        string    `json:"id"`
	Content   string    `json:"content"`
	Language  string    `json:"language,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// PasteMeta is the metadata-only view of a paste, without its content.
type PasteMeta struct {
	ID        string    `json:"id"`
	Language  string    `json:"language,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// Store is an in-memory, mutex-guarded paste repository.
type Store struct {
	mu     sync.RWMutex
	pastes map[string]Paste
}

// NewStore returns an empty, ready-to-use Store.
func NewStore() *Store {
	return &Store{
		pastes: make(map[string]Paste),
	}
}

// Create stores a new paste and returns it. The ID is generated with
// crypto/rand (16 bytes -> 32 hex characters, 128 bits of entropy).
// expiresIn <= 0 means the default 24h lifetime.
func (s *Store) Create(content, language string, expiresIn time.Duration) (Paste, error) {
	id, err := newID()
	if err != nil {
		return Paste{}, err
	}

	now := time.Now()
	expires := now.Add(expiresIn)
	if expiresIn <= 0 {
		expires = now.Add(defaultExpiry)
	}

	p := Paste{
		ID:        id,
		Content:   content,
		Language:  language,
		CreatedAt: now,
		ExpiresAt: expires,
	}

	s.mu.Lock()
	s.pastes[id] = p
	s.mu.Unlock()

	return p, nil
}

// Get returns the paste for id and whether it exists and has not expired.
// An expired paste is removed before returning.
func (s *Store) Get(id string) (Paste, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	p, ok := s.pastes[id]
	if !ok {
		return Paste{}, false
	}
	if time.Now().After(p.ExpiresAt) || time.Now().Equal(p.ExpiresAt) {
		delete(s.pastes, id)
		return Paste{}, false
	}
	return p, true
}

// List returns metadata for every non-expired paste, newest first.
// It always returns a non-nil slice.
func (s *Store) List() []PasteMeta {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	metas := make([]PasteMeta, 0, len(s.pastes))
	for id, p := range s.pastes {
		if now.After(p.ExpiresAt) || now.Equal(p.ExpiresAt) {
			delete(s.pastes, id)
			continue
		}
		metas = append(metas, PasteMeta{
			ID:        p.ID,
			Language:  p.Language,
			CreatedAt: p.CreatedAt,
			ExpiresAt: p.ExpiresAt,
		})
	}

	sort.Slice(metas, func(i, j int) bool {
		return metas[i].CreatedAt.After(metas[j].CreatedAt)
	})

	return metas
}

// Delete removes the paste for id and reports whether it existed and had not
// expired. An expired paste is removed and reported as not present.
func (s *Store) Delete(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	p, ok := s.pastes[id]
	if !ok {
		return false
	}
	if time.Now().After(p.ExpiresAt) || time.Now().Equal(p.ExpiresAt) {
		delete(s.pastes, id)
		return false
	}
	delete(s.pastes, id)
	return true
}

// newID generates a 32-character hex ID from 16 cryptographically random bytes.
func newID() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", errors.New("failed to generate random id")
	}
	return hex.EncodeToString(buf), nil
}
