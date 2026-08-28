package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"lmsctl/internal/lmstudio"
	"lmsctl/internal/lmstudio/lmstudiotest"
)

func modelsWithLoaded() *lmstudio.ModelsResponse {
	return &lmstudio.ModelsResponse{Models: []lmstudio.Model{
		{Key: "openai/gpt-oss-20b", LoadedInstances: []lmstudio.LoadedInstance{{ID: "inst-1"}}},
		{Key: "other/model", LoadedInstances: []lmstudio.LoadedInstance{{ID: "inst-2"}}},
	}}
}

func TestRunUnload_UnloadsMatchingModelOnly(t *testing.T) {
	fake := &lmstudiotest.Fake{ModelsResponse: modelsWithLoaded()}
	cmd := &cobra.Command{}
	out := &bytes.Buffer{}
	cmd.SetOut(out)

	if err := runUnload(cmd, fake, "openai/gpt-oss-20b", false); err != nil {
		t.Fatalf("runUnload: %v", err)
	}
	if len(fake.UnloadInstanceIDs) != 1 || fake.UnloadInstanceIDs[0] != "inst-1" {
		t.Errorf("unloaded instance ids = %v, want [inst-1]", fake.UnloadInstanceIDs)
	}
}

func TestRunUnload_AllUnloadsEveryLoadedInstance(t *testing.T) {
	fake := &lmstudiotest.Fake{ModelsResponse: modelsWithLoaded()}
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})

	if err := runUnload(cmd, fake, "", true); err != nil {
		t.Fatalf("runUnload: %v", err)
	}
	if len(fake.UnloadInstanceIDs) != 2 {
		t.Errorf("unloaded instance ids = %v, want 2 entries", fake.UnloadInstanceIDs)
	}
}

func TestRunUnload_NoModelAndNoAllReturnsError(t *testing.T) {
	fake := &lmstudiotest.Fake{}
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})

	if err := runUnload(cmd, fake, "", false); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestRunUnload_ModelNotLoadedReturnsError(t *testing.T) {
	fake := &lmstudiotest.Fake{ModelsResponse: modelsWithLoaded()}
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})

	err := runUnload(cmd, fake, "not/loaded", false)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "not/loaded") {
		t.Errorf("err = %v, want it to mention the model", err)
	}
}

func TestRunUnload_AllWithNothingLoadedPrintsMessageNoError(t *testing.T) {
	fake := &lmstudiotest.Fake{ModelsResponse: &lmstudio.ModelsResponse{}}
	cmd := &cobra.Command{}
	out := &bytes.Buffer{}
	cmd.SetOut(out)

	if err := runUnload(cmd, fake, "", true); err != nil {
		t.Fatalf("runUnload: %v", err)
	}
	if !strings.Contains(out.String(), "No models currently loaded") {
		t.Errorf("output = %q, want it to say nothing is loaded", out.String())
	}
}
