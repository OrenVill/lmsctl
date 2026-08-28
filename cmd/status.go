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
		jsonOut, _ := cmd.Flags().GetBool("json")
		return runStatus(cmd, client, eff.Host, jsonOut)
	},
}

// statusJSON is the --json schema for `lmsctl status`.
type statusJSON struct {
	Host   string            `json:"host"`
	Loaded []loadedModelJSON `json:"loaded_models"`
}

type loadedModelJSON struct {
	Key        string `json:"key"`
	InstanceID string `json:"instance_id"`
}

// runStatus reports whether client is reachable and what it currently has
// loaded, as human-readable text or (when jsonOut is true) JSON.
func runStatus(cmd *cobra.Command, client lmstudio.Client, host string, jsonOut bool) error {
	resp, err := client.ListModels(cmd.Context())
	if err != nil {
		return err
	}

	loaded := []loadedModelJSON{}
	for _, m := range resp.Models {
		for _, inst := range m.LoadedInstances {
			loaded = append(loaded, loadedModelJSON{Key: m.Key, InstanceID: inst.ID})
		}
	}

	w := cmd.OutOrStdout()

	if jsonOut {
		return output.JSON(w, statusJSON{Host: host, Loaded: loaded})
	}

	fmt.Fprintf(w, "LM Studio at %s: reachable\n", host)
	if len(loaded) == 0 {
		fmt.Fprintln(w, "No models currently loaded.")
		return nil
	}
	fmt.Fprintln(w, "Loaded models:")
	for _, l := range loaded {
		fmt.Fprintf(w, "  - %s (%s)\n", l.Key, l.InstanceID)
	}
	return nil
}

func init() {
	statusCmd.Flags().Bool("json", false, "output machine-readable JSON")
	rootCmd.AddCommand(statusCmd)
}
