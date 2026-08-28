# lmsctl Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build `lmsctl`, a Go CLI that manages a remote LM Studio instance over its `/api/v1` REST API — status, model listing, load, and unload — per `docs/superpowers/specs/2026-08-28-lmsctl-design.md`.

**Architecture:** An `internal/lmstudio` package wraps the LM Studio REST API behind a small `Client` interface (real `HTTPClient` + a `lmstudiotest.Fake` test double). `cobra`-based commands in `cmd/` call only that interface. An `internal/config` package resolves host/token from flag > env var > `~/.config/lmsctl/config.yaml`. An `internal/output` package renders human tables or `--json`.

**Tech Stack:** Go 1.26, `github.com/spf13/cobra` (CLI framework), `gopkg.in/yaml.v3` (config file), Go's standard `net/http/httptest` for HTTP-level tests.

---

## Confirmed LM Studio API details (from lmstudio.ai/docs, checked 2026-08-28)

- `GET /api/v1/models` → `{"models": [Model, ...]}`. Each `Model` has `type`, `publisher`, `key`, `display_name`, `architecture` (nullable), `quantization` (nullable object with `name`, `bits_per_weight`), `size_bytes`, `params_string` (nullable), `loaded_instances` (array of `{id, config}`), `max_context_length`, `format` (nullable).
- `POST /api/v1/models/load` body `{"model": "...", "context_length"?, "flash_attention"?, "offload_kv_cache_to_gpu"?, ...}` → `{"type", "instance_id", "load_time_seconds", "status", ...}`.
- `POST /api/v1/models/unload` body `{"instance_id": "..."}` → `{"instance_id": "..."}`.
- Auth: optional `Authorization: Bearer <token>` header.
- No dedicated health endpoint exists — `status` uses `GET /api/v1/models` as the reachability probe, and reports currently-loaded instances from it (there is no server-version field in the response, so `status` does not report a version).

---

### Task 1: Project scaffold

**Files:**
- Create: `go.mod`
- Create: `main.go`
- Create: `cmd/root.go`

- [ ] **Step 1: Initialize the Go module**

Run: `go mod init lmsctl`
Expected: creates `go.mod` with `module lmsctl` and a `go` directive.

- [ ] **Step 2: Add the cobra dependency**

Run: `go get github.com/spf13/cobra@latest`
Expected: `go.mod`/`go.sum` updated, no errors.

- [ ] **Step 3: Write `cmd/root.go`**

```go
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	flagHost  string
	flagToken string
	flagJSON  bool
)

var rootCmd = &cobra.Command{
	Use:   "lmsctl",
	Short: "Manage a remote LM Studio instance from the command line",
}

// Execute runs the root command and exits the process on error.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&flagHost, "host", "", "LM Studio host:port (overrides config file and LMSCTL_HOST)")
	rootCmd.PersistentFlags().StringVar(&flagToken, "token", "", "LM Studio API token (overrides config file and LMSCTL_TOKEN)")
	rootCmd.PersistentFlags().BoolVar(&flagJSON, "json", false, "output machine-readable JSON")
}
```

- [ ] **Step 4: Write `main.go`**

```go
package main

import "lmsctl/cmd"

func main() {
	cmd.Execute()
}
```

- [ ] **Step 5: Build and smoke-test**

Run: `go build -o /tmp/lmsctl . && /tmp/lmsctl --help`
Expected: usage text listing `lmsctl` with the `--host`, `--token`, `--json` flags, exit code 0.

- [ ] **Step 6: Commit**

```bash
git add go.mod go.sum main.go cmd/root.go
git commit -m "Scaffold lmsctl Go module and root command"
```

---

### Task 2: LM Studio API types

**Files:**
- Create: `internal/lmstudio/types.go`
- Test: `internal/lmstudio/types_test.go`

- [ ] **Step 1: Write the failing test**

```go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/lmstudio/... -run 'TestModelsResponse_UnmarshalsFullShape|TestLoadModelRequest_OmitsUnsetOptionalFields' -v`
Expected: FAIL — `types.go` doesn't exist yet, package fails to build (`ModelsResponse`/`LoadModelRequest` undefined).

- [ ] **Step 3: Write `internal/lmstudio/types.go`**

```go
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
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/lmstudio/... -run 'TestModelsResponse_UnmarshalsFullShape|TestLoadModelRequest_OmitsUnsetOptionalFields' -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/lmstudio/types.go internal/lmstudio/types_test.go
git commit -m "Add LM Studio API JSON types"
```

---

### Task 3: Typed errors with actionable messages

**Files:**
- Create: `internal/lmstudio/errors.go`
- Test: `internal/lmstudio/errors_test.go`

- [ ] **Step 1: Write the failing test**

```go
package lmstudio

import (
	"errors"
	"strings"
	"testing"
)

func TestErrUnreachable_MessageMentionsHostAndLocalNetworkSetting(t *testing.T) {
	err := &ErrUnreachable{Host: "http://192.168.1.50:1234", Err: errors.New("connection refused")}
	msg := err.Error()
	if !strings.Contains(msg, "192.168.1.50:1234") {
		t.Errorf("message = %q, want it to mention the host", msg)
	}
	if !strings.Contains(msg, "Serve on Local Network") {
		t.Errorf("message = %q, want it to mention the LM Studio setting", msg)
	}
}

func TestErrUnauthorized_MessageMentionsToken(t *testing.T) {
	msg := (&ErrUnauthorized{}).Error()
	if !strings.Contains(msg, "token") {
		t.Errorf("message = %q, want it to mention the token", msg)
	}
}

func TestErrModelNotFound_MessageMentionsModelKey(t *testing.T) {
	msg := (&ErrModelNotFound{Model: "nonexistent/model"}).Error()
	if !strings.Contains(msg, "nonexistent/model") {
		t.Errorf("message = %q, want it to mention the model key", msg)
	}
}

func TestErrModelNotLoaded_MessageMentionsModelKey(t *testing.T) {
	msg := (&ErrModelNotLoaded{Model: "openai/gpt-oss-20b"}).Error()
	if !strings.Contains(msg, "openai/gpt-oss-20b") {
		t.Errorf("message = %q, want it to mention the model key", msg)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/lmstudio/... -run 'TestErr' -v`
