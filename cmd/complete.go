package cmd

import (
	"strings"

	"github.com/spf13/cobra"

	"lmsctl/internal/lmstudio"
)

// completeModelKeys returns the model keys matching toComplete as shell
// completion suggestions, fetched live via client. When onlyLoaded is true,
// only keys with at least one loaded instance are suggested (used by
// unload, which rejects a model that isn't currently loaded). It returns no
// suggestions once a model has already been given (len(args) != 0, since
// load/unload/info all take exactly one model argument) or if the lookup
// itself fails -- a failed background request during tab completion should
// stay silent rather than surface an error to the shell.
func completeModelKeys(cmd *cobra.Command, client lmstudio.Client, args []string, toComplete string, onlyLoaded bool) ([]string, cobra.ShellCompDirective) {
	if len(args) != 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	resp, err := client.ListModels(cmd.Context())
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	var keys []string
	for _, m := range resp.Models {
		if onlyLoaded && len(m.LoadedInstances) == 0 {
			continue
		}
		if strings.HasPrefix(m.Key, toComplete) {
			keys = append(keys, m.Key)
		}
	}
	return keys, cobra.ShellCompDirectiveNoFileComp
}

// completeModelKeysFunc builds a cobra.Command.ValidArgsFunction that
// resolves a real client via newClient() and delegates to
// completeModelKeys. A client-resolution failure (e.g. no host configured)
// yields no suggestions, same as a failed ListModels call.
func completeModelKeysFunc(onlyLoaded bool) func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		client, _, err := newClient()
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return completeModelKeys(cmd, client, args, toComplete, onlyLoaded)
	}
}
