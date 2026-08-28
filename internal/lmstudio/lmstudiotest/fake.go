// Package lmstudiotest provides an in-memory lmstudio.Client double for
// testing cobra commands without a real LM Studio server.
package lmstudiotest

import (
	"context"

	"lmsctl/internal/lmstudio"
)

type Fake struct {
	ModelsResponse *lmstudio.ModelsResponse
	ListModelsErr  error

	LoadModelResponse *lmstudio.LoadModelResponse
	LoadModelErr      error
	LoadModelRequests []lmstudio.LoadModelRequest

	UnloadModelErr    error
	UnloadInstanceIDs []string
}

var _ lmstudio.Client = (*Fake)(nil)

func (f *Fake) ListModels(ctx context.Context) (*lmstudio.ModelsResponse, error) {
	if f.ListModelsErr != nil {
		return nil, f.ListModelsErr
	}
	return f.ModelsResponse, nil
}

func (f *Fake) LoadModel(ctx context.Context, req lmstudio.LoadModelRequest) (*lmstudio.LoadModelResponse, error) {
	f.LoadModelRequests = append(f.LoadModelRequests, req)
	if f.LoadModelErr != nil {
		return nil, f.LoadModelErr
	}
	return f.LoadModelResponse, nil
}

func (f *Fake) UnloadModel(ctx context.Context, instanceID string) error {
	f.UnloadInstanceIDs = append(f.UnloadInstanceIDs, instanceID)
	return f.UnloadModelErr
}
