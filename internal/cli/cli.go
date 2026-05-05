// Package cli wires CLI subcommands to the store and clipboard.
//
// The router is intentionally tiny: cmd/pe/main.go calls Run(args) and
// returns its exit code. Everything else lives behind testable methods
// on App so we can inject a temp store path and a fake clipboard.
package cli

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/Shin-R2un/pe/internal/clip"
	"github.com/Shin-R2un/pe/internal/store"
)

// Version is set at build time via -ldflags or stays at the default.
var Version = "0.2.0"

// App holds CLI dependencies. Zero value uses real defaults; tests
// override Path / Out / Err / Copy / Now.
type App struct {
	Path string             // snippet file path; "" → store.DefaultPath()
	Out  io.Writer          // stdout; nil → os.Stdout
	Err  io.Writer          // stderr; nil → os.Stderr
	Copy func(string) error // clipboard sink; nil → clip.Copy
	Now  func() time.Time   // clock; nil → time.Now
}

func (a *App) out() io.Writer {
	if a.Out == nil {
		return os.Stdout
	}
	return a.Out
}

func (a *App) err() io.Writer {
	if a.Err == nil {
		return os.Stderr
	}
	return a.Err
}

func (a *App) copyFn() func(string) error {
	if a.Copy != nil {
		return a.Copy
	}
	return clip.Copy
}

func (a *App) now() time.Time {
	if a.Now != nil {
		return a.Now()
	}
	return time.Now()
}

func (a *App) path() (string, error) {
	if a.Path != "" {
		return a.Path, nil
	}
	return store.DefaultPath()
}

func (a *App) load() (*store.File, string, error) {
	p, err := a.path()
	if err != nil {
		return nil, "", err
	}
	f, err := store.Load(p)
	if err != nil {
		return nil, p, err
	}
	return f, p, nil
}

// Run executes argv (without the program name) and returns an exit code:
//
//	0 — success
//	1 — runtime error (not found, already exists, no clipboard, etc.)
//	2 — usage error (bad / missing arguments, unknown subcommand)
func (a *App) Run(args []string) int {
	if len(args) == 0 {
		return a.cmdInteractive()
	}
	head := args[0]
	rest := args[1:]
	switch head {
	case "-h", "--help", "help":
		fmt.Fprint(a.out(), helpText)
		return 0
	case "-v", "--version", "version":
		fmt.Fprintln(a.out(), Version)
		return 0
	case "a", "add":
		return a.cmdAdd(rest)
	case "l", "list", "ls":
		return a.cmdList(rest)
	case "s", "search", "find":
		return a.cmdSearch(rest)
	case "?", "show":
		return a.cmdShow(rest)
	case "e", "edit":
		return a.cmdEdit(rest)
	case "d", "delete", "rm":
		return a.cmdDelete(rest)
	case "completion":
		return a.cmdCompletion(rest)
	case "__complete":
		return a.cmdInternalComplete(rest)
	case "update":
		return a.cmdUpdate(rest)
	}
	// Anything else is treated as `pe <key>` (copy by key).
	if len(rest) > 0 {
		fmt.Fprintf(a.err(), "pe: unknown subcommand %q (did you mean `pe a %s ...` to register?)\n", head, head)
		return 2
	}
	return a.cmdCopy(head)
}

const helpText = "pe — copy a saved snippet and paste it instantly.\n" +
	"\n" +
	"Usage:\n" +
	"  pe <key>              copy snippet to clipboard\n" +
	"  pe                    interactive search\n" +
	"  pe a   <key> <text>   add a snippet         (alias: add)\n" +
	"  pe l                  list snippets         (alias: list)\n" +
	"  pe s   <query>        search snippets       (alias: search)\n" +
	"  pe ?   <key>          show snippet contents (alias: show)\n" +
	"  pe e   <key>          edit a snippet        (alias: edit)\n" +
	"  pe d   <key>          delete a snippet      (alias: delete)\n" +
	"  pe completion <sh>    print bash/zsh/fish tab-completion\n" +
	"  pe update             reinstall the latest release via `go install`\n" +
	"  pe help               this message\n" +
	"  pe version            print version\n" +
	"\n" +
	"Environment:\n" +
	"  PE_DIR     override snippet directory (default: ~/.pe)\n" +
	"  PE_EDITOR  override editor for `pe e`\n" +
	"  EDITOR / VISUAL also honored\n" +
	"\n" +
	"Storage: ~/.pe/pe.json (JSON, 0600). Do not store secrets — use kpot.\n"

func (a *App) errorf(format string, args ...interface{}) int {
	fmt.Fprintf(a.err(), format, args...)
	if len(format) == 0 || format[len(format)-1] != '\n' {
		fmt.Fprintln(a.err())
	}
	return 1
}

func (a *App) usage(format string, args ...interface{}) int {
	fmt.Fprintf(a.err(), format, args...)
	if len(format) == 0 || format[len(format)-1] != '\n' {
		fmt.Fprintln(a.err())
	}
	return 2
}
