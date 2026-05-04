// Package store handles snippet persistence as a single JSON file.
//
// The on-disk format is documented in docs/format.md. Keep that doc and
// this package in sync — external compatibility lives there.
package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const FormatVersion = 1

// Snippet is a single registered phrase / command / prompt fragment.
type Snippet struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	Tags      []string  `json:"tags,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// File is the root JSON document on disk.
type File struct {
	Version  int       `json:"version"`
	Snippets []Snippet `json:"snippets"`
}

// DefaultPath returns ~/.config/pe/snippets.json (XDG-aware via XDG_CONFIG_HOME).
func DefaultPath() (string, error) {
	if x := os.Getenv("XDG_CONFIG_HOME"); x != "" {
		return filepath.Join(x, "pe", "snippets.json"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "pe", "snippets.json"), nil
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
	return &f, nil
}

// Save atomically writes the file (write to .tmp + rename).
func Save(path string, f *File) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
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
