package azurenh

import (
	"context"
	"net/http"
	"testing"
)

func TestExec_Success(t *testing.T) {
	c := newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Header.Get("Authorization") == "" {
			t.Error("expected Authorization header")
		}
		if req.Header.Get("X-Custom") != "value" {
			t.Error("expected custom header")
		}
		return mockResponse(200, `{"ok":true}`, nil), nil
	})

	resp, err := c.exec(context.Background(), "GET", c.buildURL("messages"), map[string]string{"X-Custom": "value"}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.statusCode != 200 {
		t.Errorf("status = %d, want 200", resp.statusCode)
	}
	if string(resp.body) != `{"ok":true}` {
		t.Errorf("body = %q, want %q", string(resp.body), `{"ok":true}`)
	}
}

func TestExec_APIError(t *testing.T) {
	c := newTestClient(func(req *http.Request) (*http.Response, error) {
		return mockResponse(404, "not found", nil), nil
	})

	_, err := c.exec(context.Background(), "GET", c.buildURL("messages"), nil, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if !IsNotFound(err) {
		t.Errorf("expected NotFound error, got %v", err)
	}
}

func TestExec_WithResilience(t *testing.T) {
	attempts := 0
	c := newTestClient(func(req *http.Request) (*http.Response, error) {
		attempts++
		if attempts < 2 {
			return mockResponse(500, "server error", nil), nil
		}
		return mockResponse(200, "ok", nil), nil
	}, WithRetryPolicy(RetryPolicy{
		MaxAttempts:  3,
		InitialDelay: 0,
		Multiplier:   1,
	}))

	resp, err := c.exec(context.Background(), "GET", c.buildURL("messages"), nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.statusCode != 200 {
		t.Errorf("status = %d, want 200", resp.statusCode)
	}
	if attempts != 2 {
		t.Errorf("attempts = %d, want 2", attempts)
	}
}
