package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var completionInstall bool

// completionCmd replaces cobra's auto-generated "completion" command (see
// init below) so it can offer --install: writing the script to the right
// place for the user's shell instead of just printing it to stdout.
var completionCmd = &cobra.Command{
	Use:       "completion [bash|zsh|fish|powershell]",
	Short:     "Generate (or install) the autocompletion script for the specified shell",
	Args:      cobra.ExactArgs(1),
	ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
	RunE: func(cmd *cobra.Command, args []string) error {
		shell := args[0]
		if completionInstall {
			return installCompletion(cmd, shell)
		}
		return writeCompletionScript(cmd.OutOrStdout(), cmd.Root(), shell)
	},
}

// writeCompletionScript writes root's completion script for shell to w,
// unchanged from cobra's default "completion <shell>" behavior.
func writeCompletionScript(w io.Writer, root *cobra.Command, shell string) error {
	switch shell {
	case "bash":
		return root.GenBashCompletionV2(w, true)
	case "zsh":
		return root.GenZshCompletion(w)
	case "fish":
		return root.GenFishCompletion(w, true)
	case "powershell":
		return root.GenPowerShellCompletionWithDesc(w)
	default:
		return fmt.Errorf("unsupported shell %q (want bash, zsh, fish, or powershell)", shell)
	}
}

// installCompletion installs shell's completion script to the conventional
// location for that shell, using cmd's resolved home directory.
func installCompletion(cmd *cobra.Command, shell string) error {
	switch shell {
	case "bash":
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("finding home directory: %w", err)
		}
		return installViaRCFile(cmd, filepath.Join(home, ".bashrc"), "bash")
	case "zsh":
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("finding home directory: %w", err)
		}
		return installViaRCFile(cmd, filepath.Join(home, ".zshrc"), "zsh")
	case "fish":
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("finding home directory: %w", err)
		}
		return installFish(cmd, home)
	default:
		return fmt.Errorf("--install isn't supported for %q; run \"lmsctl completion %s\" and install the script manually", shell, shell)
	}
}

// installViaRCFile appends a line sourcing "lmsctl completion <shell>" to
// rcPath, creating the file if it doesn't exist. It's a no-op if that line
// is already present, so running --install more than once is harmless.
func installViaRCFile(cmd *cobra.Command, rcPath, shell string) error {
	line := fmt.Sprintf("source <(lmsctl completion %s)", shell)

	data, err := os.ReadFile(rcPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("reading %s: %w", rcPath, err)
	}
	if strings.Contains(string(data), line) {
		fmt.Fprintf(cmd.OutOrStdout(), "%s already sources lmsctl completion; nothing to do.\n", rcPath)
		return nil
	}

	f, err := os.OpenFile(rcPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("opening %s: %w", rcPath, err)
	}
	defer f.Close()
	if _, err := fmt.Fprintf(f, "\n# Added by \"lmsctl completion %s --install\"\n%s\n", shell, line); err != nil {
		return fmt.Errorf("writing to %s: %w", rcPath, err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "%s %s to source lmsctl completions in new shells.\nOpen a new terminal (or run \"source %s\") to start using it.\n",
		palette.Green("Updated"), rcPath, rcPath)
	return nil
}

// installFish writes the fish completion script to the directory fish
// auto-loads completions from, overwriting any previous copy -- fish needs
// no rc-file edit, so unlike installViaRCFile this is naturally idempotent.
func installFish(cmd *cobra.Command, home string) error {
	dir := filepath.Join(home, ".config", "fish", "completions")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating %s: %w", dir, err)
	}
	path := filepath.Join(dir, "lmsctl.fish")
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating %s: %w", path, err)
	}
	defer f.Close()
	if err := cmd.Root().GenFishCompletion(f, true); err != nil {
		return fmt.Errorf("generating fish completion: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "%s %s\nOpen a new terminal to start using it.\n", palette.Green("Wrote"), path)
	return nil
}

func init() {
	completionCmd.Flags().BoolVar(&completionInstall, "install", false,
		"install the completion script instead of printing it (appends a source line to ~/.bashrc or ~/.zshrc; writes ~/.config/fish/completions/lmsctl.fish for fish)")
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.AddCommand(completionCmd)
}
