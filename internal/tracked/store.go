package tracked

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

type Store struct {
	path string
	mu   sync.RWMutex
	set  map[string]struct{}
}

func New(path string) (*Store, error) {
	s := &Store{path: path, set: make(map[string]struct{})}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) load() error {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var jids []string
	if len(data) == 0 {
		return nil
	}
	if err := json.Unmarshal(data, &jids); err != nil {
		return err
	}
	for _, j := range jids {
		s.set[j] = struct{}{}
	}
	return nil
}

func (s *Store) Has(jid string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.set[jid]
	return ok
}

func (s *Store) All() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]string, 0, len(s.set))
	for j := range s.set {
		out = append(out, j)
	}
	sort.Strings(out)
	return out
}

func (s *Store) Set(jid string, tracked bool) error {
	s.mu.Lock()
	if tracked {
		s.set[jid] = struct{}{}
	} else {
		delete(s.set, jid)
	}
	jids := make([]string, 0, len(s.set))
	for j := range s.set {
		jids = append(jids, j)
	}
	s.mu.Unlock()
	sort.Strings(jids)
	return s.persist(jids)
}

func (s *Store) persist(jids []string) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(jids, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}