Expected: FAIL — `ErrUnreachable`, `ErrUnauthorized`, `ErrModelNotFound`, `ErrModelNotLoaded` undefined.

- [ ] **Step 3: Write `internal/lmstudio/errors.go`**

```go
package lmstudio

import "fmt"

// ErrUnreachable indicates the LM Studio server could not be reached at all
// (connection refused, DNS failure, timeout).
type ErrUnreachable struct {
	Host string
	Err  error
}

func (e *ErrUnreachable) Error() string {
	return fmt.Sprintf("could not reach LM Studio at %s: %v\n"+
		"check that LM Studio is running on that machine and that "+
		"\"Serve on Local Network\" is enabled in its Developer settings", e.Host, e.Err)
}

func (e *ErrUnreachable) Unwrap() error { return e.Err }

// ErrUnauthorized indicates the server rejected the request due to a
// missing or incorrect API token.
type ErrUnauthorized struct{}

func (e *ErrUnauthorized) Error() string {
	return "LM Studio rejected the request: missing or incorrect API token\n" +
		"set the correct token with --token, LMSCTL_TOKEN, or 'lmsctl config set-host'"
}

// ErrModelNotFound indicates the requested model key does not match any
// downloaded model.
type ErrModelNotFound struct {
	Model string
}

func (e *ErrModelNotFound) Error() string {
	return fmt.Sprintf("no downloaded model matches %q — run 'lmsctl models' to see available models", e.Model)
}

// ErrModelNotLoaded indicates an unload was requested for a model that has
// no loaded instance.
type ErrModelNotLoaded struct {
	Model string
}

func (e *ErrModelNotLoaded) Error() string {
	return fmt.Sprintf("%q is not currently loaded", e.Model)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/lmstudio/... -run 'TestErr' -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/lmstudio/errors.go internal/lmstudio/errors_test.go
git commit -m "Add typed, actionable lmstudio errors"
```

---

### Task 4: HTTP client core + ListModels

**Files:**
- Create: `internal/lmstudio/client.go`
- Test: `internal/lmstudio/client_test.go`

- [ ] **Step 1: Write the failing tests**

```go
package lmstudio

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestListModels_SendsGetAndParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/models" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"models":[{"type":"llm","key":"openai/gpt-oss-20b","display_name":"GPT OSS 20B","loaded_instances":[]}]}`))
	}))
	defer srv.Close()

	c := NewHTTPClient(strings.TrimPrefix(srv.URL, "http://"), "")
	got, err := c.ListModels(context.Background())
	if err != nil {
		t.Fatalf("ListModels: %v", err)
	}
	if len(got.Models) != 1 || got.Models[0].Key != "openai/gpt-oss-20b" {
		t.Errorf("unexpected result: %+v", got)
	}
}

func TestListModels_SendsBearerTokenWhenSet(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Write([]byte(`{"models":[]}`))
	}))
	defer srv.Close()

	c := NewHTTPClient(strings.TrimPrefix(srv.URL, "http://"), "my-token")
	if _, err := c.ListModels(context.Background()); err != nil {
		t.Fatalf("ListModels: %v", err)
	}
	if gotAuth != "Bearer my-token" {
		t.Errorf("Authorization header = %q, want %q", gotAuth, "Bearer my-token")
	}
}

func TestListModels_OmitsAuthHeaderWhenNoToken(t *testing.T) {
	var gotAuth string
	seen := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		seen = true
		w.Write([]byte(`{"models":[]}`))
	}))
	defer srv.Close()

	c := NewHTTPClient(strings.TrimPrefix(srv.URL, "http://"), "")
	if _, err := c.ListModels(context.Background()); err != nil {
		t.Fatalf("ListModels: %v", err)
	}
	if !seen || gotAuth != "" {
		t.Errorf("Authorization header = %q, want empty", gotAuth)
	}
}

func TestListModels_ConnectionRefusedReturnsErrUnreachable(t *testing.T) {
	c := NewHTTPClient("127.0.0.1:1", "") // port 1 is reserved; nothing listens there
	_, err := c.ListModels(context.Background())
	var unreachable *ErrUnreachable
	if !errors.As(err, &unreachable) {
		t.Fatalf("err = %v, want *ErrUnreachable", err)
	}
}

