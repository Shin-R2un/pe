package cli

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Shin-R2un/pe/internal/store"
)

// helper: build an App that writes to a temp store and a fake clipboard.
func newTestApp(t *testing.T) (*App, *bytes.Buffer, *bytes.Buffer, *string) {
	t.Helper()
	dir := t.TempDir()
	out := &bytes.Buffer{}
	errBuf := &bytes.Buffer{}
	clipboard := ""
	app := &App{
		Path: filepath.Join(dir, "pe.json"),
		Out:  out,
		Err:  errBuf,
		Copy: func(s string) error { clipboard = s; return nil },
		Now:  func() time.Time { return time.Date(2026, 5, 5, 0, 0, 0, 0, time.UTC) },
	}
	return app, out, errBuf, &clipboard
}

func TestAddThenCopy(t *testing.T) {
	app, out, errBuf, clip := newTestApp(t)
	if code := app.Run([]string{"a", "claude", "claude --x"}); code != 0 {
		t.Fatalf("add exit=%d err=%q", code, errBuf)
	}
	if !strings.Contains(out.String(), "added: claude") {
		t.Errorf("out = %q", out)
	}

	out.Reset()
	errBuf.Reset()
	if code := app.Run([]string{"claude"}); code != 0 {
		t.Fatalf("copy exit=%d err=%q", code, errBuf)
	}
	if *clip != "claude --x" {
		t.Errorf("clipboard = %q, want %q", *clip, "claude --x")
	}
	if !strings.Contains(out.String(), "copied: claude") {
		t.Errorf("out = %q", out)
	}
}

func TestAddJoinsTrailingArgs(t *testing.T) {
	app, _, _, clip := newTestApp(t)
	app.Run([]string{"a", "k", "hello", "world"})
	app.Run([]string{"k"})
	if *clip != "hello world" {
		t.Errorf("clipboard = %q, want %q", *clip, "hello world")
	}
}

func TestCopyMissingKey(t *testing.T) {
	app, _, errBuf, _ := newTestApp(t)
	code := app.Run([]string{"nope"})
	if code != 1 {
		t.Errorf("exit = %d, want 1", code)
	}
	s := errBuf.String()
	if !strings.Contains(s, "not found: nope") {
		t.Errorf("err = %q (no 'not found:')", s)
	}
	if !strings.Contains(s, "hint: try `pe s nope`") {
		t.Errorf("err = %q (no hint)", s)
	}
}

func TestCopyMissingKeyShowsDidYouMean(t *testing.T) {
	app, _, errBuf, _ := newTestApp(t)
	app.Run([]string{"a", "claude", "v"})
	app.Run([]string{"a", "clawd", "v"})

	errBuf.Reset()
	code := app.Run([]string{"clude"})
	if code != 1 {
		t.Errorf("exit = %d, want 1", code)
	}
	s := errBuf.String()
	if !strings.Contains(s, "not found: clude") {
		t.Errorf("err = %q", s)
	}
	if !strings.Contains(s, "did you mean:") {
		t.Errorf("missing did-you-mean: %q", s)
	}
	if !strings.Contains(s, "claude") {
		t.Errorf("expected claude in suggestions: %q", s)
	}
}

func TestEditMissingShowsDidYouMean(t *testing.T) {
	app, _, errBuf, _ := newTestApp(t)
	app.Run([]string{"a", "claude", "v"})

	errBuf.Reset()
	code := app.Run([]string{"e", "clude"})
	if code != 1 {
		t.Errorf("exit = %d, want 1", code)
	}
	if !strings.Contains(errBuf.String(), "did you mean") {
		t.Errorf("err = %q", errBuf)
	}
}

func TestInternalCompleteListsKeysWithPrefix(t *testing.T) {
	app, out, _, _ := newTestApp(t)
	app.Run([]string{"a", "claude", "v"})
	app.Run([]string{"a", "clawd", "v"})
	app.Run([]string{"a", "rebase", "v"})

	out.Reset()
	if code := app.Run([]string{"__complete", "cla"}); code != 0 {
		t.Fatalf("exit = %d", code)
	}
	body := out.String()
	if !strings.Contains(body, "claude") || !strings.Contains(body, "clawd") {
		t.Errorf("missing claude/clawd: %q", body)
	}
	if strings.Contains(body, "rebase") {
		t.Errorf("rebase should not match prefix 'cla': %q", body)
	}
}

