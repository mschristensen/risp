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
	Set(clientUUID uuid.UUID, ack, window uint16) error
	Clear(clientUUID uuid.UUID) error
}

type Session struct {
	Sequence []uint32
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
		Sequence: make([]uint32, sequenceLength),
	}
	r := rand.New(rand.NewSource(time.Now().Unix()))
	for i := uint16(0); i < sequenceLength; i++ {
		p.sessions[clientUUID].Sequence[i] = r.Uint32()
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

func (p *MemoryStore) Set(clientUUID uuid.UUID, ack, window uint16) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, ok := p.sessions[clientUUID]; !ok {
		return ErrSessionNotFound
	}
	cpy := p.sessions[clientUUID]
	cpy.Ack = ack
	cpy.Window = window
	p.sessions[clientUUID] = cpy
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