func TestListModels_401ReturnsErrUnauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	c := NewHTTPClient(strings.TrimPrefix(srv.URL, "http://"), "wrong-token")
	_, err := c.ListModels(context.Background())
	var unauthorized *ErrUnauthorized
	if !errors.As(err, &unauthorized) {
		t.Fatalf("err = %v, want *ErrUnauthorized", err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/lmstudio/... -run 'TestListModels' -v`
Expected: FAIL — `NewHTTPClient` undefined.

- [ ] **Step 3: Write `internal/lmstudio/client.go`**

```go
package lmstudio

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client is the interface commands use to talk to LM Studio. Implemented by
// *HTTPClient for real use and by lmstudiotest.Fake in tests.
type Client interface {
	ListModels(ctx context.Context) (*ModelsResponse, error)
	LoadModel(ctx context.Context, req LoadModelRequest) (*LoadModelResponse, error)
	UnloadModel(ctx context.Context, instanceID string) error
}

// HTTPClient implements Client over LM Studio's /api/v1 REST API.
type HTTPClient struct {
	BaseURL    string // e.g. "http://192.168.1.50:1234"
	Token      string
	HTTPClient *http.Client
}

// NewHTTPClient builds an HTTPClient with a sane default timeout.
func NewHTTPClient(host, token string) *HTTPClient {
	return &HTTPClient{
		BaseURL:    "http://" + host,
		Token:      token,
		HTTPClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// httpStatusError wraps a non-2xx response that isn't handled by a more
// specific typed error (401 -> ErrUnauthorized, 404 -> ErrModelNotFound).
type httpStatusError struct {
	StatusCode int
	Body       string
}

func (e *httpStatusError) Error() string {
	return fmt.Sprintf("LM Studio returned HTTP %d: %s", e.StatusCode, e.Body)
}

func (c *HTTPClient) do(ctx context.Context, method, path string, body, out any) error {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encoding request body: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, reqBody)
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return &ErrUnreachable{Host: c.BaseURL, Err: err}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response body: %w", err)
	}

	switch {
	case resp.StatusCode == http.StatusUnauthorized:
		return &ErrUnauthorized{}
	case resp.StatusCode < 200 || resp.StatusCode >= 300:
		return &httpStatusError{StatusCode: resp.StatusCode, Body: string(respBody)}
	}

	if out != nil {
		if err := json.Unmarshal(respBody, out); err != nil {
			return fmt.Errorf("parsing response body: %w", err)
		}
	}
	return nil
}

func (c *HTTPClient) ListModels(ctx context.Context) (*ModelsResponse, error) {
	var out ModelsResponse
	if err := c.do(ctx, http.MethodGet, "/api/v1/models", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *HTTPClient) LoadModel(ctx context.Context, req LoadModelRequest) (*LoadModelResponse, error) {
	var out LoadModelResponse
	if err := c.do(ctx, http.MethodPost, "/api/v1/models/load", req, &out); err != nil {
		var statusErr *httpStatusError
		if errors.As(err, &statusErr) && statusErr.StatusCode == http.StatusNotFound {
			return nil, &ErrModelNotFound{Model: req.Model}
		}
		return nil, err
	}
	return &out, nil
}

func (c *HTTPClient) UnloadModel(ctx context.Context, instanceID string) error {
	// Deliberately does not map 404 to ErrModelNotFound: that error's message
	// ("no downloaded model matches ... run 'lmsctl models'") is written for
	// a model key the user typed, not an instance ID this package resolved
	// itself via ListModels. A 404 here means the instance vanished between
	// that list call and this one; the generic httpStatusError from do() is
	// more honest than misdirected advice.
	return c.do(ctx, http.MethodPost, "/api/v1/models/unload", unloadModelRequest{InstanceID: instanceID}, nil)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/lmstudio/... -run 'TestListModels' -v`
Expected: PASS (all 5 subtests)

- [ ] **Step 5: Commit**

```bash
git add internal/lmstudio/client.go internal/lmstudio/client_test.go
git commit -m "Add HTTPClient core request handling and ListModels"
```

---

### Task 5: LoadModel

**Files:**
- Modify: `internal/lmstudio/client_test.go`

- [ ] **Step 1: Write the failing tests (append to `client_test.go`)**

```go
func TestLoadModel_SendsCorrectBodyAndParsesInstanceID(t *testing.T) {
	var gotBody LoadModelRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/models/load" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.Write([]byte(`{"type":"llm","instance_id":"inst-1","status":"loaded","load_time_seconds":1.5}`))
	}))
	defer srv.Close()

	contextLength := 8192
	c := NewHTTPClient(strings.TrimPrefix(srv.URL, "http://"), "")
	got, err := c.LoadModel(context.Background(), LoadModelRequest{
		Model:         "openai/gpt-oss-20b",
		ContextLength: &contextLength,
	})
	if err != nil {
		t.Fatalf("LoadModel: %v", err)
	}
	if got.InstanceID != "inst-1" {
		t.Errorf("InstanceID = %q, want %q", got.InstanceID, "inst-1")
	}
	if gotBody.Model != "openai/gpt-oss-20b" || gotBody.ContextLength == nil || *gotBody.ContextLength != 8192 {
		t.Errorf("unexpected request body: %+v", gotBody)
	}
}

func TestLoadModel_404ReturnsErrModelNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := NewHTTPClient(strings.TrimPrefix(srv.URL, "http://"), "")
	_, err := c.LoadModel(context.Background(), LoadModelRequest{Model: "nonexistent/model"})
	var notFound *ErrModelNotFound
	if !errors.As(err, &notFound) {
		t.Fatalf("err = %v, want *ErrModelNotFound", err)
	}
	if notFound.Model != "nonexistent/model" {
		t.Errorf("Model = %q, want %q", notFound.Model, "nonexistent/model")
	}
}
```

Add `"encoding/json"` to the existing import block in `client_test.go` if it isn't already imported (it is needed for `json.NewDecoder`).

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/lmstudio/... -run 'TestLoadModel' -v`
Expected: FAIL only if `encoding/json` import is missing (add it); otherwise this should already PASS since `LoadModel` was implemented in Task 4 — confirm it passes.

- [ ] **Step 3: Run test to verify it passes**

Run: `go test ./internal/lmstudio/... -run 'TestLoadModel' -v`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add internal/lmstudio/client_test.go
git commit -m "Add LoadModel HTTP request/response tests"
```

---

### Task 6: UnloadModel

**Files:**
- Modify: `internal/lmstudio/client_test.go`

- [ ] **Step 1: Write the failing test (append to `client_test.go`)**

```go
func TestUnloadModel_SendsInstanceID(t *testing.T) {
	var gotBody unloadModelRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/models/unload" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.Write([]byte(`{"instance_id":"inst-1"}`))
	}))
	defer srv.Close()

	c := NewHTTPClient(strings.TrimPrefix(srv.URL, "http://"), "")
	if err := c.UnloadModel(context.Background(), "inst-1"); err != nil {
		t.Fatalf("UnloadModel: %v", err)
	}
	if gotBody.InstanceID != "inst-1" {
		t.Errorf("InstanceID = %q, want %q", gotBody.InstanceID, "inst-1")
	}
}

