// Package store handles snippet persistence as a single JSON file.
//
// The on-disk format is documented in docs/format.md. Keep that doc and
// this package in sync — external compatibility lives there.
package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const FormatVersion = 1

// Snippet is a single registered phrase / command / prompt fragment.
type Snippet struct {
	Key         string     `json:"key"`
	Value       string     `json:"value"`
	Description string     `json:"description,omitempty"`
	Tags        []string   `json:"tags,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
	LastUsedAt  *time.Time `json:"lastUsedAt,omitempty"`
	UseCount    int        `json:"useCount"`
}

// File is the root JSON document on disk.
type File struct {
	Version  int       `json:"version"`
	Snippets []Snippet `json:"snippets"`
}

// Common errors.
var (
	ErrNotFound = errors.New("snippet not found")
	ErrExists   = errors.New("snippet already exists")
	ErrEmptyKey = errors.New("key must not be empty")
)

// DefaultPath returns the default snippet file location.
//
// PE_DIR overrides everything. Otherwise falls back to ~/.pe/pe.json
// (per the project spec). XDG paths are intentionally NOT consulted —
// keeping a single, stable, easy-to-remember location is a feature.
func DefaultPath() (string, error) {
	if d := os.Getenv("PE_DIR"); d != "" {
		return filepath.Join(d, "pe.json"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".pe", "pe.json"), nil
}

// Load reads and parses the snippet file. Returns an empty File if missing.
func Load(path string) (*File, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &File{Version: FormatVersion}, nil
		}
		return nil, err
	}
	var f File
	if err := json.Unmarshal(b, &f); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if f.Version == 0 {
		f.Version = FormatVersion
	}
	return &f, nil
}

// Save atomically writes the file (write to .tmp + rename).
func Save(path string, f *File) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	if f.Version == 0 {
		f.Version = FormatVersion
	}
	b, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// Get returns the snippet with the given key, or ErrNotFound.
func (f *File) Get(key string) (*Snippet, error) {
	for i := range f.Snippets {
		if f.Snippets[i].Key == key {
			return &f.Snippets[i], nil
		}
	}
	return nil, ErrNotFound
}

// Has reports whether a snippet with the given key exists.
func (f *File) Has(key string) bool {
	_, err := f.Get(key)
	return err == nil
}

// Add inserts a new snippet. Returns ErrExists if the key is taken.
// CreatedAt / UpdatedAt are stamped if zero.
func (f *File) Add(s Snippet) error {
	if strings.TrimSpace(s.Key) == "" {
		return ErrEmptyKey
	}
	if f.Has(s.Key) {
		return ErrExists
	}
	now := time.Now().UTC()
	if s.CreatedAt.IsZero() {
		s.CreatedAt = now
	}
	if s.UpdatedAt.IsZero() {
		s.UpdatedAt = now
	}
	f.Snippets = append(f.Snippets, s)
	return nil
}

// Update replaces the snippet with the given oldKey. If newKey != oldKey,
// it is checked for collisions. UpdatedAt is bumped to now.
func (f *File) Update(oldKey string, newSnippet Snippet) error {
	idx := -1
	for i := range f.Snippets {
		if f.Snippets[i].Key == oldKey {
			idx = i
			break
		}
	}
	if idx == -1 {
		return ErrNotFound
	}
	if newSnippet.Key != oldKey && f.Has(newSnippet.Key) {
		return ErrExists
	}
	if strings.TrimSpace(newSnippet.Key) == "" {
		return ErrEmptyKey
	}
	prev := f.Snippets[idx]
	if newSnippet.CreatedAt.IsZero() {
		newSnippet.CreatedAt = prev.CreatedAt
	}
	if newSnippet.LastUsedAt == nil {
		newSnippet.LastUsedAt = prev.LastUsedAt
	}
	if newSnippet.UseCount == 0 {
		newSnippet.UseCount = prev.UseCount
	}
	newSnippet.UpdatedAt = time.Now().UTC()
	f.Snippets[idx] = newSnippet
	return nil
}

// Delete removes a snippet by key.
func (f *File) Delete(key string) error {
	for i := range f.Snippets {
		if f.Snippets[i].Key == key {
			f.Snippets = append(f.Snippets[:i], f.Snippets[i+1:]...)
			return nil
		}
	}
	return ErrNotFound
}

// Touch records that the snippet was used (copied) at the given time.
// Increments UseCount. No-op (with ErrNotFound) if key is missing.
func (f *File) Touch(key string, at time.Time) error {
	for i := range f.Snippets {
		if f.Snippets[i].Key == key {
			t := at.UTC()
			f.Snippets[i].LastUsedAt = &t
			f.Snippets[i].UseCount++
			return nil
		}
	}
	return ErrNotFound
}

// SortedKeys returns all keys in sorted order.
func (f *File) SortedKeys() []string {
	keys := make([]string, len(f.Snippets))
	for i, s := range f.Snippets {
		keys[i] = s.Key
	}
	sort.Strings(keys)
	return keys
}

// Search returns all snippets whose key, description, value, or tags
// contain the query (case-insensitive). Results are sorted by key.
func (f *File) Search(query string) []Snippet {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		out := make([]Snippet, len(f.Snippets))
		copy(out, f.Snippets)
		sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
		return out
	}
	var hits []Snippet
	for _, s := range f.Snippets {
		if matches(s, q) {
			hits = append(hits, s)
		}
	}
	sort.Slice(hits, func(i, j int) bool { return hits[i].Key < hits[j].Key })
	return hits
}

func matches(s Snippet, lowerQuery string) bool {
	if strings.Contains(strings.ToLower(s.Key), lowerQuery) {
		return true
	}
	if strings.Contains(strings.ToLower(s.Description), lowerQuery) {
		return true
	}
	if strings.Contains(strings.ToLower(s.Value), lowerQuery) {
		return true
	}
	for _, t := range s.Tags {
		if strings.Contains(strings.ToLower(t), lowerQuery) {
			return true
		}
	}
	return false
}
