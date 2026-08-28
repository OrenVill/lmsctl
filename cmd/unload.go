package cmd

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"lmsctl/internal/lmstudio"
)

var unloadFlagAll bool

var unloadCmd = &cobra.Command{
	Use:   "unload [model]",
	Short: "Unload a model (or all loaded models) on the remote LM Studio instance",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var model string
		if len(args) == 1 {
			model = args[0]
		}
		if unloadFlagAll && model != "" {
			return errors.New("pass either a model or --all, not both")
		}
		if !unloadFlagAll && model == "" {
			return errors.New("specify a model to unload or pass --all")
		}
		client, _, err := newClient()
		if err != nil {
			return err
		}
		return runUnload(cmd, client, model, unloadFlagAll)
	},
}

func runUnload(cmd *cobra.Command, client lmstudio.Client, model string, all bool) error {
	if all && model != "" {
		return errors.New("pass either a model or --all, not both")
	}
	if !all && model == "" {
		return errors.New("specify a model to unload or pass --all")
	}

	resp, err := client.ListModels(cmd.Context())
	if err != nil {
		return err
	}

	found := false
	var toUnload []string
	for _, m := range resp.Models {
		if !all && m.Key != model {
			continue
		}
		found = true
		for _, inst := range m.LoadedInstances {
			toUnload = append(toUnload, inst.ID)
		}
	}

	if len(toUnload) == 0 {
		if all {
			fmt.Fprintln(cmd.OutOrStdout(), "No models currently loaded.")
			return nil
		}
		if !found {
			return &lmstudio.ErrModelNotFound{Model: model}
		}
		return &lmstudio.ErrModelNotLoaded{Model: model}
	}

	for _, id := range toUnload {
		if err := client.UnloadModel(cmd.Context(), id); err != nil {
			var notFound *lmstudio.ErrInstanceNotFound
			if errors.As(err, &notFound) {
				fmt.Fprintf(cmd.OutOrStdout(), "Instance %s was already unloaded\n", id)
				continue
			}
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Unloaded instance %s\n", id)
	}
	return nil
}

func init() {
	unloadCmd.Flags().BoolVar(&unloadFlagAll, "all", false, "unload every currently loaded model")
	rootCmd.AddCommand(unloadCmd)
}
