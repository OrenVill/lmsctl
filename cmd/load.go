package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"lmsctl/internal/lmstudio"
	"lmsctl/internal/output"
)

var (
	loadFlagContextLength int
	loadFlagFlashAttn     bool
	loadFlagOffloadKV     bool
)

var loadCmd = &cobra.Command{
	Use:   "load <model>",
	Short: "Load a model on the remote LM Studio instance",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, _, err := newClient()
		if err != nil {
			return err
		}
		return runLoad(cmd, client, args[0], flagJSON)
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
	fmt.Fprintf(cmd.OutOrStdout(), "Loaded %s as instance %s (%.1fs)\n", model, resp.InstanceID, resp.LoadTimeSeconds)
	return nil
}

func init() {
	loadCmd.Flags().IntVar(&loadFlagContextLength, "context-length", 0, "context length to load the model with")
	loadCmd.Flags().BoolVar(&loadFlagFlashAttn, "flash-attention", false, "enable flash attention (llama.cpp models only)")
	loadCmd.Flags().BoolVar(&loadFlagOffloadKV, "offload-kv-cache-to-gpu", false, "offload the KV cache to GPU (llama.cpp models only)")
	rootCmd.AddCommand(loadCmd)
}
