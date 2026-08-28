package lmstudio

// ModelsResponse is the body of GET /api/v1/models.
type ModelsResponse struct {
	Models []Model `json:"models"`
}

// Model describes one downloaded model and its currently loaded instances.
type Model struct {
	Type             string           `json:"type"`
	Publisher        string           `json:"publisher"`
	Key              string           `json:"key"`
	DisplayName      string           `json:"display_name"`
	Architecture     *string          `json:"architecture"`
	Quantization     *Quantization    `json:"quantization"`
	SizeBytes        int64            `json:"size_bytes"`
	ParamsString     *string          `json:"params_string"`
	LoadedInstances  []LoadedInstance `json:"loaded_instances"`
	MaxContextLength int              `json:"max_context_length"`
	Format           *string          `json:"format"`
}

type Quantization struct {
	Name          string  `json:"name"`
	BitsPerWeight float64 `json:"bits_per_weight"`
}

type LoadedInstance struct {
	ID     string         `json:"id"`
	Config InstanceConfig `json:"config"`
}

type InstanceConfig struct {
	ContextLength       int  `json:"context_length"`
	EvalBatchSize       int  `json:"eval_batch_size"`
	Parallel            int  `json:"parallel"`
	FlashAttention      bool `json:"flash_attention"`
	NumExperts          int  `json:"num_experts"`
	OffloadKVCacheToGPU bool `json:"offload_kv_cache_to_gpu"`
}

// LoadModelRequest is the body of POST /api/v1/models/load. Pointer fields
// are omitted from the JSON when nil, letting LM Studio apply its own
// defaults for anything the caller didn't set explicitly.
type LoadModelRequest struct {
	Model               string `json:"model"`
	ContextLength       *int   `json:"context_length,omitempty"`
	FlashAttention      *bool  `json:"flash_attention,omitempty"`
	OffloadKVCacheToGPU *bool  `json:"offload_kv_cache_to_gpu,omitempty"`
}

// LoadModelResponse is the body returned by POST /api/v1/models/load.
type LoadModelResponse struct {
	Type            string  `json:"type"`
	InstanceID      string  `json:"instance_id"`
	LoadTimeSeconds float64 `json:"load_time_seconds"`
	Status          string  `json:"status"`
}

// unloadModelRequest is the body of POST /api/v1/models/unload.
type unloadModelRequest struct {
	InstanceID string `json:"instance_id"`
}
