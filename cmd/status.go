package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"lmsctl/internal/lmstudio"
	"lmsctl/internal/output"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check whether the remote LM Studio server is reachable and what's loaded",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, eff, err := newClient()
		if err != nil {
			return err
		}
		return runStatus(cmd, client, eff.Host)
	},
}

func runStatus(cmd *cobra.Command, client lmstudio.Client, host string) error {
	resp, err := client.ListModels(cmd.Context())
	if err != nil {
		return err
	}

	var loaded []string
	for _, m := range resp.Models {
		for _, inst := range m.LoadedInstances {
			loaded = append(loaded, fmt.Sprintf("%s (%s)", m.Key, inst.ID))
		}
	}

	if flagJSON {
		return output.JSON(cmd.OutOrStdout(), map[string]any{
			"reachable":     true,
			"host":          host,
			"loaded_models": loaded,
		})
	}

	fmt.Fprintf(cmd.OutOrStdout(), "LM Studio at %s: reachable\n", host)
	if len(loaded) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No models currently loaded.")
		return nil
	}
	fmt.Fprintln(cmd.OutOrStdout(), "Loaded models:")
	for _, l := range loaded {
		fmt.Fprintf(cmd.OutOrStdout(), "  - %s\n", l)
	}
	return nil
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
