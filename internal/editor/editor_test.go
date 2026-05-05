package editor

import (
	"runtime"
	"testing"
)

// pick() resolves the external editor command. It's a pure function of
// the environment, so we exercise the precedence ladder and the
// platform default.

func TestPickPrefersPEEditor(t *testing.T) {
	t.Setenv("PE_EDITOR", "nvim")
	t.Setenv("VISUAL", "code -w")
	t.Setenv("EDITOR", "vim")

	cmd, args := pick()
	if cmd != "nvim" {
		t.Errorf("cmd = %q, want nvim (PE_EDITOR wins)", cmd)
	}
	if len(args) != 0 {
		t.Errorf("args = %v, want []", args)
	}
}

func TestPickFallsBackToVISUAL(t *testing.T) {
	t.Setenv("PE_EDITOR", "")
	t.Setenv("VISUAL", "code -w")
	t.Setenv("EDITOR", "vim")

	cmd, args := pick()
	if cmd != "code" {
		t.Errorf("cmd = %q, want code (VISUAL wins after PE_EDITOR empty)", cmd)
	}
	if len(args) != 1 || args[0] != "-w" {
		t.Errorf("args = %v, want [-w]", args)
	}
}

func TestPickFallsBackToEDITOR(t *testing.T) {
	t.Setenv("PE_EDITOR", "")
	t.Setenv("VISUAL", "")
	t.Setenv("EDITOR", "vim")

	cmd, args := pick()
	if cmd != "vim" {
		t.Errorf("cmd = %q, want vim", cmd)
	}
	if len(args) != 0 {
		t.Errorf("args = %v, want []", args)
	}
}

func TestPickPlatformDefault(t *testing.T) {
	t.Setenv("PE_EDITOR", "")
	t.Setenv("VISUAL", "")
	t.Setenv("EDITOR", "")

	cmd, args := pick()
	want := "vi"
	if runtime.GOOS == "windows" {
		want = "notepad"
	}
	if cmd != want {
		t.Errorf("cmd = %q, want %q (platform default)", cmd, want)
	}
	if len(args) != 0 {
		t.Errorf("args = %v, want []", args)
	}
}

func TestPickWhitespaceTreatedAsEmpty(t *testing.T) {
	t.Setenv("PE_EDITOR", "   ")
	t.Setenv("VISUAL", "")
	t.Setenv("EDITOR", "vim")

	cmd, _ := pick()
	if cmd != "vim" {
		t.Errorf("cmd = %q, want vim (whitespace-only PE_EDITOR should be ignored)", cmd)
	}
}
