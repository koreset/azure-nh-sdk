package azurenh

import (
	"context"
	"math"
	"math/rand/v2"
	"sync"
	"time"
)

// RetryPolicy configures retry behavior with exponential backoff.
type RetryPolicy struct {
	MaxAttempts  int           // Maximum number of attempts (default 3, includes initial).
	InitialDelay time.Duration // Delay before first retry (default 500ms).
	MaxDelay     time.Duration // Maximum delay between retries (default 30s).
	Multiplier   float64       // Backoff multiplier (default 2.0).
	JitterFactor float64       // Jitter as fraction of delay (default 0.1).
}

// DefaultRetryPolicy returns a RetryPolicy with sensible defaults.
func DefaultRetryPolicy() RetryPolicy {
	return RetryPolicy{
		MaxAttempts:  3,
		InitialDelay: 500 * time.Millisecond,
		MaxDelay:     30 * time.Second,
		Multiplier:   2.0,
		JitterFactor: 0.1,
	}
}

func (rp RetryPolicy) delay(attempt int) time.Duration {
	d := float64(rp.InitialDelay) * math.Pow(rp.Multiplier, float64(attempt))
	if d > float64(rp.MaxDelay) {
		d = float64(rp.MaxDelay)
	}
	jitter := d * rp.JitterFactor * (rand.Float64()*2 - 1)
	d += jitter
	if d < 0 {
		d = 0
	}
	return time.Duration(d)
}

// CircuitBreakerConfig configures the circuit breaker.
type CircuitBreakerConfig struct {
	FailureThreshold int           // Consecutive failures to trip open (default 5).
	ResetTimeout     time.Duration // Duration in open state before half-open (default 30s).
	HalfOpenMax      int           // Max requests in half-open state (default 1).
}

// CircuitBreakerState represents the state of the circuit breaker.
type CircuitBreakerState int

const (
	CircuitClosed   CircuitBreakerState = iota
	CircuitOpen
	CircuitHalfOpen
)

type circuitBreaker struct {
	mu          sync.Mutex
	state       CircuitBreakerState
	failures    int
	lastFailure time.Time
	halfOpenReq int
	config      CircuitBreakerConfig
	clock       Clock
}

func newCircuitBreaker(config CircuitBreakerConfig, clock Clock) *circuitBreaker {
	if config.FailureThreshold <= 0 {
		config.FailureThreshold = 5
	}
	if config.ResetTimeout <= 0 {
		config.ResetTimeout = 30 * time.Second
	}
	if config.HalfOpenMax <= 0 {
		config.HalfOpenMax = 1
	}
	return &circuitBreaker{config: config, clock: clock}
}

// State returns the current circuit breaker state.
func (cb *circuitBreaker) State() CircuitBreakerState {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.currentState()
}

func (cb *circuitBreaker) currentState() CircuitBreakerState {
	if cb.state == CircuitOpen && cb.clock.Now().Sub(cb.lastFailure) >= cb.config.ResetTimeout {
		cb.state = CircuitHalfOpen
		cb.halfOpenReq = 0
	}
	return cb.state
}

func (cb *circuitBreaker) allow() error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.currentState() {
	case CircuitOpen:
		return ErrCircuitOpen
	case CircuitHalfOpen:
		if cb.halfOpenReq >= cb.config.HalfOpenMax {
			return ErrCircuitOpen
		}
		cb.halfOpenReq++
	}
	return nil
}

func (cb *circuitBreaker) recordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.failures = 0
	cb.state = CircuitClosed
}

func (cb *circuitBreaker) recordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.failures++
	cb.lastFailure = cb.clock.Now()
	if cb.failures >= cb.config.FailureThreshold {
		cb.state = CircuitOpen
	}
}

// RateLimiter controls request rate. Implement this interface to provide
// custom rate limiting. Wait should block until a token is available or
// the context is cancelled.
type RateLimiter interface {
	Wait(ctx context.Context) error
}

// tokenBucketLimiter implements RateLimiter with a simple token bucket.
type tokenBucketLimiter struct {
	mu       sync.Mutex
	tokens   float64
	maxBurst float64
	rate     float64 // tokens per second
	lastTime time.Time
	clock    Clock
}

// NewTokenBucketLimiter creates a rate limiter. rate is requests per second,
// burst is the maximum burst size.
func NewTokenBucketLimiter(rate float64, burst int) RateLimiter {
	return newTokenBucketLimiterWithClock(rate, burst, &realClock{})
}

func newTokenBucketLimiterWithClock(rate float64, burst int, clock Clock) *tokenBucketLimiter {
	return &tokenBucketLimiter{
		tokens:   float64(burst),
		maxBurst: float64(burst),
		rate:     rate,
		lastTime: clock.Now(),
		clock:    clock,
	}
}

func (l *tokenBucketLimiter) Wait(ctx context.Context) error {
	for {
		l.mu.Lock()
		now := l.clock.Now()
		elapsed := now.Sub(l.lastTime).Seconds()
		l.tokens += elapsed * l.rate
		if l.tokens > l.maxBurst {
			l.tokens = l.maxBurst
		}
		l.lastTime = now

		if l.tokens >= 1 {
			l.tokens--
			l.mu.Unlock()
			return nil
		}
		waitDur := time.Duration((1 - l.tokens) / l.rate * float64(time.Second))
		l.mu.Unlock()

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(waitDur):
		}
	}
}

// resiliencePolicy combines retry, circuit breaker, and rate limiter.
type resiliencePolicy struct {
	retry          *RetryPolicy
	circuitBreaker *circuitBreaker
	rateLimiter    RateLimiter
}

// execute wraps an operation with resilience: rate limit -> circuit breaker -> retry.
func (rp *resiliencePolicy) execute(ctx context.Context, op func(ctx context.Context) (*response, error)) (*response, error) {
	maxAttempts := 1
	if rp.retry != nil {
		maxAttempts = rp.retry.MaxAttempts
		if maxAttempts < 1 {
			maxAttempts = 1
		}
	}

	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 && rp.retry != nil {
			delay := rp.retry.delay(attempt - 1)

			// Respect Retry-After header if present.
			var apiErr *APIError
			if lastErr != nil {
				if asAPIErr, ok := lastErr.(*APIError); ok {
					apiErr = asAPIErr
				}
			}
			if apiErr != nil {
				if ra := apiErr.RetryAfter(); ra > delay {
					delay = ra
				}
			}

			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
		}

		if rp.rateLimiter != nil {
			if err := rp.rateLimiter.Wait(ctx); err != nil {
				return nil, err
			}
		}

		if rp.circuitBreaker != nil {
			if err := rp.circuitBreaker.allow(); err != nil {
				return nil, err
			}
		}

		resp, err := op(ctx)
		if err == nil {
			if rp.circuitBreaker != nil {
				rp.circuitBreaker.recordSuccess()
			}
			return resp, nil
		}

		lastErr = err

		if rp.circuitBreaker != nil {
			rp.circuitBreaker.recordFailure()
		}

		// Only retry if the error is retryable.
		apiErr, isAPIErr := err.(*APIError)
		if !isAPIErr || !apiErr.IsRetryable() {
			return nil, err
		}
	}
	return nil, lastErr
}
