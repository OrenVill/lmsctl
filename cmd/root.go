package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	flagHost  string
	flagToken string
	flagJSON  bool
)

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
	rootCmd.PersistentFlags().BoolVar(&flagJSON, "json", false, "output machine-readable JSON")
}
