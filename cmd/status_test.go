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

	if err := runStatus(cmd, fake, "192.168.1.50:1234", false); err != nil {
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

	if err := runStatus(cmd, fake, "192.168.1.50:1234", false); err != nil {
		t.Fatalf("runStatus: %v", err)
	}
	if !strings.Contains(out.String(), "openai/gpt-oss-20b") || !strings.Contains(out.String(), "inst-1") {
		t.Errorf("output = %q, want it to mention the loaded model", out.String())
	}
}

func TestRunStatus_PropagatesClientError(t *testing.T) {
	fake := &lmstudiotest.Fake{ListModelsErr: &lmstudio.ErrUnreachable{Host: "http://host:1234"}}
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})

	if err := runStatus(cmd, fake, "host:1234", false); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestRunStatus_JSONOutput(t *testing.T) {
	fake := &lmstudiotest.Fake{
		ModelsResponse: &lmstudio.ModelsResponse{Models: []lmstudio.Model{
			{Key: "openai/gpt-oss-20b", LoadedInstances: []lmstudio.LoadedInstance{{ID: "inst-1"}}},
		}},
	}
	cmd := &cobra.Command{}
	out := &bytes.Buffer{}
	cmd.SetOut(out)

	if err := runStatus(cmd, fake, "host:1234", true); err != nil {
		t.Fatalf("runStatus: %v", err)
	}
	if !strings.Contains(out.String(), `"key": "openai/gpt-oss-20b"`) || !strings.Contains(out.String(), `"instance_id": "inst-1"`) {
		t.Errorf("output = %q, want JSON containing the loaded model's key and instance_id", out.String())
	}
}

func TestRunStatus_JSONOutputWithNothingLoadedOmitsNull(t *testing.T) {
	fake := &lmstudiotest.Fake{
		ModelsResponse: &lmstudio.ModelsResponse{Models: []lmstudio.Model{
			{Key: "openai/gpt-oss-20b"},
		}},
	}
	cmd := &cobra.Command{}
	out := &bytes.Buffer{}
	cmd.SetOut(out)

	if err := runStatus(cmd, fake, "host:1234", true); err != nil {
		t.Fatalf("runStatus: %v", err)
	}
	if strings.Contains(out.String(), "null") {
		t.Errorf("output = %q, want loaded_models to be [] not null when nothing is loaded", out.String())
	}
}
