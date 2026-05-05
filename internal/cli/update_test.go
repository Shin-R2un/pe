package cli

import (
	"errors"
	"strings"
	"testing"
)

// withMockUpdater swaps the package-level installer / goEnvFn /
// versionReader hooks for the duration of a test.
func withMockUpdater(t *testing.T, install func(string) (string, error), env func(string, string) (string, error), readVer func(string) (string, error)) {
	t.Helper()
	origI, origE, origV := installer, goEnvFn, versionReader
	t.Cleanup(func() {
		installer = origI
		goEnvFn = origE
		versionReader = origV
	})
	if install != nil {
		installer = install
	}
	if env != nil {
		goEnvFn = env
	}
	if readVer != nil {
		versionReader = readVer
	}
}

func TestUpdateReportsVersionDiff(t *testing.T) {
	app, out, _, _ := newTestApp(t)
	origVer := Version
	t.Cleanup(func() { Version = origVer })
	Version = "0.2.0"

	withMockUpdater(t,
		func(string) (string, error) { return "", nil },
		func(_, key string) (string, error) {
			if key == "GOBIN" {
				return "/fake/gobin", nil
			}
			return "", nil
		},
		func(string) (string, error) { return "0.3.0", nil },
	)

	if code := app.Run([]string{"update"}); code != 0 {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(out.String(), "updated: 0.2.0 → 0.3.0") {
		t.Errorf("expected version diff line, got %q", out)
	}
}

func TestUpdateReportsAlreadyUpToDate(t *testing.T) {
	app, out, _, _ := newTestApp(t)
	origVer := Version
	t.Cleanup(func() { Version = origVer })
	Version = "0.2.0"

	withMockUpdater(t,
		func(string) (string, error) { return "", nil },
		func(_, key string) (string, error) {
			if key == "GOBIN" {
				return "/fake/gobin", nil
			}
			return "", nil
		},
		func(string) (string, error) { return "0.2.0", nil },
	)

	if code := app.Run([]string{"update"}); code != 0 {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(out.String(), "already up to date") {
		t.Errorf("expected already-up-to-date line, got %q", out)
	}
}

func TestUpdateInstallFailureGivesProxyHint(t *testing.T) {
	app, _, errBuf, _ := newTestApp(t)

	withMockUpdater(t,
		func(string) (string, error) { return "module not found", errors.New("exit 1") },
		nil, nil,
	)

	code := app.Run([]string{"update"})
	if code != 1 {
		t.Errorf("exit = %d, want 1", code)
	}
	body := errBuf.String()
	if !strings.Contains(body, "go install failed") {
		t.Errorf("missing failure header: %q", body)
	}
	if !strings.Contains(body, "GOPROXY=direct") {
		t.Errorf("missing proxy hint: %q", body)
	}
}

func TestUpdateGracefulIfPostInstallProbeFails(t *testing.T) {
	app, out, _, _ := newTestApp(t)

	withMockUpdater(t,
		func(string) (string, error) { return "", nil },
		func(_, key string) (string, error) { return "", nil }, // no GOBIN, no GOPATH
		nil,
	)

	if code := app.Run([]string{"update"}); code != 0 {
		t.Fatalf("exit = %d", code)
	}
	body := out.String()
	if !strings.Contains(body, "updated.") {
		t.Errorf("expected fallback success message, got %q", body)
	}
	if !strings.Contains(body, "pe version") {
		t.Errorf("expected pointer to `pe version`, got %q", body)
	}
}

func TestAddRejectsUpdateAsKey(t *testing.T) {
	app, _, errBuf, _ := newTestApp(t)
	if code := app.Run([]string{"a", "update", "x"}); code != 1 {
		t.Errorf("exit = %d, want 1", code)
	}
	if !strings.Contains(errBuf.String(), "reserved keyword") {
		t.Errorf("err = %q", errBuf)
	}
}
