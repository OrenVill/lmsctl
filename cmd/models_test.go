package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"lmsctl/internal/lmstudio"
	"lmsctl/internal/lmstudio/lmstudiotest"
)

func TestRunModels_TableOutputShowsStateAndSize(t *testing.T) {
	fake := &lmstudiotest.Fake{
		ModelsResponse: &lmstudio.ModelsResponse{Models: []lmstudio.Model{
			{
				Key:             "openai/gpt-oss-20b",
				SizeBytes:       12884901888, // 12 GiB
				Quantization:    &lmstudio.Quantization{Name: "Q4_K_M"},
				LoadedInstances: []lmstudio.LoadedInstance{{ID: "inst-1"}},
			},
			{
				Key:       "embed/model",
				SizeBytes: 500,
			},
		}},
	}
	cmd := &cobra.Command{}
	out := &bytes.Buffer{}
	cmd.SetOut(out)

	if err := runModels(cmd, fake, false); err != nil {
		t.Fatalf("runModels: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "openai/gpt-oss-20b") || !strings.Contains(got, "12.0GiB") || !strings.Contains(got, "loaded") {
		t.Errorf("output missing expected loaded model row: %q", got)
	}
	if !strings.Contains(got, "embed/model") || !strings.Contains(got, "not-loaded") {
		t.Errorf("output missing expected not-loaded model row: %q", got)
	}
}

func TestFormatBytes(t *testing.T) {
	cases := map[int64]string{
		500:         "500B",
		1024:        "1.0KiB",
		12884901888: "12.0GiB",
	}
	for in, want := range cases {
		if got := formatBytes(in); got != want {
			t.Errorf("formatBytes(%d) = %q, want %q", in, got, want)
		}
	}
}

func TestRunModels_JSONOutput(t *testing.T) {
	fake := &lmstudiotest.Fake{
		ModelsResponse: &lmstudio.ModelsResponse{Models: []lmstudio.Model{
			{Key: "openai/gpt-oss-20b"},
		}},
	}
	cmd := &cobra.Command{}
	out := &bytes.Buffer{}
	cmd.SetOut(out)

	if err := runModels(cmd, fake, true); err != nil {
		t.Fatalf("runModels: %v", err)
	}
	if !strings.Contains(out.String(), `"key": "openai/gpt-oss-20b"`) {
		t.Errorf("output = %q, want JSON containing the model key", out.String())
	}
}

func TestRunModels_JSONOutputWithNoModelsIsEmptyArrayNotNull(t *testing.T) {
	fake := &lmstudiotest.Fake{ModelsResponse: &lmstudio.ModelsResponse{}}
	cmd := &cobra.Command{}
	out := &bytes.Buffer{}
	cmd.SetOut(out)

	if err := runModels(cmd, fake, true); err != nil {
		t.Fatalf("runModels: %v", err)
	}
	if strings.Contains(out.String(), "null") {
		t.Errorf("output = %q, want [] not null when there are no models", out.String())
	}
}

func TestRunModels_PropagatesClientError(t *testing.T) {
	fake := &lmstudiotest.Fake{ListModelsErr: &lmstudio.ErrUnreachable{Host: "http://host:1234"}}
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})

	if err := runModels(cmd, fake, false); err == nil {
		t.Fatal("expected error, got nil")
	}
}
