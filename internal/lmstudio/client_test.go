package lmstudio

import (
	"context"
	"encoding/json"
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

func TestListModels_500ReturnsGenericHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	}))
	defer srv.Close()

	c := NewHTTPClient(strings.TrimPrefix(srv.URL, "http://"), "")
	_, err := c.ListModels(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "500") || !strings.Contains(err.Error(), "internal error") {
		t.Errorf("err = %v, want it to mention the status code and body", err)
	}
}

func TestListModels_MalformedJSONReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not json"))
	}))
	defer srv.Close()

	c := NewHTTPClient(strings.TrimPrefix(srv.URL, "http://"), "")
	_, err := c.ListModels(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

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
