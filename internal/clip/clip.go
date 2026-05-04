// Package clip writes text to the system clipboard by shelling out to
// platform-native helpers. No CGO, no third-party deps.
//
//	Linux/X11   xclip -selection clipboard
//	Linux/Wayland wl-copy
//	macOS       pbcopy
//	Windows     clip
package clip

import (
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// ErrNoBackend is returned when no supported clipboard helper is found.
var ErrNoBackend = errors.New("no clipboard backend found")

// Copy writes s to the system clipboard.
func Copy(s string) error {
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

func pickCommand() (*exec.Cmd, error) {
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
	}
	return nil, ErrNoBackend
}
