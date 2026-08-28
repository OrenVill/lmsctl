package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"lmsctl/internal/lmstudio"
	"lmsctl/internal/output"
)

var modelsCmd = &cobra.Command{
	Use:     "models",
	Aliases: []string{"ls"},
	Short:   "List downloaded models and whether each is loaded",
	Args:    cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, _, err := newClient()
		if err != nil {
			return err
		}
		return runModels(cmd, client)
	},
}

func runModels(cmd *cobra.Command, client lmstudio.Client) error {
	resp, err := client.ListModels(cmd.Context())
	if err != nil {
		return err
	}

	if flagJSON {
		return output.JSON(cmd.OutOrStdout(), resp.Models)
	}

	tw := output.NewTable(cmd.OutOrStdout())
	fmt.Fprintln(tw, "KEY\tSIZE\tQUANTIZATION\tSTATE")
	for _, m := range resp.Models {
		quant := "-"
		if m.Quantization != nil {
			quant = m.Quantization.Name
		}
		state := "not-loaded"
		if len(m.LoadedInstances) > 0 {
			state = "loaded"
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", m.Key, formatBytes(m.SizeBytes), quant, state)
	}
	return tw.Flush()
}

func formatBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%dB", n)
	}
	div, exp := int64(unit), 0
	for x := n / unit; x >= unit; x /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%ciB", float64(n)/float64(div), "KMGTPE"[exp])
}

func init() {
	rootCmd.AddCommand(modelsCmd)
}
