package cli

import (
	"reflect"
	"testing"

	"github.com/Shin-R2un/pe/internal/store"
)

func mkFile(keys ...string) *store.File {
	f := &store.File{}
	for _, k := range keys {
		_ = f.Add(store.Snippet{Key: k, Value: "v"})
	}
	return f
}

func TestSuggestPrefix(t *testing.T) {
	f := mkFile("claude", "clawd", "codex", "rebase")
	got := suggest(f, "cla", 3)
	want := []string{"claude", "clawd"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestSuggestSubstring(t *testing.T) {
	f := mkFile("install-foo", "uninstall-bar", "noop")
	got := suggest(f, "stall", 3)
	want := []string{"install-foo", "uninstall-bar"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestSuggestLevenshtein(t *testing.T) {
	f := mkFile("claude", "clawd", "rebase")
	// "celude" → distance 2 from "claude" (a→e, swap order — actually
	// e+l+u+d+e vs claude is dist 2). prefix/substring don't match.
	got := suggest(f, "celude", 3)
	if len(got) == 0 || got[0] != "claude" {
		t.Errorf("got %v, want first=claude", got)
	}
}

func TestSuggestEmptyOnNoMatch(t *testing.T) {
	f := mkFile("claude")
	got := suggest(f, "completelyunrelated", 3)
	if len(got) != 0 {
		t.Errorf("got %v, want []", got)
	}
}

func TestSuggestEmptyQuery(t *testing.T) {
	f := mkFile("claude")
	if got := suggest(f, "", 3); got != nil {
		t.Errorf("got %v, want nil for empty query", got)
	}
}

func TestSuggestRespectsLimit(t *testing.T) {
	f := mkFile("a1", "a2", "a3", "a4", "a5")
	got := suggest(f, "a", 3)
	if len(got) != 3 {
		t.Errorf("len = %d, want 3 (limit)", len(got))
	}
}

func TestSuggestCaseInsensitive(t *testing.T) {
	f := mkFile("Claude", "ClawD")
	got := suggest(f, "cla", 3)
	if len(got) != 2 {
		t.Errorf("got %v, want 2 hits", got)
	}
}

func TestLevenshteinKnownDistances(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"", "", 0},
		{"abc", "abc", 0},
		{"abc", "abd", 1},
		{"kitten", "sitting", 3},
		{"", "abc", 3},
		{"abc", "", 3},
	}
	for _, c := range cases {
		if got := levenshtein(c.a, c.b); got != c.want {
			t.Errorf("levenshtein(%q, %q) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}
