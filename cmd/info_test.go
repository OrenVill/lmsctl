package cmd

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"lmsctl/internal/lmstudio"
	"lmsctl/internal/lmstudio/lmstudiotest"
)

func TestRunInfo_NotFoundReturnsErrModelNotFound(t *testing.T) {
	fake := &lmstudiotest.Fake{ModelsResponse: &lmstudio.ModelsResponse{Models: []lmstudio.Model{
		{Key: "other/model"},
	}}}
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})

	err := runInfo(cmd, fake, "missing/model", false)
	var notFound *lmstudio.ErrModelNotFound
	if !errors.As(err, &notFound) {
		t.Fatalf("err = %v, want *lmstudio.ErrModelNotFound", err)
	}
}

func TestRunInfo_ShowsStaticFieldsWhenNotLoaded(t *testing.T) {
	arch := "llama"
	format := "gguf"
	fake := &lmstudiotest.Fake{ModelsResponse: &lmstudio.ModelsResponse{Models: []lmstudio.Model{
		{
			Key:              "openai/gpt-oss-20b",
			Publisher:        "openai",
			DisplayName:      "GPT OSS 20B",
			Architecture:     &arch,
			Format:           &format,
			Quantization:     &lmstudio.Quantization{Name: "MXFP4", BitsPerWeight: 4.25},
			SizeBytes:        12137712025,
			MaxContextLength: 131072,
		},
	}}}
	cmd := &cobra.Command{}
	out := &bytes.Buffer{}
	cmd.SetOut(out)

	if err := runInfo(cmd, fake, "openai/gpt-oss-20b", false); err != nil {
		t.Fatalf("runInfo: %v", err)
	}

	got := out.String()
	for _, want := range []string{"openai/gpt-oss-20b", "openai", "GPT OSS 20B", "llama", "gguf", "MXFP4", "131072", "Not currently loaded."} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q: %q", want, got)
		}
	}
}

func TestRunInfo_ShowsLoadedInstanceConfig(t *testing.T) {
	fake := &lmstudiotest.Fake{ModelsResponse: &lmstudio.ModelsResponse{Models: []lmstudio.Model{
		{
			Key: "openai/gpt-oss-20b",
			LoadedInstances: []lmstudio.LoadedInstance{
				{ID: "inst-1", Config: lmstudio.InstanceConfig{ContextLength: 16384, FlashAttention: true, OffloadKVCacheToGPU: false, Parallel: 1, EvalBatchSize: 512}},
			},
		},
	}}}
	cmd := &cobra.Command{}
	out := &bytes.Buffer{}
	cmd.SetOut(out)

	if err := runInfo(cmd, fake, "openai/gpt-oss-20b", false); err != nil {
		t.Fatalf("runInfo: %v", err)
	}

	got := out.String()
	for _, want := range []string{"inst-1", "16384", "true", "false", "512"} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q: %q", want, got)
		}
	}
	if strings.Contains(got, "Not currently loaded") {
		t.Errorf("output = %q, should not say not-loaded when an instance exists", got)
	}
}

func TestRunInfo_JSONOutput(t *testing.T) {
	fake := &lmstudiotest.Fake{ModelsResponse: &lmstudio.ModelsResponse{Models: []lmstudio.Model{
		{Key: "openai/gpt-oss-20b", Publisher: "openai"},
	}}}
	cmd := &cobra.Command{}
	out := &bytes.Buffer{}
	cmd.SetOut(out)

	if err := runInfo(cmd, fake, "openai/gpt-oss-20b", true); err != nil {
		t.Fatalf("runInfo: %v", err)
	}
	if !strings.Contains(out.String(), `"key": "openai/gpt-oss-20b"`) {
		t.Errorf("output = %q, want JSON containing the model key", out.String())
	}
}

func TestRunInfo_PropagatesClientError(t *testing.T) {
	fake := &lmstudiotest.Fake{ListModelsErr: &lmstudio.ErrUnreachable{Host: "http://host:1234"}}
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})

	if err := runInfo(cmd, fake, "any/model", false); err == nil {
		t.Fatal("expected error, got nil")
	}
}
