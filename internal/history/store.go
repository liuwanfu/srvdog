package history

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/liuwanfu/srvdog/internal/model"
)

type Store struct {
	Dir string
	mu  sync.Mutex
}

func (s *Store) Append(sample model.Sample) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := os.MkdirAll(s.Dir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(s.Dir, sample.Timestamp.UTC().Format("2006-01-02")+".jsonl")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()

	encoded, err := json.Marshal(sample)
	if err != nil {
		return err
	}
	_, err = file.Write(append(encoded, '\n'))
	return err
}

func (s *Store) ReadRange(start, end time.Time) ([]model.Sample, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entries, err := os.ReadDir(s.Dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)

	out := make([]model.Sample, 0)
	for _, name := range names {
		day, ok := parseDayFile(name)
		if !ok {
			continue
		}
		if day.After(end.UTC()) || day.Add(24*time.Hour).Before(start.UTC()) {
			continue
		}
		file, err := os.Open(filepath.Join(s.Dir, name))
		if err != nil {
			return nil, err
		}
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			var sample model.Sample
			if err := json.Unmarshal(scanner.Bytes(), &sample); err != nil {
				file.Close()
				return nil, err
			}
			ts := sample.Timestamp.UTC()
			if (ts.Equal(start.UTC()) || ts.After(start.UTC())) && (ts.Equal(end.UTC()) || ts.Before(end.UTC())) {
				out = append(out, sample)
			}
		}
		if err := scanner.Err(); err != nil {
			file.Close()
			return nil, err
		}
		file.Close()
	}
	return out, nil
}

func (s *Store) Clear() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	entries, err := os.ReadDir(s.Dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if _, ok := parseDayFile(entry.Name()); !ok {
			continue
		}
		if err := os.Remove(filepath.Join(s.Dir, entry.Name())); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) CleanupExpired(cutoff time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	entries, err := os.ReadDir(s.Dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	cutoff = cutoff.UTC().Truncate(24 * time.Hour)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		day, ok := parseDayFile(entry.Name())
		if !ok {
			continue
		}
		if day.Before(cutoff) {
			if err := os.Remove(filepath.Join(s.Dir, entry.Name())); err != nil {
				return err
			}
		}
	}
	return nil
}

func parseDayFile(name string) (time.Time, bool) {
	if !strings.HasSuffix(name, ".jsonl") || len(name) != len("2006-01-02.jsonl") {
		return time.Time{}, false
	}
	day, err := time.Parse("2006-01-02", strings.TrimSuffix(name, ".jsonl"))
	if err != nil {
		return time.Time{}, false
	}
	return day.UTC(), true
}
