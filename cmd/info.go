package cmd

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"lmsctl/internal/lmstudio"
	"lmsctl/internal/output"
)

var infoCmd = &cobra.Command{
	Use:   "info <model>",
	Short: "Show details for one downloaded model",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, _, err := newClient()
		if err != nil {
			return err
		}
		jsonOut, _ := cmd.Flags().GetBool("json")
		return runInfo(cmd, client, args[0], jsonOut)
	},
}

// fieldValue is one "Label: value" line in runInfo's output. yellow marks
// values that are quantitative (sizes, counts, context lengths) rather
// than free text (publisher, architecture, format).
type fieldValue struct {
	label, value string
	yellow       bool
}

func runInfo(cmd *cobra.Command, client lmstudio.Client, model string, jsonOut bool) error {
	resp, err := client.ListModels(cmd.Context())
	if err != nil {
		return err
	}

	var match *lmstudio.Model
	for i := range resp.Models {
		if resp.Models[i].Key == model {
			match = &resp.Models[i]
			break
		}
	}
	if match == nil {
		return &lmstudio.ErrModelNotFound{Model: model}
	}

	if jsonOut {
		return output.JSON(cmd.OutOrStdout(), match)
	}

	w := cmd.OutOrStdout()

	quant := "-"
	if match.Quantization != nil {
		quant = fmt.Sprintf("%s (%.2f bits/weight)", match.Quantization.Name, match.Quantization.BitsPerWeight)
	}

	printFields(w, []fieldValue{
		{"Key", match.Key, false},
		{"Publisher", match.Publisher, false},
		{"Display name", match.DisplayName, false},
		{"Architecture", derefOr(match.Architecture, "-"), false},
		{"Format", derefOr(match.Format, "-"), false},
		{"Quantization", quant, false},
		{"Size", formatBytes(match.SizeBytes), true},
		{"Max context", fmt.Sprintf("%d", match.MaxContextLength), true},
	})

	fmt.Fprintln(w)
	if len(match.LoadedInstances) == 0 {
		fmt.Fprintln(w, palette.Dim("Not currently loaded."))
		return nil
	}
	fmt.Fprintln(w, palette.Bold("Loaded instances:"))
	for _, inst := range match.LoadedInstances {
		fmt.Fprintln(w)
		fields := []fieldValue{
			{"Instance", inst.ID, false},
			{"Context length", fmt.Sprintf("%d", inst.Config.ContextLength), true},
			{"Flash attention", fmt.Sprintf("%t", inst.Config.FlashAttention), false},
			{"Offload KV to GPU", fmt.Sprintf("%t", inst.Config.OffloadKVCacheToGPU), false},
			{"Parallel", fmt.Sprintf("%d", inst.Config.Parallel), true},
			{"Eval batch size", fmt.Sprintf("%d", inst.Config.EvalBatchSize), true},
		}
		if inst.Config.NumExperts != 0 {
			fields = append(fields, fieldValue{"Num experts", fmt.Sprintf("%d", inst.Config.NumExperts), true})
		}
		printFields(w, fields)
	}
	return nil
}

// printFields prints label-aligned "Label: value" lines: labels are
// right-padded to the widest label in fields and bolded; "Key"/"Instance"
// values are colored cyan (they're identifiers), quantitative values
// yellow, everything else plain.
func printFields(w io.Writer, fields []fieldValue) {
	width := 0
	for _, f := range fields {
		if n := len(f.label) + 1; n > width { // +1 accounts for the trailing colon
			width = n
		}
	}
	for _, f := range fields {
		label := f.label + ":"
		value := f.value
		switch {
		case f.label == "Key" || f.label == "Instance":
			value = palette.Cyan(value)
		case f.yellow:
			value = palette.Yellow(value)
		}
		fmt.Fprintln(w, output.PadRight(label, width, palette.Bold)+" "+value)
	}
}

func derefOr(s *string, fallback string) string {
	if s == nil {
		return fallback
	}
	return *s
}

func init() {
	infoCmd.Flags().Bool("json", false, "output machine-readable JSON")
	rootCmd.AddCommand(infoCmd)
}
