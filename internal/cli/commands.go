package cli

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/Shin-R2un/pe/internal/store"
)

// displayWidth returns the character count for column-padding purposes.
// Uses rune count (not byte length) so non-ASCII keys don't blow up the
// padding. Note: CJK fullwidth chars still occupy 2 cells but are
// counted as 1 here — good enough for now; introduce x/text/width or
// go-runewidth if exact column alignment for CJK is needed later.
func displayWidth(s string) int {
	return utf8.RuneCountInString(s)
}

// padRight left-aligns s in a column `width` runes wide. fmt's `%-*s`
// is byte-oriented and produces wrong padding for multi-byte input.
func padRight(s string, width int) string {
	pad := width - displayWidth(s)
	if pad <= 0 {
		return s
	}
	return s + strings.Repeat(" ", pad)
}

// preview returns a short single-line summary of a snippet for `pe l`.
// Prefers description; falls back to a truncated first line of value.
func preview(s store.Snippet, max int) string {
	if s.Description != "" {
		return truncate(s.Description, max)
	}
	v := s.Value
	if i := strings.IndexAny(v, "\r\n"); i >= 0 {
		v = v[:i] + " …"
	}
	return truncate(v, max)
}

func truncate(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max-1]) + "…"
}

// notFound prints a "not found" error with up to 3 did-you-mean
// suggestions when the snippet file has any near-matches.
func (a *App) notFound(f *store.File, key string) int {
	fmt.Fprintf(a.err(), "not found: %s\n", key)
	if hits := suggest(f, key, 3); len(hits) > 0 {
		fmt.Fprintf(a.err(), "did you mean: %s?\n", strings.Join(hits, ", "))
	} else {
		fmt.Fprintf(a.err(), "hint: try `pe s %s`\n", key)
	}
	return 1
}

// validateKey rejects empty / whitespace-only / reserved-word / control-char keys.
func validateKey(k string) error {
	if strings.TrimSpace(k) == "" {
		return fmt.Errorf("key must not be empty")
	}
	if isReserved(k) {
		return fmt.Errorf("cannot use reserved keyword as key: %s", k)
	}
	for _, r := range k {
		if r == '\n' || r == '\r' || r == '\t' || r == ' ' {
			return fmt.Errorf("key must not contain whitespace: %q", k)
		}
	}
	return nil
}

// ----- add ---------------------------------------------------------------

func (a *App) cmdAdd(args []string) int {
	if len(args) < 2 {
		return a.usage("usage: pe a <key> <text>")
	}
	key := args[0]
	value := strings.Join(args[1:], " ")
	if err := validateKey(key); err != nil {
		return a.errorf("%v", err)
	}
	f, p, err := a.load()
	if err != nil {
		return a.errorf("load %s: %v", p, err)
	}
	now := a.now().UTC()
	err = f.Add(store.Snippet{
		Key:       key,
		Value:     value,
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		switch err {
		case store.ErrExists:
			fmt.Fprintf(a.err(), "already exists: %s\nhint: use `pe e %s` to edit\n", key, key)
			return 1
		default:
			return a.errorf("%v", err)
		}
	}
	if err := store.Save(p, f); err != nil {
		return a.errorf("save %s: %v", p, err)
	}
	fmt.Fprintf(a.out(), "added: %s\n", key)
	return 0
}

// ----- copy --------------------------------------------------------------

func (a *App) cmdCopy(key string) int {
	f, p, err := a.load()
	if err != nil {
		return a.errorf("load %s: %v", p, err)
	}
	s, err := f.Get(key)
	if err != nil {
		return a.notFound(f, key)
	}
	if err := a.copyFn()(s.Value); err != nil {
		fmt.Fprintf(a.err(), "copy: %v\n", err)
		if strings.Contains(err.Error(), "no clipboard backend") {
			fmt.Fprintln(a.err(), "hint: install xclip / wl-copy / xsel,")
			fmt.Fprintln(a.err(), "      or in SSH/tmux+WezTerm: see README \"OSC 52\" section")
		}
		return 1
	}
	if err := f.Touch(key, a.now()); err == nil {
		_ = store.Save(p, f) // best-effort; touch failures shouldn't block UX
	}
	fmt.Fprintf(a.out(), "copied: %s\n", key)
	return 0
}

// ----- list --------------------------------------------------------------

func (a *App) cmdList(args []string) int {
	_ = args
	f, p, err := a.load()
	if err != nil {
		return a.errorf("load %s: %v", p, err)
	}
	if len(f.Snippets) == 0 {
		fmt.Fprintln(a.out(), "no snippets yet — try `pe a <key> <text>`")
		return 0
	}
	keys := f.SortedKeys()
	width := 0
	for _, k := range keys {
		if w := displayWidth(k); w > width {
			width = w
		}
	}
	if width > 24 {
		width = 24
	}
	for _, k := range keys {
		s, _ := f.Get(k)
		fmt.Fprintf(a.out(), "%s  %s\n", padRight(k, width), preview(*s, 60))
	}
	return 0
}

// ----- search ------------------------------------------------------------

func (a *App) cmdSearch(args []string) int {
	if len(args) < 1 {
		return a.usage("usage: pe s <query>")
	}
	query := strings.Join(args, " ")
	f, p, err := a.load()
	if err != nil {
		return a.errorf("load %s: %v", p, err)
	}
	hits := f.Search(query)
	if len(hits) == 0 {
		fmt.Fprintf(a.out(), "no matches for: %s\n", query)
		return 0
	}
	width := 0
	for _, s := range hits {
		if w := displayWidth(s.Key); w > width {
			width = w
		}
	}
	if width > 24 {
		width = 24
	}
	for _, s := range hits {
		fmt.Fprintf(a.out(), "%s  %s\n", padRight(s.Key, width), preview(s, 60))
	}
	return 0
}

// ----- show --------------------------------------------------------------

func (a *App) cmdShow(args []string) int {
	if len(args) < 1 {
		return a.usage("usage: pe ? <key>")
	}
	key := args[0]
	f, p, err := a.load()
	if err != nil {
		return a.errorf("load %s: %v", p, err)
	}
	s, err := f.Get(key)
	if err != nil {
		return a.notFound(f, key)
	}
	fmt.Fprintf(a.out(), "key:         %s\n", s.Key)
	if s.Description != "" {
		fmt.Fprintf(a.out(), "description: %s\n", s.Description)
	}
	if len(s.Tags) > 0 {
		fmt.Fprintf(a.out(), "tags:        %s\n", strings.Join(s.Tags, ", "))
	}
	if s.UseCount > 0 {
		fmt.Fprintf(a.out(), "useCount:    %d\n", s.UseCount)
	}
	fmt.Fprintln(a.out(), "value:")
	fmt.Fprintln(a.out(), s.Value)
	return 0
}

// ----- delete ------------------------------------------------------------

func (a *App) cmdDelete(args []string) int {
	if len(args) < 1 {
		return a.usage("usage: pe d <key>")
	}
	key := args[0]
	f, p, err := a.load()
	if err != nil {
		return a.errorf("load %s: %v", p, err)
	}
	if err := f.Delete(key); err != nil {
		return a.notFound(f, key)
	}
	if err := store.Save(p, f); err != nil {
		return a.errorf("save %s: %v", p, err)
	}
	fmt.Fprintf(a.out(), "deleted: %s\n", key)
	return 0
}
