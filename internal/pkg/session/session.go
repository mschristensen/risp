// Package session implements the session state storage.
package session

import (
	"math/rand"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Store provides an API for perforing CRUD operations on client state.
type Store interface {
	New(clientUUID uuid.UUID, sequenceLength uint16) error
	Get(clientUUID uuid.UUID) (Session, error)
	Set(clientUUID uuid.UUID, session Session) error
	Clear(clientUUID uuid.UUID) error
}

// Session captures the current session state of a client.
type Session struct {
	Sequence Sequence
	Ack      uint16
	Window   uint16
}

// MemoryStore is a in-memory implementation of Store.
type MemoryStore struct {
	sessions map[uuid.UUID]Session
	mu       sync.RWMutex
}

// NewMemoryStore creates a new in-memory store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		sessions: make(map[uuid.UUID]Session),
	}
}

// New creates a new session state for the given client uuid,
// and initialises the sequence with the given length with random values.
func (p *MemoryStore) New(clientUUID uuid.UUID, sequenceLength uint16) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, ok := p.sessions[clientUUID]; ok {
		return ErrSessionAlreadyExists
	}
	p.sessions[clientUUID] = Session{
		Sequence: make([]*uint32, sequenceLength),
	}
	r := rand.New(rand.NewSource(time.Now().Unix())) // nolint: gosec // we don't need high security here
	for i := uint16(0); i < sequenceLength; i++ {
		x := r.Uint32()
		p.sessions[clientUUID].Sequence[i] = &x
	}
	return nil
}

// Get returns the session state for the given client uuid.
func (p *MemoryStore) Get(clientUUID uuid.UUID) (Session, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if sess, ok := p.sessions[clientUUID]; ok {
		return sess, nil
	}
	return Session{}, ErrSessionNotFound
}

// Set updates the session state for the given client uuid.
func (p *MemoryStore) Set(clientUUID uuid.UUID, session Session) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, ok := p.sessions[clientUUID]; !ok {
		return ErrSessionNotFound
	}
	p.sessions[clientUUID] = session
	return nil
}

// Clear removes the session state for the given client uuid.
func (p *MemoryStore) Clear(clientUUID uuid.UUID) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, ok := p.sessions[clientUUID]; !ok {
		return ErrSessionNotFound
	}
	delete(p.sessions, clientUUID)
	return nil
}
