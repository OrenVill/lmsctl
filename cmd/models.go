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
		jsonOut, _ := cmd.Flags().GetBool("json")
		return runModels(cmd, client, jsonOut)
	},
}

type modelRow struct {
	key, size, quant, state string
	loaded                  bool
}

func runModels(cmd *cobra.Command, client lmstudio.Client, jsonOut bool) error {
	resp, err := client.ListModels(cmd.Context())
	if err != nil {
		return err
	}

	if jsonOut {
		models := resp.Models
		if models == nil {
			models = []lmstudio.Model{}
		}
		return output.JSON(cmd.OutOrStdout(), models)
	}

	rows := make([]modelRow, 0, len(resp.Models))
	for _, m := range resp.Models {
		quant := "-"
		if m.Quantization != nil {
			quant = m.Quantization.Name
		}
		loaded := len(m.LoadedInstances) > 0
		state := "not-loaded"
		if loaded {
			state = "loaded"
		}
		rows = append(rows, modelRow{key: m.Key, size: formatBytes(m.SizeBytes), quant: quant, state: state, loaded: loaded})
	}

	const headerKey, headerSize, headerQuant, headerState = "KEY", "SIZE", "QUANTIZATION", "STATE"
	keyW, sizeW, quantW := len(headerKey), len(headerSize), len(headerQuant)
	for _, r := range rows {
		if len(r.key) > keyW {
			keyW = len(r.key)
		}
		if len(r.size) > sizeW {
			sizeW = len(r.size)
		}
		if len(r.quant) > quantW {
			quantW = len(r.quant)
		}
	}

	w := cmd.OutOrStdout()
	header := output.PadRight(headerKey, keyW, palette.Bold) + "  " +
		output.PadRight(headerSize, sizeW, palette.Bold) + "  " +
		output.PadRight(headerQuant, quantW, palette.Bold) + "  " +
		palette.Bold(headerState)
	fmt.Fprintln(w, header)

	for _, r := range rows {
		stateStyle := palette.Dim
		if r.loaded {
			stateStyle = palette.Green
		}
		line := output.PadRight(r.key, keyW, palette.Cyan) + "  " +
			output.PadRight(r.size, sizeW, palette.Yellow) + "  " +
			output.PadRight(r.quant, quantW, palette.Dim) + "  " +
			stateStyle(r.state)
		fmt.Fprintln(w, line)
	}
	return nil
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
	modelsCmd.Flags().Bool("json", false, "output machine-readable JSON")
	rootCmd.AddCommand(modelsCmd)
}
