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

func TestRunUnload_DownloadedButIdleModelReturnsErrModelNotLoaded(t *testing.T) {
	fake := &lmstudiotest.Fake{ModelsResponse: &lmstudio.ModelsResponse{Models: []lmstudio.Model{
		{Key: "idle/model"}, // downloaded, no loaded instances
	}}}
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})

	err := runUnload(cmd, fake, "idle/model", false)
	var notLoaded *lmstudio.ErrModelNotLoaded
	if !errors.As(err, &notLoaded) {
		t.Fatalf("err = %v, want *lmstudio.ErrModelNotLoaded", err)
	}
	if !strings.Contains(err.Error(), "idle/model") {
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

func TestRunUnload_RejectsModelWithAll(t *testing.T) {
	fake := &lmstudiotest.Fake{ModelsResponse: modelsWithLoaded()}
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})

	if err := runUnload(cmd, fake, "openai/gpt-oss-20b", true); err == nil {
		t.Fatal("expected error, got nil")
	}
	if len(fake.UnloadInstanceIDs) != 0 {
		t.Errorf("unloaded instance ids = %v, want none (should reject before calling the client)", fake.UnloadInstanceIDs)
	}
}

func TestRunUnload_SkipsAlreadyUnloadedInstanceAndContinues(t *testing.T) {
	fake := &lmstudiotest.Fake{
		ModelsResponse: modelsWithLoaded(),
		UnloadModelErr: &lmstudio.ErrInstanceNotFound{InstanceID: "inst-1"},
	}
	cmd := &cobra.Command{}
	out := &bytes.Buffer{}
	cmd.SetOut(out)

	if err := runUnload(cmd, fake, "", true); err != nil {
		t.Fatalf("runUnload: %v", err)
	}
	if len(fake.UnloadInstanceIDs) != 2 {
		t.Errorf("unload attempts = %v, want 2 (both instances attempted despite the error)", fake.UnloadInstanceIDs)
	}
	if !strings.Contains(out.String(), "already unloaded") {
		t.Errorf("output = %q, want it to say the instance was already unloaded", out.String())
	}
}

func TestRunUnload_PropagatesGenericUnloadError(t *testing.T) {
	fake := &lmstudiotest.Fake{
		ModelsResponse: modelsWithLoaded(),
		UnloadModelErr: &lmstudio.ErrUnreachable{Host: "http://host:1234"},
	}
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})

	if err := runUnload(cmd, fake, "openai/gpt-oss-20b", false); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestRunUnload_PropagatesListModelsError(t *testing.T) {
	fake := &lmstudiotest.Fake{ListModelsErr: &lmstudio.ErrUnreachable{Host: "http://host:1234"}}
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})

	if err := runUnload(cmd, fake, "openai/gpt-oss-20b", false); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestRunUnload_UnknownModelReturnsErrModelNotFound(t *testing.T) {
	fake := &lmstudiotest.Fake{ModelsResponse: modelsWithLoaded()}
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})

	err := runUnload(cmd, fake, "totally/unknown", false)
	var notFound *lmstudio.ErrModelNotFound
	if !errors.As(err, &notFound) {
		t.Fatalf("err = %v, want *lmstudio.ErrModelNotFound", err)
	}
}

func TestRunUnload_UnloadsAllInstancesOfMatchingModel(t *testing.T) {
	fake := &lmstudiotest.Fake{ModelsResponse: &lmstudio.ModelsResponse{Models: []lmstudio.Model{
		{Key: "multi/instance", LoadedInstances: []lmstudio.LoadedInstance{{ID: "inst-a"}, {ID: "inst-b"}}},
	}}}
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})

	if err := runUnload(cmd, fake, "multi/instance", false); err != nil {
		t.Fatalf("runUnload: %v", err)
	}
	if len(fake.UnloadInstanceIDs) != 2 {
		t.Errorf("unloaded instance ids = %v, want 2 entries", fake.UnloadInstanceIDs)
	}
}
