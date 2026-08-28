package lmstudio

import (
	"encoding/json"
	"testing"
)

func TestModelsResponse_UnmarshalsFullShape(t *testing.T) {
	data := []byte(`{
		"models": [
			{
				"type": "llm",
				"publisher": "openai",
				"key": "openai/gpt-oss-20b",
				"display_name": "GPT OSS 20B",
				"architecture": "llama",
				"quantization": {"name": "Q4_K_M", "bits_per_weight": 4.5},
				"size_bytes": 12884901888,
				"params_string": "20B",
				"max_context_length": 131072,
				"format": "gguf",
				"loaded_instances": [
					{"id": "inst-1", "config": {"context_length": 8192, "flash_attention": true, "offload_kv_cache_to_gpu": false}}
				]
			},
			{
				"type": "embedding",
				"key": "nomic/embed-text",
				"display_name": "Nomic Embed Text",
				"architecture": null,
				"quantization": null,
				"size_bytes": 500000,
				"params_string": null,
				"max_context_length": 2048,
				"format": "gguf",
				"loaded_instances": []
			}
		]
	}`)

	var resp ModelsResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if len(resp.Models) != 2 {
		t.Fatalf("len(Models) = %d, want 2", len(resp.Models))
	}

	llm := resp.Models[0]
	if llm.Key != "openai/gpt-oss-20b" || llm.Quantization == nil || llm.Quantization.Name != "Q4_K_M" {
		t.Errorf("unexpected llm model: %+v", llm)
	}
	if len(llm.LoadedInstances) != 1 || llm.LoadedInstances[0].ID != "inst-1" {
		t.Errorf("unexpected loaded instances: %+v", llm.LoadedInstances)
	}
	if !llm.LoadedInstances[0].Config.FlashAttention {
		t.Errorf("expected FlashAttention = true")
	}

	embed := resp.Models[1]
	if embed.Architecture != nil || embed.Quantization != nil || embed.ParamsString != nil {
		t.Errorf("expected nullable fields to be nil for embedding model: %+v", embed)
	}
	if len(embed.LoadedInstances) != 0 {
		t.Errorf("expected no loaded instances for embedding model: %+v", embed.LoadedInstances)
	}
}

func TestLoadModelRequest_OmitsUnsetOptionalFields(t *testing.T) {
	req := LoadModelRequest{Model: "openai/gpt-oss-20b"}
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if string(data) != `{"model":"openai/gpt-oss-20b"}` {
		t.Errorf("got %s, want only the model field", data)
	}
}
