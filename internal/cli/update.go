package cli

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// modulePath is the canonical Go import path used by `pe update`.
// Kept as a const here (not configurable) because forking pe means
// you ship your own update story too.
const modulePath = "github.com/Shin-R2un/pe/cmd/pe@latest"

// installer is overridable for tests; default shells out to `go install`.
var installer = goInstall

// goEnvFn is overridable for tests.
var goEnvFn = goEnv

// versionReader is overridable for tests; default invokes the new binary.
var versionReader = readBinaryVersion

func (a *App) cmdUpdate(args []string) int {
	_ = args
	goPath, err := exec.LookPath("go")
	if err != nil {
		fmt.Fprintln(a.err(), "pe update needs the `go` toolchain on PATH.")
		fmt.Fprintln(a.err(), "options:")
		fmt.Fprintln(a.err(), "  - install Go: https://go.dev/dl/")
		fmt.Fprintln(a.err(), "  - or grab a release: https://github.com/Shin-R2un/pe/releases")
		return 1
	}

	old := Version
	fmt.Fprintf(a.out(), "running: go install %s\n", modulePath)
	if out, err := installer(goPath); err != nil {
		fmt.Fprintf(a.err(), "go install failed: %v\n", err)
		if trimmed := strings.TrimSpace(out); trimmed != "" {
			fmt.Fprintln(a.err(), trimmed)
		}
		fmt.Fprintln(a.err(), "if this is a freshly-pushed release, the Go module proxy")
		fmt.Fprintln(a.err(), "may still be caching a 404 — retry with: GOPROXY=direct pe update")
		return 1
	}

	binPath, err := installedBinaryPath(goPath)
	if err != nil {
		fmt.Fprintf(a.out(), "updated. run `pe version` to confirm. (%v)\n", err)
		return 0
	}
	newVer, err := versionReader(binPath)
	if err != nil {
		fmt.Fprintf(a.out(), "updated. run `pe version` to confirm. (%v)\n", err)
		return 0
	}
	if newVer == old {
		fmt.Fprintf(a.out(), "already up to date (%s)\n", newVer)
	} else {
		fmt.Fprintf(a.out(), "updated: %s → %s\n", old, newVer)
	}
	return 0
}

func goInstall(goPath string) (string, error) {
	cmd := exec.Command(goPath, "install", modulePath)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// installedBinaryPath asks `go env` where `go install` just landed pe.
// Honors GOBIN if set, else falls back to $GOPATH/bin.
func installedBinaryPath(goPath string) (string, error) {
	bin, err := goEnvFn(goPath, "GOBIN")
	if err != nil {
		return "", err
	}
	if bin == "" {
		gopath, err := goEnvFn(goPath, "GOPATH")
		if err != nil {
			return "", err
		}
		if gopath == "" {
			return "", fmt.Errorf("neither GOBIN nor GOPATH is set")
		}
		bin = filepath.Join(gopath, "bin")
	}
	exe := "pe"
	if runtime.GOOS == "windows" {
		exe = "pe.exe"
	}
	return filepath.Join(bin, exe), nil
}

func goEnv(goPath, key string) (string, error) {
	out, err := exec.Command(goPath, "env", key).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func readBinaryVersion(path string) (string, error) {
	out, err := exec.Command(path, "version").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
