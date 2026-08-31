package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func newTestCmd() *cobra.Command {
	root := &cobra.Command{Use: "lmsctl"}
	root.SetOut(&bytes.Buffer{})
	return root
}

func TestInstallViaRCFile_CreatesFileAndAppendsSourceLine(t *testing.T) {
	dir := t.TempDir()
	rcPath := filepath.Join(dir, ".bashrc")
	cmd := newTestCmd()

	if err := installViaRCFile(cmd, rcPath, "bash"); err != nil {
		t.Fatalf("installViaRCFile: %v", err)
	}

	data, err := os.ReadFile(rcPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !strings.Contains(string(data), "source <(lmsctl completion bash)") {
		t.Errorf("rc file = %q, want it to contain the source line", data)
	}
}

func TestInstallViaRCFile_AppendsToExistingContentWithoutOverwriting(t *testing.T) {
	dir := t.TempDir()
	rcPath := filepath.Join(dir, ".bashrc")
	if err := os.WriteFile(rcPath, []byte("export FOO=bar\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	cmd := newTestCmd()

	if err := installViaRCFile(cmd, rcPath, "bash"); err != nil {
		t.Fatalf("installViaRCFile: %v", err)
	}

	data, err := os.ReadFile(rcPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !strings.Contains(string(data), "export FOO=bar") {
		t.Errorf("rc file lost pre-existing content: %q", data)
	}
	if !strings.Contains(string(data), "source <(lmsctl completion bash)") {
		t.Errorf("rc file = %q, want it to contain the source line", data)
	}
}

func TestInstallViaRCFile_IsIdempotent(t *testing.T) {
	dir := t.TempDir()
	rcPath := filepath.Join(dir, ".zshrc")
	cmd := newTestCmd()

	if err := installViaRCFile(cmd, rcPath, "zsh"); err != nil {
		t.Fatalf("installViaRCFile (first): %v", err)
	}
	first, _ := os.ReadFile(rcPath)

	if err := installViaRCFile(cmd, rcPath, "zsh"); err != nil {
		t.Fatalf("installViaRCFile (second): %v", err)
	}
	second, err := os.ReadFile(rcPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	if string(first) != string(second) {
		t.Errorf("second install changed the file: before %q, after %q", first, second)
	}
	if strings.Count(string(second), "source <(lmsctl completion zsh)") != 1 {
		t.Errorf("rc file = %q, want the source line exactly once", second)
	}
}

func TestInstallFish_WritesCompletionFileUnderConfigDir(t *testing.T) {
	home := t.TempDir()
	cmd := newTestCmd()

	if err := installFish(cmd, home); err != nil {
		t.Fatalf("installFish: %v", err)
	}

	path := filepath.Join(home, ".config", "fish", "completions", "lmsctl.fish")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", path, err)
	}
	if !strings.Contains(string(data), "lmsctl") {
		t.Errorf("fish completion file looks empty/wrong: %q", data)
	}
}

func TestInstallCompletion_UnsupportedShellReturnsError(t *testing.T) {
	cmd := newTestCmd()
	if err := installCompletion(cmd, "powershell"); err == nil {
		t.Error("installCompletion(powershell) = nil error, want an error explaining --install isn't supported")
	}
}

func TestWriteCompletionScript_UnsupportedShellReturnsError(t *testing.T) {
	var buf bytes.Buffer
	if err := writeCompletionScript(&buf, newTestCmd(), "tcsh"); err == nil {
		t.Error("writeCompletionScript(tcsh) = nil error, want an error naming supported shells")
	}
}

func TestWriteCompletionScript_BashWritesNonEmptyScript(t *testing.T) {
	var buf bytes.Buffer
	if err := writeCompletionScript(&buf, newTestCmd(), "bash"); err != nil {
		t.Fatalf("writeCompletionScript: %v", err)
	}
	if buf.Len() == 0 {
		t.Error("writeCompletionScript(bash) wrote nothing")
	}
}
