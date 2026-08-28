package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"lmsctl/internal/lmstudio"
	"lmsctl/internal/output"
)

var loadCmd = &cobra.Command{
	Use:               "load <model>",
	Short:             "Load a model on the remote LM Studio instance",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeModelKeysFunc(false),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, _, err := newClient()
		if err != nil {
			return err
		}
		jsonOut, _ := cmd.Flags().GetBool("json")
		return runLoad(cmd, client, args[0], jsonOut)
	},
}

// buildLoadRequest builds a LoadModelRequest for model, including only the
// optional fields whose flag was explicitly set on fs (so e.g.
// --flash-attention=false is sent as false rather than omitted, while an
// unset flag lets LM Studio apply its own default).
func buildLoadRequest(fs *pflag.FlagSet, model string) lmstudio.LoadModelRequest {
	req := lmstudio.LoadModelRequest{Model: model}
	if fs.Changed("context-length") {
		v, _ := fs.GetInt("context-length")
		req.ContextLength = &v
	}
	if fs.Changed("flash-attention") {
		v, _ := fs.GetBool("flash-attention")
		req.FlashAttention = &v
	}
	if fs.Changed("offload-kv-cache-to-gpu") {
		v, _ := fs.GetBool("offload-kv-cache-to-gpu")
		req.OffloadKVCacheToGPU = &v
	}
	return req
}

func runLoad(cmd *cobra.Command, client lmstudio.Client, model string, jsonOut bool) error {
	req := buildLoadRequest(cmd.Flags(), model)

	resp, err := client.LoadModel(cmd.Context(), req)
	if err != nil {
		return err
	}

	if jsonOut {
		return output.JSON(cmd.OutOrStdout(), resp)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "%s %s as instance %s (%.1fs)\n", palette.Green("Loaded"), palette.Cyan(model), palette.Cyan(resp.InstanceID), resp.LoadTimeSeconds)
	return nil
}

func init() {
	loadCmd.Flags().Int("context-length", 0, "context length to load the model with")
	loadCmd.Flags().Bool("flash-attention", false, "enable flash attention (llama.cpp models only)")
	loadCmd.Flags().Bool("offload-kv-cache-to-gpu", false, "offload the KV cache to GPU (llama.cpp models only)")
	loadCmd.Flags().Bool("json", false, "output machine-readable JSON")
	rootCmd.AddCommand(loadCmd)
}
