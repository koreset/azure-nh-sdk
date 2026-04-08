package azurenh

import (
	"encoding/xml"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// Sentinel errors for errors.Is() matching.
var (
	ErrInvalidConnectionString = errors.New("azurenh: invalid connection string")
	ErrInvalidPayload          = errors.New("azurenh: invalid notification payload")
	ErrInvalidPlatform         = errors.New("azurenh: invalid platform")
	ErrCircuitOpen             = errors.New("azurenh: circuit breaker is open")
	ErrRateLimited             = errors.New("azurenh: rate limited")
	ErrBatchTooLarge           = errors.New("azurenh: batch exceeds 1000 device limit")
	ErrScheduleInPast          = errors.New("azurenh: cannot schedule notification in the past")
)

// ErrorCode maps to Azure Notification Hubs service error codes.
type ErrorCode string

const (
	CodeBadRequest            ErrorCode = "BadRequest"
	CodeUnauthorized          ErrorCode = "Unauthorized"
	CodeForbidden             ErrorCode = "Forbidden"
	CodeNotFound              ErrorCode = "NotFound"
	CodeConflict              ErrorCode = "Conflict"
	CodeRequestEntityTooLarge ErrorCode = "RequestEntityTooLarge"
	CodeTooManyRequests       ErrorCode = "TooManyRequests"
	CodeInternalServerError   ErrorCode = "InternalServerError"
	CodeServiceUnavailable    ErrorCode = "ServiceUnavailable"
	CodeGatewayTimeout        ErrorCode = "GatewayTimeout"
)

// APIError represents an error response from the Azure Notification Hubs REST API.
type APIError struct {
	Code       ErrorCode `json:"code" xml:"Code"`
	Message    string    `json:"message" xml:"Detail"`
	StatusCode int       `json:"-" xml:"-"`
	RequestID  string    `json:"-" xml:"-"`
	RetryAfterHeader string `json:"-" xml:"-"`
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("azurenh: %s (HTTP %d): %s", e.Code, e.StatusCode, e.Message)
	}
	return fmt.Sprintf("azurenh: %s (HTTP %d)", e.Code, e.StatusCode)
}

// IsRetryable returns true if the error indicates the request should be retried.
func (e *APIError) IsRetryable() bool {
	switch e.StatusCode {
	case http.StatusTooManyRequests,
		http.StatusInternalServerError,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout,
		http.StatusRequestTimeout:
		return true
	}
	return false
}

// RetryAfter returns the Retry-After duration if present in the response, or zero.
func (e *APIError) RetryAfter() time.Duration {
	if e.RetryAfterHeader == "" {
		return 0
	}
	if seconds, err := strconv.Atoi(e.RetryAfterHeader); err == nil {
		return time.Duration(seconds) * time.Second
	}
	if t, err := http.ParseTime(e.RetryAfterHeader); err == nil {
		d := time.Until(t)
		if d > 0 {
			return d
		}
	}
	return 0
}

// ValidationError represents client-side input validation failures.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("azurenh: validation error on %s: %s", e.Field, e.Message)
}

// classifyHTTPResponse inspects an HTTP response and returns an *APIError
// for non-2xx status codes, or nil for success.
func classifyHTTPResponse(statusCode int, headers http.Header, body []byte) *APIError {
	if statusCode >= 200 && statusCode < 300 {
		return nil
	}

	apiErr := &APIError{
		StatusCode:       statusCode,
		RequestID:        headers.Get("x-ms-request-id"),
		RetryAfterHeader: headers.Get("Retry-After"),
	}

	// Try XML parsing first (Azure NH often returns XML errors).
	var xmlErr struct {
		Code   string `xml:"Code"`
		Detail string `xml:"Detail"`
	}
	if xml.Unmarshal(body, &xmlErr) == nil && xmlErr.Code != "" {
		apiErr.Code = ErrorCode(xmlErr.Code)
		apiErr.Message = xmlErr.Detail
		return apiErr
	}

	// Fall back to status code mapping.
	apiErr.Code = statusCodeToErrorCode(statusCode)
	if len(body) > 0 {
		apiErr.Message = string(body)
	}
	return apiErr
}

func statusCodeToErrorCode(statusCode int) ErrorCode {
	switch statusCode {
	case http.StatusBadRequest:
		return CodeBadRequest
	case http.StatusUnauthorized:
		return CodeUnauthorized
	case http.StatusForbidden:
		return CodeForbidden
	case http.StatusNotFound:
		return CodeNotFound
	case http.StatusConflict:
		return CodeConflict
	case http.StatusRequestEntityTooLarge:
		return CodeRequestEntityTooLarge
	case http.StatusTooManyRequests:
		return CodeTooManyRequests
	case http.StatusInternalServerError:
		return CodeInternalServerError
	case http.StatusServiceUnavailable:
		return CodeServiceUnavailable
	case http.StatusGatewayTimeout:
		return CodeGatewayTimeout
	default:
		return ErrorCode(fmt.Sprintf("HTTP%d", statusCode))
	}
}

// IsNotFound returns true if the error is a 404 Not Found from the API.
func IsNotFound(err error) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound
}

// IsUnauthorized returns true if the error is a 401/403 auth error.
func IsUnauthorized(err error) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && (apiErr.StatusCode == http.StatusUnauthorized || apiErr.StatusCode == http.StatusForbidden)
}

// IsThrottled returns true if the error is a 429 Too Many Requests.
func IsThrottled(err error) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusTooManyRequests
}
