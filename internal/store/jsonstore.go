package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/aleka7sk/booking-os/internal/domain"
)

type JSONStore struct {
	mu    sync.RWMutex
	path  string
	state domain.State
}

type Seeder func() (domain.State, error)

func Open(path string, seed Seeder) (*JSONStore, error) {
	s := &JSONStore{path: path}
	if err := s.load(seed); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *JSONStore) load(seed Seeder) error {
	data, err := os.ReadFile(s.path)
	if err == nil {
		if err := json.Unmarshal(data, &s.state); err != nil {
			return fmt.Errorf("decode store: %w", err)
		}
		return nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("read store: %w", err)
	}
	state, err := seed()
	if err != nil {
		return fmt.Errorf("seed store: %w", err)
	}
	s.state = state
	return s.persistLocked()
}

func (s *JSONStore) View(fn func(domain.State) error) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return fn(cloneState(s.state))
}

func (s *JSONStore) Update(fn func(*domain.State) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	working := cloneState(s.state)
	if err := fn(&working); err != nil {
		return err
	}
	previous := s.state
	s.state = working
	if err := s.persistLocked(); err != nil {
		s.state = previous
		return err
	}
	return nil
}

func (s *JSONStore) Snapshot() domain.State {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneState(s.state)
}

func (s *JSONStore) persistLocked() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("create data directory: %w", err)
	}
	data, err := json.MarshalIndent(s.state, "", "  ")
	if err != nil {
		return fmt.Errorf("encode store: %w", err)
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("write temporary store: %w", err)
	}
	if err := os.Rename(tmp, s.path); err != nil {
		return fmt.Errorf("replace store: %w", err)
	}
	return nil
}

func cloneState(in domain.State) domain.State {
	data, err := json.Marshal(in)
	if err != nil {
		panic(err)
	}
	var out domain.State
	if err := json.Unmarshal(data, &out); err != nil {
		panic(err)
	}
	return out
}