func TestInternalCompleteEmptyPrefixListsAll(t *testing.T) {
	app, out, _, _ := newTestApp(t)
	app.Run([]string{"a", "zeta", "v"})
	app.Run([]string{"a", "alpha", "v"})

	out.Reset()
	app.Run([]string{"__complete"})
	body := out.String()
	if body != "alpha\nzeta\n" {
		t.Errorf("got %q, want %q", body, "alpha\nzeta\n")
	}
}

func TestCompletionEmitsScripts(t *testing.T) {
	for _, shell := range []string{"bash", "zsh", "fish", "powershell", "pwsh"} {
		app, out, _, _ := newTestApp(t)
		out.Reset()
		if code := app.Run([]string{"completion", shell}); code != 0 {
			t.Errorf("%s: exit = %d", shell, code)
		}
		if !strings.Contains(out.String(), "pe __complete") {
			t.Errorf("%s: script doesn't reference `pe __complete`: %q", shell, out)
		}
	}
}

func TestCompletionPowerShellHasRegisterArgumentCompleter(t *testing.T) {
	app, out, _, _ := newTestApp(t)
	if code := app.Run([]string{"completion", "powershell"}); code != 0 {
		t.Fatalf("exit = %d", code)
	}
	body := out.String()
	if !strings.Contains(body, "Register-ArgumentCompleter") {
		t.Errorf("missing Register-ArgumentCompleter: %q", body[:200])
	}
	if !strings.Contains(body, "CommandName pe") {
		t.Errorf("missing -CommandName pe: %q", body[:200])
	}
}

func TestCompletionUnknownShellErrors(t *testing.T) {
	app, _, errBuf, _ := newTestApp(t)
	if code := app.Run([]string{"completion", "tcsh"}); code != 2 {
		t.Errorf("exit = %d, want 2", code)
	}
	if !strings.Contains(errBuf.String(), "unsupported shell") {
		t.Errorf("err = %q", errBuf)
	}
}

func TestAddRejectsCompletionAsKey(t *testing.T) {
	app, _, errBuf, _ := newTestApp(t)
	if code := app.Run([]string{"a", "completion", "x"}); code != 1 {
		t.Errorf("exit = %d, want 1", code)
	}
	if !strings.Contains(errBuf.String(), "reserved keyword") {
		t.Errorf("err = %q", errBuf)
	}
}

func TestAddDuplicateGivesHint(t *testing.T) {
	app, _, errBuf, _ := newTestApp(t)
	app.Run([]string{"a", "k", "v"})
	errBuf.Reset()
	code := app.Run([]string{"a", "k", "v2"})
	if code != 1 {
		t.Errorf("exit = %d, want 1", code)
	}
	if !strings.Contains(errBuf.String(), "already exists: k") {
		t.Errorf("err = %q", errBuf)
	}
	if !strings.Contains(errBuf.String(), "use `pe e k`") {
		t.Errorf("err = %q (no edit hint)", errBuf)
	}
}

func TestAddRejectsReservedKey(t *testing.T) {
	app, _, errBuf, _ := newTestApp(t)
	for _, k := range []string{"help", "version", "a", "?"} {
		errBuf.Reset()
		code := app.Run([]string{"a", k, "x"})
		if code != 1 {
			t.Errorf("reserved key %q: exit = %d, want 1", k, code)
		}
		if !strings.Contains(errBuf.String(), "reserved keyword") {
			t.Errorf("reserved key %q: err = %q", k, errBuf)
		}
	}
}

func TestAddRejectsKeyWithWhitespace(t *testing.T) {
	app, _, errBuf, _ := newTestApp(t)
	code := app.Run([]string{"a", "with space", "x"})
	if code != 1 {
		t.Errorf("exit = %d, want 1", code)
	}
	if !strings.Contains(errBuf.String(), "whitespace") {
		t.Errorf("err = %q", errBuf)
	}
}

