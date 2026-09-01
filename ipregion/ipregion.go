// Package ipregion wraps a process-wide ipregion.db searcher.
//
// Query: Find, Name, Areas. Lifecycle: Open, Close, StatusOf.
// Callers compose trees and address→IP resolution themselves.
package ipregion

import (
	"fmt"
	"os"
	"strings"
	"sync"

	gopkg "github.com/penndev/gopkg/ipregion"
)

type Status struct {
	Exists  bool   `json:"exists"`
	Path    string `json:"path"`
	Size    int64  `json:"size"`
	Version string `json:"version,omitempty"`
	Remark  string `json:"remark,omitempty"`
	Areas   int    `json:"areas,omitempty"`
	V4      int    `json:"v4,omitempty"`
	V6      int    `json:"v6,omitempty"`
}

type Area struct {
	ID       uint32 `json:"id"`
	ParentID uint32 `json:"parent_id"`
	Name     string `json:"name"`
}

var (
	mu       sync.Mutex
	searcher *gopkg.Searcher
	dbPath   string
)

// Open opens ipregion.db at path. On error the previously opened DB is left unchanged.
func Open(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return fmt.Errorf("db path required")
	}
	s, err := gopkg.Open(path)
	if err != nil {
		return fmt.Errorf("open ipregion.db: %w", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if searcher != nil {
		_ = searcher.Close()
	}
	searcher = s
	dbPath = path
	return nil
}

// Close releases the cached searcher. Path is kept for Status.
func Close() {
	mu.Lock()
	defer mu.Unlock()
	if searcher != nil {
		_ = searcher.Close()
		searcher = nil
	}
}

func Opened() bool {
	mu.Lock()
	defer mu.Unlock()
	return searcher != nil
}

func StatusOf() Status {
	mu.Lock()
	path := dbPath
	s := searcher
	mu.Unlock()

	out := Status{Path: path}
	if path == "" {
		return out
	}
	fi, err := os.Stat(path)
	if err != nil {
		return out
	}
	out.Exists = true
	out.Size = fi.Size()
	if s != nil {
		m := s.Meta()
		out.Version = m.Version
		out.Remark = m.Remark
		out.Areas = m.Areas
		out.V4 = m.V4
		out.V6 = m.V6
	}
	return out
}

func Find(ip string) (gopkg.Info, error) {
	s := get()
	if s == nil {
		return gopkg.Info{}, fmt.Errorf("ipregion.db not open")
	}
	return s.Find(ip)
}

func Name(id uint32) string {
	s := get()
	if s == nil || id == 0 {
		return ""
	}
	a, ok := s.Area(id)
	if !ok {
		return ""
	}
	return a.Name
}

func Areas(parentID uint32) []Area {
	s := get()
	if s == nil {
		return nil
	}
	src := s.Areas(parentID)
	out := make([]Area, 0, len(src))
	for _, a := range src {
		out = append(out, Area{ID: a.ID, ParentID: a.ParentID, Name: a.Name})
	}
	return out
}

func get() *gopkg.Searcher {
	mu.Lock()
	defer mu.Unlock()
	return searcher
}

// Valid opens path as ipregion.db without replacing the cached searcher.
func Valid(path string) error {
	s, err := gopkg.Open(path)
	if err != nil {
		return err
	}
	return s.Close()
}
