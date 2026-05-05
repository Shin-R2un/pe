package cli

import (
	"fmt"
	"os"

	"github.com/Shin-R2un/pe/internal/store"
	"golang.org/x/term"
)

const interactiveLimit = 10

// cmdInteractive runs the no-arg `pe` flow: incremental fuzzy filter,
// number-key quick-pick, Enter for top hit, Ctrl-C / Esc to cancel.
func (a *App) cmdInteractive() int {
	f, p, err := a.load()
	if err != nil {
		return a.errorf("load %s: %v", p, err)
	}
	if len(f.Snippets) == 0 {
		fmt.Fprintln(a.out(), "no snippets yet — try `pe a <key> <text>`")
		return 0
	}

	stdin := int(os.Stdin.Fd())
	stdout := int(os.Stdout.Fd())
	if !term.IsTerminal(stdin) || !term.IsTerminal(stdout) {
		return a.errorf("interactive mode needs a TTY (use `pe l` / `pe s <q>` instead)")
	}

	old, err := term.MakeRaw(stdin)
	if err != nil {
		return a.errorf("raw mode: %v", err)
	}
	defer term.Restore(stdin, old)

	query := ""
	prevLines := 0
	picked := -1
	hits := f.Search("")

	for {
		// Erase previous frame.
		if prevLines > 0 {
			for i := 0; i < prevLines; i++ {
				fmt.Fprint(os.Stdout, "\x1b[1A\x1b[2K")
			}
			fmt.Fprint(os.Stdout, "\r")
		}

		hits = f.Search(query)
		drawn := drawFrame(query, hits)
		prevLines = drawn

		var buf [4]byte
		n, err := os.Stdin.Read(buf[:])
		if err != nil || n == 0 {
			break
		}
		b := buf[0]
		switch {
		case b == 0x03 || b == 0x1b: // Ctrl-C / Esc → cancel
			eraseFrame(prevLines)
			fmt.Fprint(os.Stdout, "\r")
			return 1
		case b == '\r' || b == '\n': // Enter → first hit
			if len(hits) > 0 {
				picked = 0
			}
		case b >= '1' && b <= '9':
			idx := int(b-'0') - 1
			if idx < len(hits) && idx < interactiveLimit {
				picked = idx
			}
		case b == 0x7f || b == 0x08: // Backspace
			if len(query) > 0 {
				query = query[:len(query)-1]
			}
		default:
			if b >= 0x20 && b < 0x7f {
				query += string(rune(b))
			}
		}
		if picked >= 0 {
			break
		}
	}

	eraseFrame(prevLines)
	fmt.Fprint(os.Stdout, "\r")
	term.Restore(stdin, old)

	if picked < 0 || picked >= len(hits) {
		return 1
	}
	chosen := hits[picked]
	if err := a.copyFn()(chosen.Value); err != nil {
		return a.errorf("copy: %v", err)
	}
	if err := f.Touch(chosen.Key, a.now()); err == nil {
		_ = store.Save(p, f)
	}
	fmt.Fprintf(a.out(), "copied: %s\n", chosen.Key)
	return 0
}

// drawFrame writes the search prompt + filtered hits and returns how
// many terminal lines were drawn.
func drawFrame(query string, hits []store.Snippet) int {
	fmt.Fprintf(os.Stdout, "search: %s\r\n", query)
	if len(hits) == 0 {
		fmt.Fprint(os.Stdout, "  (no matches)\r\n")
		return 2
	}
	limit := len(hits)
	if limit > interactiveLimit {
		limit = interactiveLimit
	}
	width := 0
	for i := 0; i < limit; i++ {
		if l := len(hits[i].Key); l > width {
			width = l
		}
	}
	if width > 24 {
		width = 24
	}
	for i := 0; i < limit; i++ {
		fmt.Fprintf(os.Stdout, "  %d) %-*s  %s\r\n", i+1, width, hits[i].Key, preview(hits[i], 50))
	}
	return 1 + limit
}

func eraseFrame(n int) {
	for i := 0; i < n; i++ {
		fmt.Fprint(os.Stdout, "\x1b[1A\x1b[2K")
	}
}
