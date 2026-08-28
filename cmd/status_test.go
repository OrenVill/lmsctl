package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"lmsctl/internal/lmstudio"
	"lmsctl/internal/lmstudio/lmstudiotest"
)

func TestRunStatus_NoModelsLoaded(t *testing.T) {
	fake := &lmstudiotest.Fake{
		ModelsResponse: &lmstudio.ModelsResponse{Models: []lmstudio.Model{
			{Key: "openai/gpt-oss-20b"},
		}},
	}
	cmd := &cobra.Command{}
	out := &bytes.Buffer{}
	cmd.SetOut(out)

	if err := runStatus(cmd, fake, "192.168.1.50:1234"); err != nil {
		t.Fatalf("runStatus: %v", err)
	}
	if !strings.Contains(out.String(), "No models currently loaded") {
		t.Errorf("output = %q, want it to say nothing is loaded", out.String())
	}
}

func TestRunStatus_ReportsLoadedModel(t *testing.T) {
	fake := &lmstudiotest.Fake{
		ModelsResponse: &lmstudio.ModelsResponse{Models: []lmstudio.Model{
			{Key: "openai/gpt-oss-20b", LoadedInstances: []lmstudio.LoadedInstance{{ID: "inst-1"}}},
		}},
	}
	cmd := &cobra.Command{}
	out := &bytes.Buffer{}
	cmd.SetOut(out)

	if err := runStatus(cmd, fake, "192.168.1.50:1234"); err != nil {
		t.Fatalf("runStatus: %v", err)
	}
	if !strings.Contains(out.String(), "openai/gpt-oss-20b (inst-1)") {
		t.Errorf("output = %q, want it to mention the loaded model", out.String())
	}
}

func TestRunStatus_PropagatesClientError(t *testing.T) {
	fake := &lmstudiotest.Fake{ListModelsErr: &lmstudio.ErrUnreachable{Host: "http://host:1234"}}
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})

	if err := runStatus(cmd, fake, "host:1234"); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestRunStatus_JSONOutput(t *testing.T) {
	flagJSON = true
	defer func() { flagJSON = false }()

	fake := &lmstudiotest.Fake{
		ModelsResponse: &lmstudio.ModelsResponse{Models: []lmstudio.Model{
			{Key: "openai/gpt-oss-20b", LoadedInstances: []lmstudio.LoadedInstance{{ID: "inst-1"}}},
		}},
	}
	cmd := &cobra.Command{}
	out := &bytes.Buffer{}
	cmd.SetOut(out)

	if err := runStatus(cmd, fake, "host:1234"); err != nil {
		t.Fatalf("runStatus: %v", err)
	}
	if !strings.Contains(out.String(), `"openai/gpt-oss-20b (inst-1)"`) {
		t.Errorf("output = %q, want JSON containing the loaded model", out.String())
	}
}
