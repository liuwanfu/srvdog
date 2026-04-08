package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

type Settings struct {
	RetentionDays int `json:"retention_days"`
}

type SettingsStore struct {
	path string
	mu   sync.RWMutex
	data Settings
}

func NewSettingsStore(path string, defaults Settings) (*SettingsStore, error) {
	store := &SettingsStore{
		path: path,
		data: defaults,
	}
	if err := store.load(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *SettingsStore) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return s.saveLocked()
		}
		return err
	}
	return json.Unmarshal(data, &s.data)
}

func (s *SettingsStore) Get() Settings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.data
}

func (s *SettingsStore) SetRetention(days int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.RetentionDays = days
	return s.saveLocked()
}

func (s *SettingsStore) saveLocked() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0o644)
}
