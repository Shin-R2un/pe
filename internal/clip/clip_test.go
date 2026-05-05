package clip

import (
	"bytes"
	"encoding/base64"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestCopyShellsOutToBackend(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses POSIX `tee` as a stand-in clipboard backend")
	}
	t.Setenv("PE_CLIP", "")
	dir := t.TempDir()
	target := filepath.Join(dir, "captured.txt")

	orig := pickCommand
	t.Cleanup(func() { pickCommand = orig })

	pickCommand = func() (*exec.Cmd, error) {
		return exec.Command("tee", target), nil
	}

	if err := Copy("hello pe"); err != nil {
		t.Fatalf("Copy: %v", err)
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "hello pe" {
		t.Errorf("captured = %q, want %q", got, "hello pe")
	}
}

// In native-only mode, missing helpers must propagate ErrNoBackend
// (no OSC 52 fallback).
func TestCopyNativeModePropagatesNoBackend(t *testing.T) {
	t.Setenv("PE_CLIP", "native")
	orig := pickCommand
	t.Cleanup(func() { pickCommand = orig })
	pickCommand = func() (*exec.Cmd, error) { return nil, ErrNoBackend }

	err := Copy("x")
	if !errors.Is(err, ErrNoBackend) {
		t.Errorf("err = %v, want ErrNoBackend", err)
	}
}

// In auto mode, when no native helper is available, OSC 52 should be
// emitted to /dev/tty (here mocked).
func TestCopyAutoFallsBackToOSC52(t *testing.T) {
	t.Setenv("PE_CLIP", "")
	t.Setenv("TMUX", "")
	orig := pickCommand
	origTTY := openTTY
	t.Cleanup(func() {
		pickCommand = orig
		openTTY = origTTY
	})
	pickCommand = func() (*exec.Cmd, error) { return nil, ErrNoBackend }

	captured := &bytes.Buffer{}
	openTTY = func() (io.WriteCloser, error) {
		return nopWriteCloser{captured}, nil
	}

	if err := Copy("hello pe"); err != nil {
		t.Fatalf("Copy: %v", err)
	}
	want := "\x1b]52;c;" + base64.StdEncoding.EncodeToString([]byte("hello pe")) + "\x1b\\"
	if captured.String() != want {
		t.Errorf("OSC 52 sequence = %q, want %q", captured, want)
	}
}

// PE_CLIP=osc52 forces OSC 52 even when a native helper would work.
func TestCopyForcedOSC52(t *testing.T) {
	t.Setenv("PE_CLIP", "osc52")
	t.Setenv("TMUX", "")
	origTTY := openTTY
	t.Cleanup(func() { openTTY = origTTY })

	captured := &bytes.Buffer{}
	openTTY = func() (io.WriteCloser, error) {
		return nopWriteCloser{captured}, nil
	}
	if err := Copy("forced"); err != nil {
		t.Fatalf("Copy: %v", err)
	}
	if !strings.Contains(captured.String(), "\x1b]52;c;") {
		t.Errorf("expected OSC 52 prefix, got %q", captured)
	}
}

// PE_CLIP=none silences copying for testing.
func TestCopyDisabled(t *testing.T) {
	t.Setenv("PE_CLIP", "none")
	origTTY := openTTY
	t.Cleanup(func() { openTTY = origTTY })
	openTTY = func() (io.WriteCloser, error) {
		t.Fatal("openTTY should not be called when disabled")
		return nil, nil
	}
	if err := Copy("x"); err != nil {
		t.Errorf("err = %v, want nil", err)
	}
}

// When inside tmux, OSC 52 is wrapped in DCS passthrough.
func TestOSC52SequenceTmuxWrapped(t *testing.T) {
	t.Setenv("TMUX", "/tmp/tmux-1000/default,123,0")
	got := osc52Sequence("hi")
	wantPrefix := "\x1bPtmux;"
	wantSuffix := "\x1b\\"
	if !strings.HasPrefix(got, wantPrefix) {
		t.Errorf("missing tmux DCS prefix: %q", got)
	}
	if !strings.HasSuffix(got, wantSuffix) {
		t.Errorf("missing ST suffix: %q", got)
	}
	// inner ESC must be doubled
	if !strings.Contains(got, "\x1b\x1b]52;c;") {
		t.Errorf("inner ESC not doubled: %q", got)
	}
	// inner OSC terminates with BEL
	if !strings.Contains(got, "\x07") {
		t.Errorf("inner OSC should end with BEL: %q", got)
	}
}

func TestOSC52SequencePlain(t *testing.T) {
	t.Setenv("TMUX", "")
	got := osc52Sequence("hi")
	want := "\x1b]52;c;" + base64.StdEncoding.EncodeToString([]byte("hi")) + "\x1b\\"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// When no /dev/tty is available, OSC 52 path returns ErrNoBackend.
func TestOSC52NoTTY(t *testing.T) {
	t.Setenv("PE_CLIP", "")
	orig := pickCommand
	origTTY := openTTY
	t.Cleanup(func() {
		pickCommand = orig
		openTTY = origTTY
	})
	pickCommand = func() (*exec.Cmd, error) { return nil, ErrNoBackend }
	openTTY = func() (io.WriteCloser, error) {
		return nil, errors.New("no tty in CI")
	}

	err := Copy("x")
	if !errors.Is(err, ErrNoBackend) {
		t.Errorf("err = %v, want ErrNoBackend", err)
	}
}

func TestUTF16LEWithBOM(t *testing.T) {
	got := utf16LEWithBOM("こんにちは")
	// 5 chars × 2 bytes + 2-byte BOM
	want := []byte{
		0xFF, 0xFE, // BOM
		0x53, 0x30, // こ U+3053
		0x93, 0x30, // ん U+3093
		0x6B, 0x30, // に U+306B
		0x61, 0x30, // ち U+3061
		0x6F, 0x30, // は U+306F
	}
	if !bytes.Equal(got, want) {
		t.Errorf("got % X\nwant % X", got, want)
	}
}

func TestUTF16LEWithBOMASCIIIsLossless(t *testing.T) {
	got := utf16LEWithBOM("hi")
	want := []byte{0xFF, 0xFE, 'h', 0x00, 'i', 0x00}
	if !bytes.Equal(got, want) {
		t.Errorf("got % X, want % X", got, want)
	}
}

func TestStdinForClipExeUsesUTF16(t *testing.T) {
	cmd := exec.Command(`C:\Windows\System32\clip.exe`)
	r := stdinFor(cmd, "あ")
	buf, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	if len(buf) < 4 || buf[0] != 0xFF || buf[1] != 0xFE {
		t.Errorf("expected UTF-16LE BOM for clip.exe, got % X", buf)
	}
}

func TestStdinForOthersStaysUTF8(t *testing.T) {
	for _, helper := range []string{"/usr/bin/xclip", "/usr/bin/wl-copy", "/usr/bin/pbcopy", "/usr/bin/xsel"} {
		cmd := exec.Command(helper)
		r := stdinFor(cmd, "あ")
		buf, _ := io.ReadAll(r)
		if string(buf) != "あ" {
			t.Errorf("%s: stdin = % X, want UTF-8 %q", helper, buf, "あ")
		}
	}
}

func TestIsClipExeBasenameMatching(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		{`C:\Windows\System32\clip.exe`, true},
		{`/mnt/c/Windows/System32/clip.exe`, true}, // WSL fallback path
		{`CLIP.EXE`, true},                         // case-insensitive
		{`/usr/bin/xclip`, false},
		{`/usr/bin/wl-copy`, false},
		{`/usr/local/bin/clip`, false}, // not "clip.exe"
		{``, false},
	}
	for _, c := range cases {
		got := isClipExe(exec.Command(c.path))
		if got != c.want {
			t.Errorf("isClipExe(%q) = %v, want %v", c.path, got, c.want)
		}
	}
}

func TestIsWSLDetectsEnvVar(t *testing.T) {
	t.Setenv("WSL_DISTRO_NAME", "Ubuntu")
	if !isWSL() {
		t.Error("WSL_DISTRO_NAME=Ubuntu should be detected as WSL")
	}
}

func TestIsWSLDetectsProcVersion(t *testing.T) {
	t.Setenv("WSL_DISTRO_NAME", "")
	dir := t.TempDir()
	fixture := filepath.Join(dir, "version")
	if err := os.WriteFile(fixture, []byte("Linux version 5.15.0 Microsoft WSL2"), 0o600); err != nil {
		t.Fatal(err)
	}
	orig := procVersionPath
	procVersionPath = fixture
	t.Cleanup(func() { procVersionPath = orig })

	if !isWSL() {
		t.Error("microsoft signature in /proc/version should be detected as WSL")
	}
}

func TestIsWSLNegative(t *testing.T) {
	t.Setenv("WSL_DISTRO_NAME", "")
	dir := t.TempDir()
	fixture := filepath.Join(dir, "version")
	if err := os.WriteFile(fixture, []byte("Linux version 6.12.10-76061203-generic"), 0o600); err != nil {
		t.Fatal(err)
	}
	orig := procVersionPath
	procVersionPath = fixture
	t.Cleanup(func() { procVersionPath = orig })

	if isWSL() {
		t.Error("plain Linux should not be detected as WSL")
	}
}

type nopWriteCloser struct{ io.Writer }

func (nopWriteCloser) Close() error { return nil }
