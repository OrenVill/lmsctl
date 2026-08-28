package cmd

import (
	"bytes"
	"regexp"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"lmsctl/internal/color"
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
	if !strings.Contains(got, "KEY") || !strings.Contains(got, "SIZE") || !strings.Contains(got, "QUANTIZATION") || !strings.Contains(got, "STATE") {
		t.Errorf("output missing expected header row: %q", got)
	}
	if !strings.Contains(got, "openai/gpt-oss-20b") || !strings.Contains(got, "12.0GiB") || !strings.Contains(got, "loaded") {
		t.Errorf("output missing expected loaded model row: %q", got)
	}
	if !strings.Contains(got, "embed/model") || !strings.Contains(got, "not-loaded") {
		t.Errorf("output missing expected not-loaded model row: %q", got)
	}
}

func TestRunModels_ColumnsStayAlignedWithColorEnabled(t *testing.T) {
	old := palette
	palette = color.Palette{Enabled: true}
	defer func() { palette = old }()

	fake := &lmstudiotest.Fake{
		ModelsResponse: &lmstudio.ModelsResponse{Models: []lmstudio.Model{
			{
				Key:             "openai/gpt-oss-20b", // longer than the "KEY" header
				SizeBytes:       12884901888,          // "12.0GiB", longer than "SIZE" header
				Quantization:    &lmstudio.Quantization{Name: "Q4_K_M"},
				LoadedInstances: []lmstudio.LoadedInstance{{ID: "inst-1"}},
			},
			{Key: "b", SizeBytes: 500}, // shorter than the header on every column
		}},
	}
	cmd := &cobra.Command{}
	out := &bytes.Buffer{}
	cmd.SetOut(out)

	if err := runModels(cmd, fake, false); err != nil {
		t.Fatalf("runModels: %v", err)
	}

	raw := out.String()
	if !strings.Contains(raw, "\x1b[") {
		t.Fatalf("output has no ANSI escape codes; palette.Enabled wasn't honored, so this test can't prove anything: %q", raw)
	}

	ansi := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	lines := strings.Split(strings.TrimRight(raw, "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("got %d lines, want 3 (header + 2 rows): %q", len(lines), raw)
	}
	plainHeader := ansi.ReplaceAllString(lines[0], "")
	plainRow1 := ansi.ReplaceAllString(lines[1], "")
	plainRow2 := ansi.ReplaceAllString(lines[2], "")

	sizeIdx := strings.Index(plainHeader, "SIZE")
	if idx := strings.Index(plainRow1, "12.0GiB"); idx != sizeIdx {
		t.Errorf("SIZE column misaligned: header at %d, row1 (\"12.0GiB\") at %d\nheader: %q\nrow1:   %q", sizeIdx, idx, plainHeader, plainRow1)
	}
	if idx := strings.Index(plainRow2, "500B"); idx != sizeIdx {
		t.Errorf("SIZE column misaligned: header at %d, row2 (\"500B\") at %d\nheader: %q\nrow2:   %q", sizeIdx, idx, plainHeader, plainRow2)
	}

	quantIdx := strings.Index(plainHeader, "QUANTIZATION")
	if idx := strings.Index(plainRow1, "Q4_K_M"); idx != quantIdx {
		t.Errorf("QUANTIZATION column misaligned: header at %d, row1 at %d\nheader: %q\nrow1:   %q", quantIdx, idx, plainHeader, plainRow1)
	}
	if idx := strings.Index(plainRow2, "-"); idx != quantIdx {
		t.Errorf("QUANTIZATION column misaligned: header at %d, row2 (\"-\") at %d\nheader: %q\nrow2:   %q", quantIdx, idx, plainHeader, plainRow2)
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