func TestUnloadModel_404ReturnsGenericErrorNotErrModelNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := NewHTTPClient(strings.TrimPrefix(srv.URL, "http://"), "")
	err := c.UnloadModel(context.Background(), "inst-missing")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	// UnloadModel deliberately does NOT map 404 to ErrModelNotFound: that
	// error's "run 'lmsctl models'" advice is for a model key the user
	// typed, not an instance ID this package resolved itself.
	var notFound *ErrModelNotFound
	if errors.As(err, &notFound) {
		t.Fatalf("err = %v, want a generic error, not *ErrModelNotFound (instance IDs aren't model keys)", err)
	}
}
```

- [ ] **Step 2: Run test to verify it passes**

Run: `go test ./internal/lmstudio/... -v`
Expected: PASS — full `internal/lmstudio` suite green.

- [ ] **Step 3: Commit**

```bash
git add internal/lmstudio/client_test.go
git commit -m "Add UnloadModel HTTP request/response tests"
```

---

### Task 7: Fake client for command tests

**Files:**
- Create: `internal/lmstudio/lmstudiotest/fake.go`

- [ ] **Step 1: Write `internal/lmstudio/lmstudiotest/fake.go`**

```go
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
```

- [ ] **Step 2: Verify it compiles**

Run: `go build ./...`
Expected: no errors (the `var _ lmstudio.Client = (*Fake)(nil)` line fails to compile if `Fake` doesn't fully implement `Client`).

- [ ] **Step 3: Commit**

```bash
git add internal/lmstudio/lmstudiotest/fake.go
git commit -m "Add fake lmstudio.Client for command tests"
```

---

### Task 8: Config resolution (flag > env > file)

**Files:**
- Create: `internal/config/config.go`
- Test: `internal/config/config_test.go`

- [ ] **Step 1: Write the failing tests**

```go
package config

import (
	"path/filepath"
	"testing"
)

