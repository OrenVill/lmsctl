package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

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

func TestBuildLoadRequest_OnlyIncludesExplicitlySetFlags(t *testing.T) {
	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
	fs.Int("context-length", 0, "")
	fs.Bool("flash-attention", false, "")
	fs.Bool("offload-kv-cache-to-gpu", false, "")
	if err := fs.Parse([]string{"--context-length", "8192", "--flash-attention=false"}); err != nil {
		t.Fatalf("Parse: %v", err)
	}

	req := buildLoadRequest(fs, "openai/gpt-oss-20b")

	if req.Model != "openai/gpt-oss-20b" {
		t.Errorf("Model = %q, want %q", req.Model, "openai/gpt-oss-20b")
	}
	if req.ContextLength == nil || *req.ContextLength != 8192 {
		t.Errorf("ContextLength = %v, want 8192", req.ContextLength)
	}
	if req.FlashAttention == nil || *req.FlashAttention != false {
		t.Errorf("FlashAttention = %v, want a set false (explicitly passed)", req.FlashAttention)
	}
	if req.OffloadKVCacheToGPU != nil {
		t.Errorf("OffloadKVCacheToGPU = %v, want nil (flag not passed)", req.OffloadKVCacheToGPU)
	}
}

func TestBuildLoadRequest_NoFlagsSetOnlyIncludesModel(t *testing.T) {
	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
	fs.Int("context-length", 0, "")
	fs.Bool("flash-attention", false, "")
	fs.Bool("offload-kv-cache-to-gpu", false, "")

	req := buildLoadRequest(fs, "some/model")

	if req.ContextLength != nil || req.FlashAttention != nil || req.OffloadKVCacheToGPU != nil {
		t.Errorf("expected no optional fields set when no flags were passed, got %+v", req)
	}
}
