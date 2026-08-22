// Package evidence implements a filesystem-backed evidence store. Each
// evidence artifact is written under the session output directory and the
// store tracks what was saved so reports can reference artifacts by hash.
package evidence

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/QYVORA/qyvora-jabari/pkg/models"
)

// Store persists evidence artifacts to disk. It is safe for concurrent use.
type Store struct {
	mu   sync.Mutex
	dir  string
	refs map[string]string // evidence hash -> relative artifact path
}

// ErrNoDirectory is returned when the store has no output directory.
var ErrNoDirectory = errors.New("evidence store has no output directory")

// New creates a store rooted at dir, creating the directory if needed. When
// dir is empty the store is memory-only and Save refuses to write.
func New(dir string) (*Store, error) {
	if dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("creating evidence directory: %w", err)
		}
	}
	return &Store{dir: dir, refs: map[string]string{}}, nil
}

// Dir returns the output directory (possibly empty for memory-only stores).
func (s *Store) Dir() string { return s.dir }

// Save writes an artifact to the store and returns an Evidence record that
// references it. The artifact filename is derived from the evidence hash so
// identical payloads are stored once.
func (s *Store) Save(_ context.Context, kind, source string, data []byte) (models.Evidence, error) {
	ev := models.Evidence{
		ID:        models.EvidenceID("ev"),
		Kind:      kind,
		Source:    source,
		Data:      data,
		Hash:      models.HashContent(data),
		Timestamp: time.Now().UTC(),
	}
	if data != nil {
		ev.Content = string(data)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.dir != "" {
		rel := filepath.Join(kind, ev.Hash+".txt")
		path := filepath.Join(s.dir, rel)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				return ev, err
			}
			if err := os.WriteFile(path, data, 0o600); err != nil {
				return ev, err
			}
		}
		s.refs[ev.Hash] = rel
	}
	return ev, nil
}

// Resolve returns the on-disk path for a saved evidence hash, if it exists.
func (s *Store) Resolve(hash string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rel, ok := s.refs[hash]
	if !ok || s.dir == "" {
		return "", false
	}
	return filepath.Join(s.dir, rel), true
}
