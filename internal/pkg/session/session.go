package session

import (
	"math/rand"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Store interface {
	New(clientUUID uuid.UUID, sequenceLength uint16) error
	Get(clientUUID uuid.UUID) (Session, error)
	Set(clientUUID uuid.UUID, session Session) error
	Clear(clientUUID uuid.UUID) error
}

type Session struct {
	Sequence Sequence
	Ack      uint16
	Window   uint16
}

type MemoryStore struct {
	sessions map[uuid.UUID]Session
	mu       sync.RWMutex
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		sessions: make(map[uuid.UUID]Session),
	}
}

func (p *MemoryStore) New(clientUUID uuid.UUID, sequenceLength uint16) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, ok := p.sessions[clientUUID]; ok {
		return ErrSessionAlreadyExists
	}
	p.sessions[clientUUID] = Session{
		Sequence: make([]*uint32, sequenceLength),
	}
	r := rand.New(rand.NewSource(time.Now().Unix()))
	for i := uint16(0); i < sequenceLength; i++ {
		x := r.Uint32()
		p.sessions[clientUUID].Sequence[i] = &x
	}
	return nil
}

func (p *MemoryStore) Get(clientUUID uuid.UUID) (Session, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if sess, ok := p.sessions[clientUUID]; ok {
		return sess, nil
	}
	return Session{}, ErrSessionNotFound
}

func (p *MemoryStore) Set(clientUUID uuid.UUID, session Session) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, ok := p.sessions[clientUUID]; !ok {
		return ErrSessionNotFound
	}
	p.sessions[clientUUID] = session
	return nil
}

func (p *MemoryStore) Clear(clientUUID uuid.UUID) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, ok := p.sessions[clientUUID]; !ok {
		return ErrSessionNotFound
	}
	delete(p.sessions, clientUUID)
	return nil
}
