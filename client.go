package azurenh

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Clock abstracts time for testability.
type Clock interface {
	Now() time.Time
}

type realClock struct{}

func (realClock) Now() time.Time { return time.Now() }

// Client is the Azure Notification Hubs client.
type Client struct {
	hubURL      *url.URL
	sasKeyName  string
	sasKeyValue string
	apiVersion  string
	httpClient  HTTPDoer
	resilience  *resiliencePolicy
	clock       Clock
}

// Option configures the Client.
type Option func(*Client)

// NewClient creates a new Azure Notification Hubs client.
//
// connectionString format:
//
//	Endpoint=sb://<namespace>.servicebus.windows.net/;SharedAccessKeyName=<name>;SharedAccessKey=<key>
func NewClient(connectionString, hubName string, opts ...Option) (*Client, error) {
	endpoint, keyName, keyValue, err := parseConnectionString(connectionString)
	if err != nil {
		return nil, err
	}

	hubName = strings.TrimSpace(hubName)
	if hubName == "" {
		return nil, fmt.Errorf("%w: hub name cannot be empty", ErrInvalidConnectionString)
	}

	// Convert sb:// scheme to https://.
	endpoint.Scheme = schemeHTTPS
	endpoint.Path = hubName

	c := &Client{
		hubURL:      endpoint,
		sasKeyName:  keyName,
		sasKeyValue: keyValue,
		apiVersion:  defaultAPIVersion,
		httpClient:  http.DefaultClient,
		clock:       realClock{},
	}

	for _, opt := range opts {
		opt(c)
	}

	return c, nil
}

// WithHTTPClient sets a custom HTTP client (for testing or custom transport).
func WithHTTPClient(doer HTTPDoer) Option {
	return func(c *Client) {
		c.httpClient = doer
	}
}

// WithRetryPolicy configures the retry policy.
func WithRetryPolicy(policy RetryPolicy) Option {
	return func(c *Client) {
		if c.resilience == nil {
			c.resilience = &resiliencePolicy{}
		}
		c.resilience.retry = &policy
	}
}

// WithCircuitBreaker enables the circuit breaker with the given config.
func WithCircuitBreaker(config CircuitBreakerConfig) Option {
	return func(c *Client) {
		if c.resilience == nil {
			c.resilience = &resiliencePolicy{}
		}
		c.resilience.circuitBreaker = newCircuitBreaker(config, c.clock)
	}
}

// WithRateLimiter sets a rate limiter for outgoing requests.
func WithRateLimiter(limiter RateLimiter) Option {
	return func(c *Client) {
		if c.resilience == nil {
			c.resilience = &resiliencePolicy{}
		}
		c.resilience.rateLimiter = limiter
	}
}

// WithAPIVersion overrides the API version (default "2016-07").
func WithAPIVersion(version string) Option {
	return func(c *Client) {
		c.apiVersion = version
	}
}

// WithClock overrides the clock (for testing).
func WithClock(clock Clock) Option {
	return func(c *Client) {
		c.clock = clock
	}
}

// buildURL constructs a full API URL for the given path segments.
func (c *Client) buildURL(pathSegments ...string) *url.URL {
	u := *c.hubURL
	segments := append([]string{u.Path}, pathSegments...)
	u.Path = strings.Join(segments, "/")
	q := u.Query()
	q.Set(apiVersionParam, c.apiVersion)
	u.RawQuery = q.Encode()
	return &u
}
