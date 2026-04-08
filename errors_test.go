package azurenh

import (
	"errors"
	"net/http"
	"testing"
	"time"
)

func TestAPIError_Error(t *testing.T) {
	e := &APIError{Code: CodeNotFound, StatusCode: 404, Message: "hub not found"}
	want := "azurenh: NotFound (HTTP 404): hub not found"
	if got := e.Error(); got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}

	e2 := &APIError{Code: CodeNotFound, StatusCode: 404}
	want2 := "azurenh: NotFound (HTTP 404)"
	if got := e2.Error(); got != want2 {
		t.Errorf("Error() = %q, want %q", got, want2)
	}
}

func TestAPIError_IsRetryable(t *testing.T) {
	retryable := []int{429, 500, 503, 504, 408}
	for _, code := range retryable {
		e := &APIError{StatusCode: code}
		if !e.IsRetryable() {
			t.Errorf("expected status %d to be retryable", code)
		}
	}

	nonRetryable := []int{400, 401, 403, 404, 409}
	for _, code := range nonRetryable {
		e := &APIError{StatusCode: code}
		if e.IsRetryable() {
			t.Errorf("expected status %d to not be retryable", code)
		}
	}
}

func TestAPIError_RetryAfter(t *testing.T) {
	e := &APIError{RetryAfterHeader: "30"}
	if got := e.RetryAfter(); got != 30*time.Second {
		t.Errorf("RetryAfter() = %v, want 30s", got)
	}

	e2 := &APIError{RetryAfterHeader: ""}
	if got := e2.RetryAfter(); got != 0 {
		t.Errorf("RetryAfter() = %v, want 0", got)
	}
}

func TestClassifyHTTPResponse_Success(t *testing.T) {
	result := classifyHTTPResponse(200, http.Header{}, nil)
	if result != nil {
		t.Errorf("expected nil for 200, got %v", result)
	}

	result = classifyHTTPResponse(201, http.Header{}, nil)
	if result != nil {
		t.Errorf("expected nil for 201, got %v", result)
	}
}

func TestClassifyHTTPResponse_XMLError(t *testing.T) {
	body := []byte(`<Error><Code>404</Code><Detail>Hub not found</Detail></Error>`)
	headers := http.Header{}
	headers.Set("x-ms-request-id", "req-123")

	result := classifyHTTPResponse(404, headers, body)
	if result == nil {
		t.Fatal("expected error for 404")
	}
	if result.Code != "404" {
		t.Errorf("Code = %q, want %q", result.Code, "404")
	}
	if result.Message != "Hub not found" {
		t.Errorf("Message = %q, want %q", result.Message, "Hub not found")
	}
	if result.RequestID != "req-123" {
		t.Errorf("RequestID = %q, want %q", result.RequestID, "req-123")
	}
}

func TestClassifyHTTPResponse_PlainError(t *testing.T) {
	body := []byte("bad request")
	result := classifyHTTPResponse(400, http.Header{}, body)
	if result == nil {
		t.Fatal("expected error for 400")
	}
	if result.Code != CodeBadRequest {
		t.Errorf("Code = %q, want %q", result.Code, CodeBadRequest)
	}
	if result.Message != "bad request" {
		t.Errorf("Message = %q, want %q", result.Message, "bad request")
	}
}

func TestIsNotFound(t *testing.T) {
	err := &APIError{StatusCode: 404, Code: CodeNotFound}
	if !IsNotFound(err) {
		t.Error("expected IsNotFound to be true")
	}
	if IsNotFound(errors.New("other error")) {
		t.Error("expected IsNotFound to be false for non-API error")
	}
}

func TestIsUnauthorized(t *testing.T) {
	if !IsUnauthorized(&APIError{StatusCode: 401}) {
		t.Error("expected 401 to be unauthorized")
	}
	if !IsUnauthorized(&APIError{StatusCode: 403}) {
		t.Error("expected 403 to be unauthorized")
	}
	if IsUnauthorized(&APIError{StatusCode: 404}) {
		t.Error("expected 404 to not be unauthorized")
	}
}

func TestIsThrottled(t *testing.T) {
	if !IsThrottled(&APIError{StatusCode: 429}) {
		t.Error("expected 429 to be throttled")
	}
	if IsThrottled(&APIError{StatusCode: 500}) {
		t.Error("expected 500 to not be throttled")
	}
}

func TestValidationError(t *testing.T) {
	e := &ValidationError{Field: "platform", Message: "required"}
	want := "azurenh: validation error on platform: required"
	if got := e.Error(); got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}
