package lmstudio

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
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

var _ Client = (*HTTPClient)(nil)

// NewHTTPClient builds an HTTPClient with no request timeout, since model
// loads can take arbitrarily long; connection failures still fail fast via
// the OS-level TCP error.
func NewHTTPClient(host, token string) *HTTPClient {
	host = strings.TrimPrefix(strings.TrimPrefix(host, "https://"), "http://")
	return &HTTPClient{
		BaseURL:    "http://" + host,
		Token:      token,
		HTTPClient: &http.Client{},
	}
}

// maxErrorBodyBytes caps how much of a response body we read, so a
// misdirected --host (a stray web server, a captive portal) can't dump an
// arbitrarily large page into the terminal via an error message.
const maxErrorBodyBytes = 4096

// httpStatusError wraps a non-2xx response that do() itself doesn't turn
// into a more specific error (401 is handled here as ErrUnauthorized;
// callers like LoadModel may map this further, e.g. 404 to ErrModelNotFound).
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

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxErrorBodyBytes))
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
	// Maps 404 to ErrInstanceNotFound rather than ErrModelNotFound: this
	// method only ever receives instance IDs the caller resolved itself via
	// ListModels, never a user-typed model key, so ErrModelNotFound's "run
	// 'lmsctl models'" advice would be wrong here. A 404 means the instance
	// is already gone — most likely already unloaded — which callers can
	// treat as a harmless race rather than a hard failure.
	err := c.do(ctx, http.MethodPost, "/api/v1/models/unload", unloadModelRequest{InstanceID: instanceID}, nil)
	var statusErr *httpStatusError
	if errors.As(err, &statusErr) && statusErr.StatusCode == http.StatusNotFound {
		return &ErrInstanceNotFound{InstanceID: instanceID}
	}
	return err
}
