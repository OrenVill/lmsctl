package color

import (
	"os"
	"testing"
)

func TestPalette_EnabledWrapsWithAnsiCodesAndReset(t *testing.T) {
	p := Palette{Enabled: true}

	if got, want := p.Green("x"), "\033[32mx\033[0m"; got != want {
		t.Errorf("Green(%q) = %q, want %q", "x", got, want)
	}
	if got, want := p.Dim("x"), "\033[2mx\033[0m"; got != want {
		t.Errorf("Dim(%q) = %q, want %q", "x", got, want)
	}
	if got, want := p.Bold("x"), "\033[1mx\033[0m"; got != want {
		t.Errorf("Bold(%q) = %q, want %q", "x", got, want)
	}
}

func TestPalette_DisabledReturnsPlainText(t *testing.T) {
	p := Palette{Enabled: false}

	if got := p.Green("x"); got != "x" {
		t.Errorf("Green(%q) = %q, want unmodified %q", "x", got, "x")
	}
	if got := p.Dim("x"); got != "x" {
		t.Errorf("Dim(%q) = %q, want unmodified %q", "x", got, "x")
	}
	if got := p.Bold("x"); got != "x" {
		t.Errorf("Bold(%q) = %q, want unmodified %q", "x", got, "x")
	}
}

func TestNew_NoColorEnvDisablesRegardlessOfTerminal(t *testing.T) {
	t.Setenv("NO_COLOR", "1")

	p := New(os.Stdout)
	if p.Enabled {
		t.Error("Enabled = true, want false when NO_COLOR is set")
	}
}

func TestNew_NonTerminalFileDisablesColor(t *testing.T) {
	t.Setenv("NO_COLOR", "")

	f, err := os.CreateTemp(t.TempDir(), "not-a-terminal")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}
	defer f.Close()

	p := New(f)
	if p.Enabled {
		t.Error("Enabled = true, want false for a regular file (not a terminal)")
	}
}
