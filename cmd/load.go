package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

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

func runLoad(cmd *cobra.Command, client lmstudio.Client, model string, jsonOut bool) error {
	req := lmstudio.LoadModelRequest{Model: model}
	if cmd.Flags().Changed("context-length") {
		v := loadFlagContextLength
		req.ContextLength = &v
	}
	if cmd.Flags().Changed("flash-attention") {
		v := loadFlagFlashAttn
		req.FlashAttention = &v
	}
	if cmd.Flags().Changed("offload-kv-cache-to-gpu") {
		v := loadFlagOffloadKV
		req.OffloadKVCacheToGPU = &v
	}

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
