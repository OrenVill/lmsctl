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
	if llm.Type != "llm" {
		t.Errorf("Type = %q, want %q", llm.Type, "llm")
	}
	if llm.Publisher != "openai" {
		t.Errorf("Publisher = %q, want %q", llm.Publisher, "openai")
	}
	if llm.DisplayName != "GPT OSS 20B" {
		t.Errorf("DisplayName = %q, want %q", llm.DisplayName, "GPT OSS 20B")
	}
	if llm.SizeBytes != 12884901888 {
		t.Errorf("SizeBytes = %d, want %d", llm.SizeBytes, int64(12884901888))
	}
	if llm.MaxContextLength != 131072 {
		t.Errorf("MaxContextLength = %d, want %d", llm.MaxContextLength, 131072)
	}
	if llm.Format == nil || *llm.Format != "gguf" {
		t.Errorf("Format = %v, want %q", llm.Format, "gguf")
	}
	if len(llm.LoadedInstances) != 1 || llm.LoadedInstances[0].ID != "inst-1" {
		t.Errorf("unexpected loaded instances: %+v", llm.LoadedInstances)
	}
	if !llm.LoadedInstances[0].Config.FlashAttention {
		t.Errorf("expected FlashAttention = true")
	}
	if llm.LoadedInstances[0].Config.ContextLength != 8192 {
		t.Errorf("Config.ContextLength = %d, want %d", llm.LoadedInstances[0].Config.ContextLength, 8192)
	}
	if llm.LoadedInstances[0].Config.OffloadKVCacheToGPU {
		t.Errorf("expected Config.OffloadKVCacheToGPU = false")
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

func TestLoadModelRequest_IncludesSetOptionalFields(t *testing.T) {
	contextLength := 8192
	flashAttention := true
	offloadKVCacheToGPU := false
	req := LoadModelRequest{
		Model:               "openai/gpt-oss-20b",
		ContextLength:       &contextLength,
		FlashAttention:      &flashAttention,
		OffloadKVCacheToGPU: &offloadKVCacheToGPU,
	}
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	want := `{"model":"openai/gpt-oss-20b","context_length":8192,"flash_attention":true,"offload_kv_cache_to_gpu":false}`
	if string(data) != want {
		t.Errorf("got %s, want %s", data, want)
	}
}

func TestLoadModelResponse_Unmarshals(t *testing.T) {
	data := []byte(`{"type": "llm", "instance_id": "inst-1", "load_time_seconds": 2.5, "status": "loaded"}`)

	var resp LoadModelResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if resp.Type != "llm" {
		t.Errorf("Type = %q, want %q", resp.Type, "llm")
	}
	if resp.InstanceID != "inst-1" {
		t.Errorf("InstanceID = %q, want %q", resp.InstanceID, "inst-1")
	}
	if resp.LoadTimeSeconds != 2.5 {
		t.Errorf("LoadTimeSeconds = %v, want %v", resp.LoadTimeSeconds, 2.5)
	}
	if resp.Status != "loaded" {
		t.Errorf("Status = %q, want %q", resp.Status, "loaded")
	}
}

func TestUnloadModelRequest_Marshals(t *testing.T) {
	req := unloadModelRequest{InstanceID: "inst-1"}
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	want := `{"instance_id":"inst-1"}`
	if string(data) != want {
		t.Errorf("got %s, want %s", data, want)
	}
}
