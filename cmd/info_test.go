package cmd

import (
	"bytes"
	"errors"
	"regexp"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"lmsctl/internal/color"
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

func TestRunInfo_ColumnsStayAlignedWithColorEnabledAndMultipleInstances(t *testing.T) {
	old := palette
	palette = color.Palette{Enabled: true}
	defer func() { palette = old }()

	fake := &lmstudiotest.Fake{ModelsResponse: &lmstudio.ModelsResponse{Models: []lmstudio.Model{
		{
			Key:         "openai/gpt-oss-20b",
			Publisher:   "openai",
			DisplayName: "GPT OSS 20B",
			SizeBytes:   12137712025,
			LoadedInstances: []lmstudio.LoadedInstance{
				{ID: "inst-1", Config: lmstudio.InstanceConfig{ContextLength: 16384, FlashAttention: true, OffloadKVCacheToGPU: false, Parallel: 1, EvalBatchSize: 512, NumExperts: 32}},
				{ID: "inst-2", Config: lmstudio.InstanceConfig{ContextLength: 8192, FlashAttention: false, OffloadKVCacheToGPU: true, Parallel: 2, EvalBatchSize: 256}},
			},
		},
	}}}
	cmd := &cobra.Command{}
	out := &bytes.Buffer{}
	cmd.SetOut(out)

	if err := runInfo(cmd, fake, "openai/gpt-oss-20b", false); err != nil {
		t.Fatalf("runInfo: %v", err)
	}

	raw := out.String()
	if !strings.Contains(raw, "\x1b[") {
		t.Fatalf("output has no ANSI escape codes; palette.Enabled wasn't honored, so this test can't prove anything: %q", raw)
	}

	ansi := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	plain := ansi.ReplaceAllString(raw, "")
	blocks := strings.Split(strings.TrimRight(plain, "\n"), "\n\n")
	if len(blocks) != 4 {
		t.Fatalf("got %d blank-line-separated blocks, want 4 (static fields, 'Loaded instances:', instance1, instance2): %q", len(blocks), plain)
	}
	staticBlock, header, instance1Block, instance2Block := blocks[0], blocks[1], blocks[2], blocks[3]

	if header != "Loaded instances:" {
		t.Errorf("header = %q, want %q", header, "Loaded instances:")
	}

	// checkAligned fails the test if any "Label:" line in block doesn't
	// have its value starting at the same column as the block's first
	// line -- this is what would catch printFields padding the STYLED
	// label instead of the plain one (which collapses all padding to
	// zero once color is enabled, since len(styled) > width).
	checkAligned := func(t *testing.T, block string) {
		t.Helper()
		lines := strings.Split(block, "\n")
		var starts []int
		for _, l := range lines {
			idx := strings.Index(l, ":")
			if idx == -1 {
				t.Fatalf("line has no label colon: %q", l)
			}
			rest := l[idx+1:]
			trimmed := strings.TrimLeft(rest, " ")
			starts = append(starts, len(l)-len(trimmed))
		}
		for i := 1; i < len(starts); i++ {
			if starts[i] != starts[0] {
				t.Errorf("column misaligned: line 0 value starts at %d, line %d starts at %d\nblock: %q", starts[0], i, starts[i], block)
			}
		}
	}

	checkAligned(t, staticBlock)
	checkAligned(t, instance1Block)
	checkAligned(t, instance2Block)

	if !strings.Contains(instance1Block, "Num experts:") {
		t.Errorf("instance1 (NumExperts=32) block missing 'Num experts:' line: %q", instance1Block)
	}
	if strings.Contains(instance2Block, "Num experts:") {
		t.Errorf("instance2 (NumExperts=0) block should not show 'Num experts:' line: %q", instance2Block)
	}
}
