package cmd

import (
	"errors"
	"slices"
	"testing"

	"github.com/spf13/cobra"

	"lmsctl/internal/lmstudio"
	"lmsctl/internal/lmstudio/lmstudiotest"
)

func completeModelsFixture() *lmstudio.ModelsResponse {
	return &lmstudio.ModelsResponse{Models: []lmstudio.Model{
		{Key: "nvidia/nemotron-3-nano", LoadedInstances: []lmstudio.LoadedInstance{{ID: "nvidia/nemotron-3-nano"}}},
		{Key: "nvidia/nemotron-3-nano-4b"},
		{Key: "openai/gpt-oss-20b"},
	}}
}

func TestCompleteModelKeys_FiltersByPrefix(t *testing.T) {
	fake := &lmstudiotest.Fake{ModelsResponse: completeModelsFixture()}
	cmd := &cobra.Command{}

	got, directive := completeModelKeys(cmd, fake, nil, "nvidia/", false)

	want := []string{"nvidia/nemotron-3-nano", "nvidia/nemotron-3-nano-4b"}
	if !slices.Equal(got, want) {
		t.Errorf("got = %v, want %v", got, want)
	}
	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Errorf("directive = %v, want ShellCompDirectiveNoFileComp", directive)
	}
}

func TestCompleteModelKeys_OnlyLoadedFiltersOutModelsWithNoLoadedInstances(t *testing.T) {
	fake := &lmstudiotest.Fake{ModelsResponse: completeModelsFixture()}
	cmd := &cobra.Command{}

	got, _ := completeModelKeys(cmd, fake, nil, "", true)

	want := []string{"nvidia/nemotron-3-nano"}
	if !slices.Equal(got, want) {
		t.Errorf("got = %v, want %v", got, want)
	}
}

func TestCompleteModelKeys_NoSuggestionsOnceModelArgAlreadyGiven(t *testing.T) {
	fake := &lmstudiotest.Fake{ModelsResponse: completeModelsFixture()}
	cmd := &cobra.Command{}

	got, directive := completeModelKeys(cmd, fake, []string{"nvidia/nemotron-3-nano"}, "", false)

	if got != nil {
		t.Errorf("got = %v, want nil", got)
	}
	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Errorf("directive = %v, want ShellCompDirectiveNoFileComp", directive)
	}
}

func TestCompleteModelKeys_NoSuggestionsOnClientError(t *testing.T) {
	fake := &lmstudiotest.Fake{ListModelsErr: errors.New("unreachable")}
	cmd := &cobra.Command{}

	got, directive := completeModelKeys(cmd, fake, nil, "", false)

	if got != nil {
		t.Errorf("got = %v, want nil", got)
	}
	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Errorf("directive = %v, want ShellCompDirectiveNoFileComp", directive)
	}
}
