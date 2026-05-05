// Package editor opens an external editor on a temp file and returns
// the edited bytes. It deliberately keeps no state — callers serialize
// what they want, hand a hint at the file extension, and parse what
// comes back.
package editor

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// ErrUnchanged is returned when the user saved without modifying the file.
var ErrUnchanged = errors.New("editor: file unchanged")

// Edit launches an editor on a temp file pre-filled with `initial`.
// The temp file is named `<basename>.<ext>` so editors pick the right
// syntax. Returns the edited bytes or ErrUnchanged.
func Edit(basename, ext string, initial []byte) ([]byte, error) {
	dir, err := os.MkdirTemp("", "pe-edit-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)

	path := filepath.Join(dir, basename+"."+strings.TrimPrefix(ext, "."))
	if err := os.WriteFile(path, initial, 0o600); err != nil {
		return nil, err
	}

	editor, args := pick()
	args = append(args, path)
	cmd := exec.Command(editor, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("editor %q failed: %w", editor, err)
	}

	edited, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if string(edited) == string(initial) {
		return edited, ErrUnchanged
	}
	return edited, nil
}

// pick resolves the editor command and any leading flags.
// Priority: PE_EDITOR > VISUAL > EDITOR > vi (Linux/macOS) / notepad (Windows).
func pick() (string, []string) {
	for _, env := range []string{"PE_EDITOR", "VISUAL", "EDITOR"} {
		if v := strings.TrimSpace(os.Getenv(env)); v != "" {
			parts := strings.Fields(v)
			return parts[0], parts[1:]
		}
	}
	if runtime.GOOS == "windows" {
		return "notepad", nil
	}
	return "vi", nil
}
