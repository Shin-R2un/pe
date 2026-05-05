package store

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestLoadMissingFileReturnsEmpty(t *testing.T) {
	dir := t.TempDir()
	f, err := Load(filepath.Join(dir, "nope.json"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if f.Version != FormatVersion {
		t.Errorf("Version = %d, want %d", f.Version, FormatVersion)
	}
	if len(f.Snippets) != 0 {
		t.Errorf("expected empty file, got %d snippets", len(f.Snippets))
	}
}

func TestRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "pe.json")

	f := &File{Version: FormatVersion}
	if err := f.Add(Snippet{Key: "claude", Value: "claude --x"}); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := f.Add(Snippet{Key: "codex", Value: "codex"}); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := Save(path, f); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if runtime.GOOS != "windows" {
		// Windows: Mode().Perm() reflects the read-only flag, not POSIX bits.
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("stat: %v", err)
		}
		if mode := info.Mode().Perm(); mode != 0o600 {
			t.Errorf("file mode = %o, want 0600", mode)
		}
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(loaded.Snippets) != 2 {
		t.Fatalf("loaded %d snippets, want 2", len(loaded.Snippets))
	}
	got, err := loaded.Get("claude")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Value != "claude --x" {
		t.Errorf("Value = %q", got.Value)
	}
}

func TestAddRejectsDuplicates(t *testing.T) {
	f := &File{}
	if err := f.Add(Snippet{Key: "a", Value: "1"}); err != nil {
		t.Fatal(err)
	}
	err := f.Add(Snippet{Key: "a", Value: "2"})
	if err != ErrExists {
		t.Errorf("err = %v, want ErrExists", err)
	}
}

func TestAddRejectsEmptyKey(t *testing.T) {
	f := &File{}
	if err := f.Add(Snippet{Key: "  ", Value: "x"}); err != ErrEmptyKey {
		t.Errorf("err = %v, want ErrEmptyKey", err)
	}
}

func TestUpdateChangesKey(t *testing.T) {
	f := &File{}
	_ = f.Add(Snippet{Key: "old", Value: "v"})
	err := f.Update("old", Snippet{Key: "new", Value: "v"})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if f.Has("old") {
		t.Error("old key should be gone")
	}
	if !f.Has("new") {
		t.Error("new key should exist")
	}
}

func TestUpdateRejectsCollision(t *testing.T) {
	f := &File{}
	_ = f.Add(Snippet{Key: "a", Value: "x"})
	_ = f.Add(Snippet{Key: "b", Value: "y"})
	err := f.Update("a", Snippet{Key: "b", Value: "x"})
	if err != ErrExists {
		t.Errorf("err = %v, want ErrExists", err)
	}
}

func TestUpdatePreservesMetadata(t *testing.T) {
	f := &File{}
	_ = f.Add(Snippet{Key: "k", Value: "v", UseCount: 3, CreatedAt: time.Date(2026, 5, 5, 0, 0, 0, 0, time.UTC)})
	_ = f.Update("k", Snippet{Key: "k", Value: "v2"})
	got, _ := f.Get("k")
	if got.UseCount != 3 {
		t.Errorf("UseCount = %d, want 3", got.UseCount)
	}
	if got.CreatedAt.IsZero() {
		t.Error("CreatedAt should be preserved")
	}
}

func TestTouchUpdatesLastUsedAndCount(t *testing.T) {
	f := &File{}
	_ = f.Add(Snippet{Key: "k", Value: "v"})
	when := time.Date(2026, 5, 5, 12, 0, 0, 0, time.UTC)
	if err := f.Touch("k", when); err != nil {
		t.Fatalf("Touch: %v", err)
	}
	got, _ := f.Get("k")
	if got.UseCount != 1 {
		t.Errorf("UseCount = %d, want 1", got.UseCount)
	}
	if got.LastUsedAt == nil || !got.LastUsedAt.Equal(when) {
		t.Errorf("LastUsedAt = %v, want %v", got.LastUsedAt, when)
	}
	_ = f.Touch("k", when.Add(time.Hour))
	got, _ = f.Get("k")
	if got.UseCount != 2 {
		t.Errorf("UseCount after second touch = %d, want 2", got.UseCount)
	}
}

func TestSearchHitsMultipleFields(t *testing.T) {
	f := &File{}
	_ = f.Add(Snippet{Key: "claude", Value: "claude --x", Description: "AI agent"})
	_ = f.Add(Snippet{Key: "codex", Value: "codex --resume", Tags: []string{"ai", "openai"}})
	_ = f.Add(Snippet{Key: "rebase", Value: "git fetch && git rebase"})

	hits := f.Search("ai")
	if len(hits) != 2 {
		t.Errorf("ai → %d hits, want 2 (claude desc, codex tag)", len(hits))
	}
	hits = f.Search("rebase")
	if len(hits) != 1 || hits[0].Key != "rebase" {
		t.Errorf("rebase search bad: %+v", hits)
	}
	hits = f.Search("CODEX") // case-insensitive
	if len(hits) != 1 || hits[0].Key != "codex" {
		t.Errorf("CODEX search bad: %+v", hits)
	}
}

func TestLoadBrokenJSONErrors(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "pe.json")
	if err := os.WriteFile(path, []byte("not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := Load(path)
	if err == nil {
		t.Error("expected error for malformed file, got nil")
	}
}

func TestDefaultPathRespectsPEDIR(t *testing.T) {
	t.Setenv("PE_DIR", "/tmp/somewhere")
	got, err := DefaultPath()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join("/tmp/somewhere", "pe.json")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestDefaultPathFallsBackToHome(t *testing.T) {
	t.Setenv("PE_DIR", "")
	got, err := DefaultPath()
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(got) != "pe.json" {
		t.Errorf("base = %q, want pe.json", filepath.Base(got))
	}
	if filepath.Base(filepath.Dir(got)) != ".pe" {
		t.Errorf("parent = %q, want .pe", filepath.Base(filepath.Dir(got)))
	}
}
