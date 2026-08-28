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
