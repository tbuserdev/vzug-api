package state

import (
	"sync"
	"time"
)

type Snapshot struct {
	Known       bool      `json:"known"`
	Visible     bool      `json:"visible"`
	LastUpdated time.Time `json:"last_updated"`
	LastAction  string    `json:"last_action"`
	LastError   string    `json:"last_error,omitempty"`
}

type Store struct {
	mu       sync.RWMutex
	snapshot Snapshot
}

func New() *Store {
	return &Store{
		snapshot: Snapshot{
			Known:       false,
			Visible:     false,
			LastUpdated: time.Now().UTC(),
			LastAction:  "initialized",
		},
	}
}

func (s *Store) SetVisible(visible bool, action string) Snapshot {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.snapshot.Visible = visible
	s.snapshot.Known = true
	s.snapshot.LastUpdated = time.Now().UTC()
	s.snapshot.LastAction = action
	s.snapshot.LastError = ""
	return s.snapshot
}

func (s *Store) SetError(action string, err error) Snapshot {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.snapshot.LastUpdated = time.Now().UTC()
	s.snapshot.LastAction = action
	s.snapshot.LastError = err.Error()
	return s.snapshot
}

func (s *Store) Snapshot() Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.snapshot
}

func Payload(visible bool) string {
	if visible {
		return "ON"
	}
	return "OFF"
}
