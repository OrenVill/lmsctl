package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"lmsctl/internal/lmstudio"
	"lmsctl/internal/lmstudio/lmstudiotest"
)

func TestRunLoad_SendsModelAndReportsInstanceID(t *testing.T) {
	fake := &lmstudiotest.Fake{
		LoadModelResponse: &lmstudio.LoadModelResponse{InstanceID: "inst-1", LoadTimeSeconds: 2.3},
	}
	cmd := &cobra.Command{}
	out := &bytes.Buffer{}
	cmd.SetOut(out)

	if err := runLoad(cmd, fake, "openai/gpt-oss-20b", false); err != nil {
		t.Fatalf("runLoad: %v", err)
	}
	if len(fake.LoadModelRequests) != 1 || fake.LoadModelRequests[0].Model != "openai/gpt-oss-20b" {
		t.Fatalf("unexpected requests: %+v", fake.LoadModelRequests)
	}
	if !strings.Contains(out.String(), "inst-1") {
		t.Errorf("output = %q, want it to mention the instance id", out.String())
	}
}

func TestRunLoad_PropagatesModelNotFound(t *testing.T) {
	fake := &lmstudiotest.Fake{LoadModelErr: &lmstudio.ErrModelNotFound{Model: "nope"}}
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})

	if err := runLoad(cmd, fake, "nope", false); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestRunLoad_JSONOutput(t *testing.T) {
	fake := &lmstudiotest.Fake{
		LoadModelResponse: &lmstudio.LoadModelResponse{InstanceID: "inst-1"},
	}
	cmd := &cobra.Command{}
	out := &bytes.Buffer{}
	cmd.SetOut(out)

	if err := runLoad(cmd, fake, "openai/gpt-oss-20b", true); err != nil {
		t.Fatalf("runLoad: %v", err)
	}
	if !strings.Contains(out.String(), `"instance_id": "inst-1"`) {
		t.Errorf("output = %q, want JSON containing the instance id", out.String())
	}
}
