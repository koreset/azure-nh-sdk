package azurenh

import (
	"context"
	"testing"
	"time"
)

func TestRetryPolicy_Delay(t *testing.T) {
	rp := RetryPolicy{
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     5 * time.Second,
		Multiplier:   2.0,
		JitterFactor: 0, // no jitter for deterministic test
	}

	d0 := rp.delay(0)
	if d0 != 100*time.Millisecond {
		t.Errorf("delay(0) = %v, want 100ms", d0)
	}

	d1 := rp.delay(1)
	if d1 != 200*time.Millisecond {
		t.Errorf("delay(1) = %v, want 200ms", d1)
	}

	d2 := rp.delay(2)
	if d2 != 400*time.Millisecond {
		t.Errorf("delay(2) = %v, want 400ms", d2)
	}

	// Test max delay cap.
	d10 := rp.delay(10)
	if d10 > 5*time.Second {
		t.Errorf("delay(10) = %v, should be capped at 5s", d10)
	}
}

func TestCircuitBreaker_TripsOpen(t *testing.T) {
	clock := &fixedClock{t: time.Now()}
	cb := newCircuitBreaker(CircuitBreakerConfig{
		FailureThreshold: 3,
		ResetTimeout:     10 * time.Second,
	}, clock)

	for i := 0; i < 3; i++ {
		cb.recordFailure()
	}

	if cb.State() != CircuitOpen {
		t.Errorf("expected CircuitOpen, got %d", cb.State())
	}

	err := cb.allow()
	if err != ErrCircuitOpen {
		t.Errorf("expected ErrCircuitOpen, got %v", err)
	}
}

func TestCircuitBreaker_HalfOpen(t *testing.T) {
	clock := &fixedClock{t: time.Now()}
	cb := newCircuitBreaker(CircuitBreakerConfig{
		FailureThreshold: 2,
		ResetTimeout:     5 * time.Second,
		HalfOpenMax:      1,
	}, clock)

	cb.recordFailure()
	cb.recordFailure()

	// Advance time past reset timeout.
	clock.t = clock.t.Add(6 * time.Second)

	if cb.State() != CircuitHalfOpen {
		t.Errorf("expected CircuitHalfOpen, got %d", cb.State())
	}

	// First request should be allowed.
	if err := cb.allow(); err != nil {
		t.Errorf("expected nil, got %v", err)
	}

	// Second request in half-open should be denied.
	if err := cb.allow(); err != ErrCircuitOpen {
		t.Errorf("expected ErrCircuitOpen, got %v", err)
	}
}

func TestCircuitBreaker_RecoveryOnSuccess(t *testing.T) {
	clock := &fixedClock{t: time.Now()}
	cb := newCircuitBreaker(CircuitBreakerConfig{
		FailureThreshold: 2,
		ResetTimeout:     5 * time.Second,
	}, clock)

	cb.recordFailure()
	cb.recordFailure()
	clock.t = clock.t.Add(6 * time.Second)

	// Allow half-open request.
	cb.allow()
	cb.recordSuccess()

	if cb.State() != CircuitClosed {
		t.Errorf("expected CircuitClosed after success, got %d", cb.State())
	}
}

func TestResiliencePolicy_NonRetryableError(t *testing.T) {
	attempts := 0
	rp := &resiliencePolicy{
		retry: &RetryPolicy{MaxAttempts: 3, InitialDelay: 0},
	}

	_, err := rp.execute(context.Background(), func(ctx context.Context) (*response, error) {
		attempts++
		return nil, &APIError{StatusCode: 400, Code: CodeBadRequest}
	})

	if attempts != 1 {
		t.Errorf("expected 1 attempt for non-retryable error, got %d", attempts)
	}
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestResiliencePolicy_ExhaustsRetries(t *testing.T) {
	attempts := 0
	rp := &resiliencePolicy{
		retry: &RetryPolicy{MaxAttempts: 3, InitialDelay: 0},
	}

	_, err := rp.execute(context.Background(), func(ctx context.Context) (*response, error) {
		attempts++
		return nil, &APIError{StatusCode: 500, Code: CodeInternalServerError}
	})

	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
	if err == nil {
		t.Fatal("expected error")
	}
}