func TestResolve_PrecedenceFlagWinsOverEnvAndFile(t *testing.T) {
	eff, err := Resolve("flag-host:1234", "", "env-host:1234", "", Config{Host: "file-host:1234"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if eff.Host != "flag-host:1234" {
		t.Errorf("Host = %q, want %q", eff.Host, "flag-host:1234")
	}
}

func TestResolve_EnvWinsOverFile(t *testing.T) {
	eff, err := Resolve("", "", "env-host:1234", "", Config{Host: "file-host:1234"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if eff.Host != "env-host:1234" {
		t.Errorf("Host = %q, want %q", eff.Host, "env-host:1234")
	}
}

func TestResolve_FallsBackToFile(t *testing.T) {
	eff, err := Resolve("", "", "", "", Config{Host: "file-host:1234"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if eff.Host != "file-host:1234" {
		t.Errorf("Host = %q, want %q", eff.Host, "file-host:1234")
	}
}

func TestResolve_TokenFollowsSamePrecedence(t *testing.T) {
	eff, err := Resolve("host:1234", "flag-token", "", "env-token", Config{Token: "file-token"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if eff.Token != "flag-token" {
		t.Errorf("Token = %q, want %q", eff.Token, "flag-token")
	}
}

func TestResolve_NoHostAnywhereReturnsErrNoHost(t *testing.T) {
	_, err := Resolve("", "", "", "", Config{})
	if err != ErrNoHost {
		t.Errorf("err = %v, want ErrNoHost", err)
	}
}

func TestSaveAndLoad_RoundTrips(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	want := Config{Host: "192.168.1.50:1234", Token: "secret"}
	if err := Save(want); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got != want {
		t.Errorf("Load() = %+v, want %+v", got, want)
	}

	path, _ := Path()
	if filepath.Dir(path) != filepath.Join(dir, "lmsctl") {
		t.Errorf("unexpected config dir: %s", path)
	}
}

func TestLoad_MissingFileReturnsZeroValue(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	got, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got != (Config{}) {
		t.Errorf("Load() = %+v, want zero value", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/config/... -v`
Expected: FAIL — package `internal/config` doesn't exist yet.

- [ ] **Step 3: Add the yaml dependency**

Run: `go get gopkg.in/yaml.v3@latest`
Expected: `go.mod`/`go.sum` updated.

- [ ] **Step 4: Write `internal/config/config.go`**

```go
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config is the on-disk shape of ~/.config/lmsctl/config.yaml.
type Config struct {
	Host  string `yaml:"host"`
	Token string `yaml:"token,omitempty"`
}

// Path returns the path to the config file, honoring XDG_CONFIG_HOME.
func Path() (string, error) {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("determining home directory: %w", err)
		}
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "lmsctl", "config.yaml"), nil
}

// Load reads the config file. A missing file is not an error; it returns a
// zero-value Config.
func Load() (Config, error) {
	path, err := Path()
	if err != nil {
		return Config{}, err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return Config{}, nil
	}
	if err != nil {
		return Config{}, fmt.Errorf("reading config file %s: %w", path, err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parsing config file %s: %w", path, err)
	}
	return cfg, nil
}

// Save writes the config file, creating its parent directory if needed.
func Save(cfg Config) error {
	path, err := Path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("encoding config: %w", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("writing config file %s: %w", path, err)
	}
	return nil
}

// Effective holds the fully-resolved settings for one command invocation.
type Effective struct {
	Host  string
	Token string
}

// ErrNoHost is returned by Resolve when no host is configured anywhere.
var ErrNoHost = fmt.Errorf("no host configured: run 'lmsctl config set-host <host:port>', set LMSCTL_HOST, or pass --host")

// Resolve applies the precedence flag > env var > config file to produce
// the effective settings for one invocation. Empty strings mean "not set"
// at that level.
func Resolve(flagHost, flagToken, envHost, envToken string, fileCfg Config) (Effective, error) {
	host := firstNonEmpty(flagHost, envHost, fileCfg.Host)
	if host == "" {
		return Effective{}, ErrNoHost
	}
	token := firstNonEmpty(flagToken, envToken, fileCfg.Token)
	return Effective{Host: host, Token: token}, nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./internal/config/... -v`
Expected: PASS (all 7 tests)

- [ ] **Step 6: Commit**

```bash
git add go.mod go.sum internal/config/config.go internal/config/config_test.go
git commit -m "Add config file loading and flag/env/file resolution"
```

---

### Task 9: Wire config into the root command

**Files:**
- Modify: `cmd/root.go`

- [ ] **Step 1: Add `newClient` to `cmd/root.go`**

Append to the end of `cmd/root.go`:

```go
// newClient resolves the effective host/token and builds a real
// lmstudio.Client for a command to use.
func newClient() (lmstudio.Client, config.Effective, error) {
	fileCfg, err := config.Load()
	if err != nil {
		return nil, config.Effective{}, err
	}
	eff, err := config.Resolve(flagHost, flagToken, os.Getenv("LMSCTL_HOST"), os.Getenv("LMSCTL_TOKEN"), fileCfg)
	if err != nil {
		return nil, config.Effective{}, err
	}
	return lmstudio.NewHTTPClient(eff.Host, eff.Token), eff, nil
}
```

Update the import block at the top of `cmd/root.go` to:

```go
import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"lmsctl/internal/config"
	"lmsctl/internal/lmstudio"
)
```

- [ ] **Step 2: Verify it builds**

Run: `go build ./...`
Expected: no errors. (`newClient` is unused by any command yet, which is fine — it's exported to the package, not to an external caller, so no "unused" error occurs for package-level functions.)

- [ ] **Step 3: Commit**

```bash
git add cmd/root.go
git commit -m "Wire config resolution into an lmstudio client factory"
```

---

### Task 10: `lmsctl config` subcommand

**Files:**
- Create: `cmd/config.go`
- Test: `cmd/config_test.go`

- [ ] **Step 1: Write the failing tests**

```go
package cmd

import (
	"bytes"
	"strings"
	"testing"

	"lmsctl/internal/config"
)

func TestConfigSetHost_WritesHostToConfigFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	out := &bytes.Buffer{}
	configSetHostCmd.SetOut(out)
	configSetHostCmd.SetArgs([]string{"192.168.1.50:1234"})
	if err := configSetHostCmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out.String(), "192.168.1.50:1234") {
		t.Errorf("output = %q, want it to mention the host", out.String())
	}

	got, err := config.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Host != "192.168.1.50:1234" {
		t.Errorf("saved host = %q, want %q", got.Host, "192.168.1.50:1234")
	}
}

func TestConfigShow_RedactsToken(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	if err := config.Save(config.Config{Host: "host:1234", Token: "super-secret"}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	out := &bytes.Buffer{}
	configShowCmd.SetOut(out)
	configShowCmd.SetArgs([]string{})
	if err := configShowCmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if strings.Contains(out.String(), "super-secret") {
		t.Errorf("output leaked the token: %q", out.String())
	}
	if !strings.Contains(out.String(), "host:1234") {
		t.Errorf("output = %q, want it to contain the host", out.String())
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/... -run 'TestConfig' -v`
Expected: FAIL — `configSetHostCmd`/`configShowCmd` undefined.

- [ ] **Step 3: Write `cmd/config.go`**

```go
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"lmsctl/internal/config"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "View or change lmsctl's configuration",
}

var configSetHostCmd = &cobra.Command{
	Use:   "set-host <host:port>",
	Short: "Set the default remote LM Studio host",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		cfg.Host = args[0]
		if err := config.Save(cfg); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Default host set to %s\n", args[0])
		return nil
	},
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Print the effective configuration (token redacted)",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		token := "(not set)"
		if cfg.Token != "" {
			token = "(set)"
		}
		fmt.Fprintf(cmd.OutOrStdout(), "host:  %s\ntoken: %s\n", cfg.Host, token)
		return nil
	},
}

func init() {
	configCmd.AddCommand(configSetHostCmd, configShowCmd)
	rootCmd.AddCommand(configCmd)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./cmd/... -run 'TestConfig' -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add cmd/config.go cmd/config_test.go
git commit -m "Add lmsctl config set-host/show subcommands"
```

---

### Task 11: Output helpers (JSON + table)

**Files:**
- Create: `internal/output/output.go`
- Test: `internal/output/output_test.go`

- [ ] **Step 1: Write the failing test**

```go
package output

import (
	"bytes"
	"strings"
	"testing"
)

func TestJSON_WritesIndentedJSON(t *testing.T) {
	var buf bytes.Buffer
	if err := JSON(&buf, map[string]string{"key": "value"}); err != nil {
		t.Fatalf("JSON: %v", err)
	}
	if !strings.Contains(buf.String(), "\"key\": \"value\"") {
		t.Errorf("output = %q, want indented JSON containing the key/value", buf.String())
	}
}

func TestNewTable_AlignsColumns(t *testing.T) {
	var buf bytes.Buffer
	tw := NewTable(&buf)
	tw.Write([]byte("A\tBB\n"))
	tw.Write([]byte("CC\tD\n"))
	if err := tw.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}
	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("got %d lines, want 2: %q", len(lines), buf.String())
	}
	// Equal line length is a poor proxy for "aligned" (text/tabwriter doesn't
	// pad an untabbed last cell, so it isn't achievable here anyway, and
	// achieving it via a wrapper adds trailing whitespace to every row plus
	// edge cases for tab-free lines). Check what "aligned" actually means:
	// the second column starts at the same offset on both lines.
	if strings.Index(lines[0], "BB") != strings.Index(lines[1], "D") {
		t.Errorf("columns not aligned: %q vs %q", lines[0], lines[1])
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/output/... -v`
Expected: FAIL — package `internal/output` doesn't exist yet.

- [ ] **Step 3: Write `internal/output/output.go`**

```go
// Package output renders command results as either human-readable tables
// or machine-readable JSON.
package output

import (
	"encoding/json"
	"io"
	"text/tabwriter"
)

// JSON writes v to w as indented JSON.
func JSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// NewTable returns a tabwriter configured for aligned column output. Write
// tab-separated rows to it and call Flush when done.
func NewTable(w io.Writer) *tabwriter.Writer {
	return tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/output/... -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/output/output.go internal/output/output_test.go
git commit -m "Add JSON and table output helpers"
```

---

### Task 12: `lmsctl status`

**Files:**
- Create: `cmd/status.go`
- Test: `cmd/status_test.go`

- [ ] **Step 1: Write the failing tests**

```go
package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"lmsctl/internal/lmstudio"
	"lmsctl/internal/lmstudio/lmstudiotest"
)

func TestRunStatus_NoModelsLoaded(t *testing.T) {
	fake := &lmstudiotest.Fake{
		ModelsResponse: &lmstudio.ModelsResponse{Models: []lmstudio.Model{
			{Key: "openai/gpt-oss-20b"},
		}},
	}
	cmd := &cobra.Command{}
	out := &bytes.Buffer{}
	cmd.SetOut(out)

	if err := runStatus(cmd, fake, "192.168.1.50:1234"); err != nil {
		t.Fatalf("runStatus: %v", err)
	}
	if !strings.Contains(out.String(), "No models currently loaded") {
		t.Errorf("output = %q, want it to say nothing is loaded", out.String())
	}
}

func TestRunStatus_ReportsLoadedModel(t *testing.T) {
	fake := &lmstudiotest.Fake{
		ModelsResponse: &lmstudio.ModelsResponse{Models: []lmstudio.Model{
			{Key: "openai/gpt-oss-20b", LoadedInstances: []lmstudio.LoadedInstance{{ID: "inst-1"}}},
		}},
	}
	cmd := &cobra.Command{}
	out := &bytes.Buffer{}
	cmd.SetOut(out)

	if err := runStatus(cmd, fake, "192.168.1.50:1234"); err != nil {
		t.Fatalf("runStatus: %v", err)
	}
	if !strings.Contains(out.String(), "openai/gpt-oss-20b (inst-1)") {
		t.Errorf("output = %q, want it to mention the loaded model", out.String())
	}
}

func TestRunStatus_PropagatesClientError(t *testing.T) {
	fake := &lmstudiotest.Fake{ListModelsErr: &lmstudio.ErrUnreachable{Host: "http://host:1234"}}
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})

	if err := runStatus(cmd, fake, "host:1234"); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestRunStatus_JSONOutput(t *testing.T) {
	flagJSON = true
	defer func() { flagJSON = false }()

	fake := &lmstudiotest.Fake{
		ModelsResponse: &lmstudio.ModelsResponse{Models: []lmstudio.Model{
			{Key: "openai/gpt-oss-20b", LoadedInstances: []lmstudio.LoadedInstance{{ID: "inst-1"}}},
		}},
	}
	cmd := &cobra.Command{}
	out := &bytes.Buffer{}
	cmd.SetOut(out)

	if err := runStatus(cmd, fake, "host:1234"); err != nil {
		t.Fatalf("runStatus: %v", err)
	}
	if !strings.Contains(out.String(), `"openai/gpt-oss-20b (inst-1)"`) {
		t.Errorf("output = %q, want JSON containing the loaded model", out.String())
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/... -run 'TestRunStatus' -v`
Expected: FAIL — `runStatus`/`statusCmd` undefined.

- [ ] **Step 3: Write `cmd/status.go`**

```go
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"lmsctl/internal/lmstudio"
	"lmsctl/internal/output"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check whether the remote LM Studio server is reachable and what's loaded",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, eff, err := newClient()
		if err != nil {
			return err
		}
		return runStatus(cmd, client, eff.Host)
	},
}

func runStatus(cmd *cobra.Command, client lmstudio.Client, host string) error {
	resp, err := client.ListModels(cmd.Context())
	if err != nil {
		return err
	}

	var loaded []string
	for _, m := range resp.Models {
		for _, inst := range m.LoadedInstances {
			loaded = append(loaded, fmt.Sprintf("%s (%s)", m.Key, inst.ID))
		}
	}

	if flagJSON {
		return output.JSON(cmd.OutOrStdout(), map[string]any{
			"reachable":     true,
			"host":          host,
			"loaded_models": loaded,
		})
	}

	fmt.Fprintf(cmd.OutOrStdout(), "LM Studio at %s: reachable\n", host)
	if len(loaded) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No models currently loaded.")
		return nil
	}
	fmt.Fprintln(cmd.OutOrStdout(), "Loaded models:")
	for _, l := range loaded {
		fmt.Fprintf(cmd.OutOrStdout(), "  - %s\n", l)
	}
	return nil
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./cmd/... -run 'TestRunStatus' -v`
Expected: PASS (all 4 subtests)

- [ ] **Step 5: Commit**

```bash
git add cmd/status.go cmd/status_test.go
git commit -m "Add lmsctl status command"
```

---

### Task 13: `lmsctl models`

**Files:**
- Create: `cmd/models.go`
- Test: `cmd/models_test.go`

- [ ] **Step 1: Write the failing tests**

```go
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

	if err := runModels(cmd, fake); err != nil {
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
	flagJSON = true
	defer func() { flagJSON = false }()

	fake := &lmstudiotest.Fake{
		ModelsResponse: &lmstudio.ModelsResponse{Models: []lmstudio.Model{
			{Key: "openai/gpt-oss-20b"},
		}},
	}
	cmd := &cobra.Command{}
	out := &bytes.Buffer{}
	cmd.SetOut(out)

	if err := runModels(cmd, fake); err != nil {
		t.Fatalf("runModels: %v", err)
	}
	if !strings.Contains(out.String(), `"key": "openai/gpt-oss-20b"`) {
		t.Errorf("output = %q, want JSON containing the model key", out.String())
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/... -run 'TestRunModels|TestFormatBytes' -v`
Expected: FAIL — `runModels`/`formatBytes` undefined.

- [ ] **Step 3: Write `cmd/models.go`**

```go
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"lmsctl/internal/lmstudio"
	"lmsctl/internal/output"
)

var modelsCmd = &cobra.Command{
	Use:     "models",
	Aliases: []string{"ls"},
	Short:   "List downloaded models and whether each is loaded",
	Args:    cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, _, err := newClient()
		if err != nil {
			return err
		}
		return runModels(cmd, client)
	},
}

func runModels(cmd *cobra.Command, client lmstudio.Client) error {
	resp, err := client.ListModels(cmd.Context())
	if err != nil {
		return err
	}

	if flagJSON {
		return output.JSON(cmd.OutOrStdout(), resp.Models)
	}

	tw := output.NewTable(cmd.OutOrStdout())
	fmt.Fprintln(tw, "KEY\tSIZE\tQUANTIZATION\tSTATE")
	for _, m := range resp.Models {
		quant := "-"
		if m.Quantization != nil {
			quant = m.Quantization.Name
		}
		state := "not-loaded"
		if len(m.LoadedInstances) > 0 {
			state = "loaded"
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", m.Key, formatBytes(m.SizeBytes), quant, state)
	}
	return tw.Flush()
}

func formatBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%dB", n)
	}
	div, exp := int64(unit), 0
	for x := n / unit; x >= unit; x /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%ciB", float64(n)/float64(div), "KMGTPE"[exp])
}

func init() {
	rootCmd.AddCommand(modelsCmd)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./cmd/... -run 'TestRunModels|TestFormatBytes' -v`
Expected: PASS (all subtests)

- [ ] **Step 5: Commit**

```bash
git add cmd/models.go cmd/models_test.go
git commit -m "Add lmsctl models command"
```

---

### Task 14: `lmsctl load`

**Files:**
- Create: `cmd/load.go`
- Test: `cmd/load_test.go`

- [ ] **Step 1: Write the failing tests**

```go
package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"

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

	if err := runLoad(cmd, fake, "openai/gpt-oss-20b"); err != nil {
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

	if err := runLoad(cmd, fake, "nope"); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestRunLoad_JSONOutput(t *testing.T) {
	flagJSON = true
	defer func() { flagJSON = false }()

	fake := &lmstudiotest.Fake{
		LoadModelResponse: &lmstudio.LoadModelResponse{InstanceID: "inst-1"},
	}
	cmd := &cobra.Command{}
	out := &bytes.Buffer{}
	cmd.SetOut(out)

	if err := runLoad(cmd, fake, "openai/gpt-oss-20b"); err != nil {
		t.Fatalf("runLoad: %v", err)
	}
	if !strings.Contains(out.String(), `"instance_id": "inst-1"`) {
		t.Errorf("output = %q, want JSON containing the instance id", out.String())
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/... -run 'TestRunLoad' -v`
Expected: FAIL — `runLoad`/`loadCmd` undefined.

- [ ] **Step 3: Write `cmd/load.go`**

```go
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"lmsctl/internal/lmstudio"
	"lmsctl/internal/output"
)

var (
	loadFlagContextLength int
	loadFlagFlashAttn     bool
	loadFlagOffloadKV     bool
)

var loadCmd = &cobra.Command{
	Use:   "load <model>",
	Short: "Load a model on the remote LM Studio instance",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, _, err := newClient()
		if err != nil {
			return err
		}
		return runLoad(cmd, client, args[0])
	},
}

func runLoad(cmd *cobra.Command, client lmstudio.Client, model string) error {
	req := lmstudio.LoadModelRequest{Model: model}
	if cmd.Flags().Changed("context-length") {
		v := loadFlagContextLength
		req.ContextLength = &v
	}
	if cmd.Flags().Changed("flash-attention") {
		v := loadFlagFlashAttn
		req.FlashAttention = &v
	}
	if cmd.Flags().Changed("offload-kv-cache-to-gpu") {
		v := loadFlagOffloadKV
		req.OffloadKVCacheToGPU = &v
	}

	resp, err := client.LoadModel(cmd.Context(), req)
	if err != nil {
		return err
	}

	if flagJSON {
		return output.JSON(cmd.OutOrStdout(), resp)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Loaded %s as instance %s (%.1fs)\n", model, resp.InstanceID, resp.LoadTimeSeconds)
	return nil
}

func init() {
	loadCmd.Flags().IntVar(&loadFlagContextLength, "context-length", 0, "context length to load the model with")
	loadCmd.Flags().BoolVar(&loadFlagFlashAttn, "flash-attention", false, "enable flash attention (llama.cpp models only)")
	loadCmd.Flags().BoolVar(&loadFlagOffloadKV, "offload-kv-cache-to-gpu", false, "offload the KV cache to GPU (llama.cpp models only)")
	rootCmd.AddCommand(loadCmd)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./cmd/... -run 'TestRunLoad' -v`
Expected: PASS (all 3 subtests)

- [ ] **Step 5: Commit**

```bash
git add cmd/load.go cmd/load_test.go
git commit -m "Add lmsctl load command"
```

---

### Task 15: `lmsctl unload`

**Files:**
- Create: `cmd/unload.go`
- Test: `cmd/unload_test.go`

- [ ] **Step 1: Write the failing tests**

```go
package cmd

import (
	"bytes"
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

func TestRunUnload_ModelNotLoadedReturnsError(t *testing.T) {
	fake := &lmstudiotest.Fake{ModelsResponse: modelsWithLoaded()}
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})

	err := runUnload(cmd, fake, "not/loaded", false)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "not/loaded") {
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/... -run 'TestRunUnload' -v`
Expected: FAIL — `runUnload`/`unloadCmd` undefined.

- [ ] **Step 3: Write `cmd/unload.go`**

```go
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"lmsctl/internal/lmstudio"
)

var unloadFlagAll bool

var unloadCmd = &cobra.Command{
	Use:   "unload [model]",
	Short: "Unload a model (or all loaded models) on the remote LM Studio instance",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, _, err := newClient()
		if err != nil {
			return err
		}
		var model string
		if len(args) == 1 {
			model = args[0]
		}
		return runUnload(cmd, client, model, unloadFlagAll)
	},
}

func runUnload(cmd *cobra.Command, client lmstudio.Client, model string, all bool) error {
	if !all && model == "" {
		return fmt.Errorf("specify a model to unload or pass --all")
	}

	resp, err := client.ListModels(cmd.Context())
	if err != nil {
		return err
	}

	var toUnload []string
	for _, m := range resp.Models {
		if !all && m.Key != model {
			continue
		}
		for _, inst := range m.LoadedInstances {
			toUnload = append(toUnload, inst.ID)
		}
	}

	if len(toUnload) == 0 {
		if all {
			fmt.Fprintln(cmd.OutOrStdout(), "No models currently loaded.")
			return nil
		}
		return &lmstudio.ErrModelNotLoaded{Model: model}
	}

	for _, id := range toUnload {
		if err := client.UnloadModel(cmd.Context(), id); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Unloaded instance %s\n", id)
	}
	return nil
}

func init() {
	unloadCmd.Flags().BoolVar(&unloadFlagAll, "all", false, "unload every currently loaded model")
	rootCmd.AddCommand(unloadCmd)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./cmd/... -run 'TestRunUnload' -v`
Expected: PASS (all 5 subtests)

- [ ] **Step 5: Commit**

```bash
git add cmd/unload.go cmd/unload_test.go
git commit -m "Add lmsctl unload command"
```

---

### Task 16: Full test suite, build, README, and manual smoke test

**Files:**
- Create: `README.md`

- [ ] **Step 1: Run the entire test suite**

Run: `go vet ./... && go test ./... -v`
Expected: PASS across `internal/config`, `internal/lmstudio`, `internal/output`, and `cmd`, no `go vet` warnings.

- [ ] **Step 2: Build the final binary**

Run: `go build -o /tmp/lmsctl . && /tmp/lmsctl --help`
Expected: usage listing `status`, `models`, `load`, `unload`, `config` as subcommands.

- [ ] **Step 3: Write `README.md`**

```markdown
# lmsctl

A CLI for managing an LM Studio instance running on another machine on your
local network — check status, list models, and load/unload them — over LM
Studio's `/api/v1` REST API.

## Setup

On the remote machine, in LM Studio's Developer settings, enable
**"Serve on Local Network"** (and set an API token there if you want
authenticated access).

## Install

```bash
go build -o lmsctl .
```

Move the resulting `lmsctl` binary onto your `$PATH`.

## Configure

```bash
lmsctl config set-host 192.168.1.50:1234
```

This is stored in `~/.config/lmsctl/config.yaml`. Override it per-command
with `--host`, or set `LMSCTL_HOST` in your shell profile. If the remote
server requires a token, set it the same way with `--token` /
`LMSCTL_TOKEN`, or add it directly to the config file's `token:` field.

## Usage

```bash
lmsctl status                          # is it up, what's loaded
lmsctl models                          # list downloaded models (alias: ls)
lmsctl load openai/gpt-oss-20b         # load a model
lmsctl load openai/gpt-oss-20b --context-length 16384 --flash-attention
lmsctl unload openai/gpt-oss-20b       # unload one model
lmsctl unload --all                    # unload everything loaded
lmsctl config show                     # see effective config (token redacted)
```

Add `--json` to any command for machine-readable output.
```

- [ ] **Step 4: Manual smoke test against your real remote LM Studio**

Run (replacing the host with your actual device's address):

```bash
/tmp/lmsctl config set-host <your-device-ip>:1234
/tmp/lmsctl status
/tmp/lmsctl models
```

Expected: `status` reports reachable and `models` lists your downloaded
models. If it fails, follow the error message (it will point at LM Studio's
"Serve on Local Network" setting or a token problem) rather than assuming
`lmsctl` is broken — this is the point where any remaining mismatch between
the assumed API shape and your actual LM Studio version would surface. If
the JSON shape differs from what Task 2/4/5 assumed, adjust the struct tags
in `internal/lmstudio/types.go` to match and re-run the test suite.

- [ ] **Step 5: Commit**

```bash
git add README.md
git commit -m "Add README and complete lmsctl v1"
```

---

## Plan self-review notes

- **Spec coverage:** model lifecycle (list/load/unload) → Tasks 4–6, 13–15; status/monitoring → Tasks 4, 12; REST API connection → Task 4; config file with default host, flag/env override → Tasks 8–10; Go + cobra + yaml → Tasks 1, 8; error handling for unreachable/401/404 → Tasks 3–6; JSON + table output → Tasks 11–15; testing approach (fake client, httptest, manual smoke test) → Tasks 4–7, 16.
- **Out of scope confirmed:** no `chat`/`pull` commands are included, matching the spec.
- **Open item resolved:** the spec's open item (exact `load`/`list` JSON schemas) was resolved by checking LM Studio's live developer docs before writing this plan; those schemas are recorded above and used verbatim in Task 2's types.
