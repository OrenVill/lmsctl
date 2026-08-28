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
