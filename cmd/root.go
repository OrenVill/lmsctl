package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"lmsctl/internal/color"
	"lmsctl/internal/config"
	"lmsctl/internal/lmstudio"
)

var (
	flagHost  string
	flagToken string
)

// palette is used by commands to style human-readable output. It's
// disabled automatically when stdout isn't a terminal or NO_COLOR is set,
// so it's safe to call unconditionally.
var palette = color.New(os.Stdout)

var rootCmd = &cobra.Command{
	Use:           "lmsctl",
	Short:         "Manage a remote LM Studio instance from the command line",
	SilenceErrors: true,
	SilenceUsage:  true,
}

// Execute runs the root command and exits the process on error.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&flagHost, "host", "", "LM Studio host:port (overrides config file and LMSCTL_HOST)")
	rootCmd.PersistentFlags().StringVar(&flagToken, "token", "", "LM Studio API token (overrides config file and LMSCTL_TOKEN)")
}

// newClient resolves the effective host/token and builds a real
// lmstudio.Client for a command to use.
func newClient() (lmstudio.Client, config.Effective, error) {
	fileCfg, err := config.Load()
	if err != nil {
		return nil, config.Effective{}, err
	}
	eff, err := config.Resolve(flagHost, flagToken, os.Getenv("LMSCTL_HOST"), os.Getenv("LMSCTL_TOKEN"), fileCfg)
	if err != nil {
		return nil, config.Effective{}, err
	}
	return lmstudio.NewHTTPClient(eff.Host, eff.Token), eff, nil
}
