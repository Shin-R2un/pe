// Package clip writes text to the system clipboard.
//
// Two strategies, tried in this order unless overridden by PE_CLIP:
//
//  1. A native helper, if one is on PATH:
//     macOS         pbcopy
//     Linux/Wayland wl-copy
//     Linux/X11     xclip / xsel
//     WSL           clip.exe (after the above fail)
//     Windows       clip
//
//  2. OSC 52 escape sequence written to /dev/tty. Carries the clipboard
//     payload through the terminal emulator (and through tmux/SSH on
//     the way), so it works on a headless remote box as long as the
//     local terminal — WezTerm, iTerm2, kitty, Alacritty, modern xterm,
//     Windows Terminal, etc. — opts into OSC 52.
//
// PE_CLIP overrides the strategy:
//
//	PE_CLIP=native  use a native helper, fail if absent
//	PE_CLIP=osc52   skip native, always emit OSC 52
//	PE_CLIP=none    no-op (useful for testing pe itself)
//	(unset)         auto: native first, OSC 52 fallback
package clip

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// ErrNoBackend is returned when no supported clipboard backend can be used.
var ErrNoBackend = errors.New("no clipboard backend found")

// pickCommand is overridable for tests.
var pickCommand = defaultPickCommand

// openTTY is overridable for tests; returns a writer to the controlling tty.
var openTTY = func() (io.WriteCloser, error) {
	return os.OpenFile("/dev/tty", os.O_WRONLY, 0)
}

// Copy writes s to the system clipboard, honoring PE_CLIP.
func Copy(s string) error {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("PE_CLIP"))) {
	case "native":
		return nativeCopy(s)
	case "osc52":
		return osc52Copy(s)
	case "none", "off", "disabled":
		return nil
	}
	// auto
	err := nativeCopy(s)
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrNoBackend) {
		if osc52Err := osc52Copy(s); osc52Err == nil {
			return nil
		}
		return ErrNoBackend
	}
	return err
}

func nativeCopy(s string) error {
	cmd, err := pickCommand()
	if err != nil {
		return err
	}
	cmd.Stdin = strings.NewReader(s)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("clipboard copy failed: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func osc52Copy(s string) error {
	w, err := openTTY()
	if err != nil {
		// No controlling TTY → can't deliver OSC 52.
		return ErrNoBackend
	}
	defer w.Close()
	if _, err := io.WriteString(w, osc52Sequence(s)); err != nil {
		return fmt.Errorf("osc52 write: %w", err)
	}
	return nil
}

// osc52Sequence builds the escape string that asks the terminal to put
// `s` on the clipboard. When inside tmux, wraps it in DCS passthrough
// so tmux forwards it to the outer terminal verbatim — requires
// `set -g allow-passthrough on` on tmux 3.3+.
func osc52Sequence(s string) string {
	enc := base64.StdEncoding.EncodeToString([]byte(s))
	if os.Getenv("TMUX") != "" {
		// Inside tmux: DCS passthrough doubles every ESC in the inner
		// payload. Use BEL (\x07) as the OSC terminator inside —
		// safer than ST when ESCs are being doubled.
		inner := "\x1b]52;c;" + enc + "\x07"
		inner = strings.ReplaceAll(inner, "\x1b", "\x1b\x1b")
		return "\x1bPtmux;" + inner + "\x1b\\"
	}
	return "\x1b]52;c;" + enc + "\x1b\\"
}

func defaultPickCommand() (*exec.Cmd, error) {
	switch runtime.GOOS {
	case "darwin":
		if p, err := exec.LookPath("pbcopy"); err == nil {
			return exec.Command(p), nil
		}
	case "windows":
		if p, err := exec.LookPath("clip"); err == nil {
			return exec.Command(p), nil
		}
	default: // linux & others
		if p, err := exec.LookPath("wl-copy"); err == nil {
			return exec.Command(p), nil
		}
		if p, err := exec.LookPath("xclip"); err == nil {
			return exec.Command(p, "-selection", "clipboard"), nil
		}
		if p, err := exec.LookPath("xsel"); err == nil {
			return exec.Command(p, "--clipboard", "--input"), nil
		}
		if isWSL() {
			if p, err := exec.LookPath("clip.exe"); err == nil {
				return exec.Command(p), nil
			}
		}
	}
	return nil, ErrNoBackend
}

// procVersionPath is the path checked for the WSL signature; a var so
// tests can point it at a fixture.
var procVersionPath = "/proc/version"

func isWSL() bool {
	if os.Getenv("WSL_DISTRO_NAME") != "" {
		return true
	}
	b, err := os.ReadFile(procVersionPath)
	if err != nil {
		return false
	}
	return bytes.Contains(bytes.ToLower(b), []byte("microsoft"))
}
