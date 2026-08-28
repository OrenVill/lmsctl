package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"lmsctl/internal/config"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "View or change lmsctl's configuration",
}

var configSetHostCmd = &cobra.Command{
	Use:   "set-host <host:port>",
	Short: "Set the default remote LM Studio host",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		cfg.Host = args[0]
		if err := config.Save(cfg); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Default host set to %s\n", args[0])
		return nil
	},
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Print the effective configuration (token redacted)",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		fileCfg, err := config.Load()
		if err != nil {
			return err
		}
		eff := config.EffectiveDisplay(flagHost, flagToken, os.Getenv("LMSCTL_HOST"), os.Getenv("LMSCTL_TOKEN"), fileCfg)
		host := eff.Host
		if host == "" {
			host = "(not set)"
		}
		token := "(not set)"
		if eff.Token != "" {
			token = "(set)"
		}
		fmt.Fprintf(cmd.OutOrStdout(), "host:  %s\ntoken: %s\n", host, token)
		return nil
	},
}

func init() {
	configCmd.AddCommand(configSetHostCmd, configShowCmd)
	rootCmd.AddCommand(configCmd)
}
