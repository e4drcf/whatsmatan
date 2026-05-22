package chats

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// Entry is one known chat (DM or group) the bot has observed.
type Entry struct {
	JID      string    `json:"jid"`
	Name     string    `json:"name"`
	Kind     string    `json:"kind"` // "dm" | "group" | "lid"
	LastSeen time.Time `json:"last_seen"`
}

type Store struct {
	path string
	mu   sync.RWMutex
	data map[string]Entry
}

func New(path string) (*Store, error) {
	s := &Store{path: path, data: make(map[string]Entry)}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) load() error {
	b, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if len(b) == 0 {
		return nil
	}
	var list []Entry
	if err := json.Unmarshal(b, &list); err != nil {
		return err
	}
	for _, e := range list {
		s.data[e.JID] = e
	}
	return nil
}

// Upsert merges a chat record. Name and LastSeen overwrite only when non-zero.
func (s *Store) Upsert(e Entry) error {
	s.mu.Lock()
	cur, ok := s.data[e.JID]
	if !ok {
		cur = e
	} else {
		if e.Name != "" {
			cur.Name = e.Name
		}
		if e.Kind != "" {
			cur.Kind = e.Kind
		}
		if !e.LastSeen.IsZero() && e.LastSeen.After(cur.LastSeen) {
			cur.LastSeen = e.LastSeen
		}
	}
	s.data[e.JID] = cur
	list := s.snapshot()
	s.mu.Unlock()
	return s.persist(list)
}

func (s *Store) snapshot() []Entry {
	out := make([]Entry, 0, len(s.data))
	for _, e := range s.data {
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool {
		if !out[i].LastSeen.Equal(out[j].LastSeen) {
			return out[i].LastSeen.After(out[j].LastSeen)
		}
		return out[i].Name < out[j].Name
	})
	return out
}

// All returns a snapshot sorted by recency desc, name asc.
func (s *Store) All() []Entry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.snapshot()
}

func (s *Store) persist(list []Entry) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}