func TestList(t *testing.T) {
	app, out, _, _ := newTestApp(t)
	app.Run([]string{"a", "claude", "claude --x"})
	app.Run([]string{"a", "codex", "codex --resume"})

	out.Reset()
	if code := app.Run([]string{"l"}); code != 0 {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(out.String(), "claude") {
		t.Errorf("list missing claude: %q", out)
	}
	if !strings.Contains(out.String(), "codex") {
		t.Errorf("list missing codex: %q", out)
	}
}

func TestListEmpty(t *testing.T) {
	app, out, _, _ := newTestApp(t)
	if code := app.Run([]string{"l"}); code != 0 {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(out.String(), "no snippets yet") {
		t.Errorf("out = %q", out)
	}
}

func TestSearch(t *testing.T) {
	app, out, _, _ := newTestApp(t)
	app.Run([]string{"a", "claude", "claude --x"})
	app.Run([]string{"a", "clawd", "cd /openclaw"})
	app.Run([]string{"a", "rebase", "git rebase"})

	out.Reset()
	if code := app.Run([]string{"s", "cl"}); code != 0 {
		t.Fatalf("exit = %d", code)
	}
	body := out.String()
	if !strings.Contains(body, "claude") || !strings.Contains(body, "clawd") {
		t.Errorf("missing claude/clawd: %q", body)
	}
	if strings.Contains(body, "rebase") {
		t.Errorf("rebase shouldn't match: %q", body)
	}
}

func TestSearchNoQuery(t *testing.T) {
	app, _, errBuf, _ := newTestApp(t)
	code := app.Run([]string{"s"})
	if code != 2 {
		t.Errorf("exit = %d, want 2 (usage)", code)
	}
	if !strings.Contains(errBuf.String(), "usage") {
		t.Errorf("err = %q", errBuf)
	}
}

func TestShow(t *testing.T) {
	app, out, _, _ := newTestApp(t)
	app.Run([]string{"a", "k", "the-value"})

	out.Reset()
	if code := app.Run([]string{"?", "k"}); code != 0 {
		t.Fatalf("exit = %d", code)
	}
	body := out.String()
	if !strings.Contains(body, "key:") || !strings.Contains(body, "the-value") {
		t.Errorf("show output bad: %q", body)
	}
}

func TestDelete(t *testing.T) {
	app, out, _, _ := newTestApp(t)
	app.Run([]string{"a", "k", "v"})
	out.Reset()
	if code := app.Run([]string{"d", "k"}); code != 0 {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(out.String(), "deleted: k") {
		t.Errorf("out = %q", out)
	}
	// it should now be gone
	if code := app.Run([]string{"k"}); code != 1 {
		t.Errorf("post-delete copy: exit = %d, want 1", code)
	}
}

func TestDeleteMissing(t *testing.T) {
	app, _, errBuf, _ := newTestApp(t)
	code := app.Run([]string{"d", "ghost"})
	if code != 1 {
		t.Errorf("exit = %d, want 1", code)
	}
	if !strings.Contains(errBuf.String(), "not found: ghost") {
		t.Errorf("err = %q", errBuf)
	}
}

func TestVersionAndHelp(t *testing.T) {
	app, out, _, _ := newTestApp(t)
	if code := app.Run([]string{"version"}); code != 0 {
		t.Errorf("version exit = %d", code)
	}
	if out.String() == "" {
		t.Error("version printed nothing")
	}

	out.Reset()
	if code := app.Run([]string{"help"}); code != 0 {
		t.Errorf("help exit = %d", code)
	}
	if !strings.Contains(out.String(), "Usage:") {
		t.Errorf("help missing 'Usage:': %q", out)
	}
}

func TestUnknownSubcommandWithExtraArgsErrors(t *testing.T) {
	app, _, errBuf, _ := newTestApp(t)
	// `pe foo bar` — foo isn't a known subcommand, and there's a bar
	// after it; dispatcher should treat this as a usage error rather
	// than silently trying to copy "foo".
	code := app.Run([]string{"foo", "bar"})
	if code != 2 {
		t.Errorf("exit = %d, want 2", code)
	}
	if !strings.Contains(errBuf.String(), "unknown subcommand") {
		t.Errorf("err = %q", errBuf)
	}
}

func TestCopyTouchesUseCount(t *testing.T) {
	app, _, _, _ := newTestApp(t)
	app.Run([]string{"a", "k", "v"})
	app.Run([]string{"k"})
	app.Run([]string{"k"})

	f, err := store.Load(app.Path)
	if err != nil {
		t.Fatal(err)
	}
	got, err := f.Get("k")
	if err != nil {
		t.Fatal(err)
	}
	if got.UseCount != 2 {
		t.Errorf("UseCount = %d, want 2", got.UseCount)
	}
	if got.LastUsedAt == nil {
		t.Error("LastUsedAt should be set after copy")
	}
}
