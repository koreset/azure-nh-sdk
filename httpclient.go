package azurenh

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// HTTPDoer abstracts HTTP request execution. The standard *http.Client
// satisfies this interface. Implement it for testing with mock responses.
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// response wraps an HTTP response with parsed body and headers.
type response struct {
	statusCode int
	headers    http.Header
	body       []byte
}

// exec builds and executes an authenticated HTTP request. It injects the SAS
// token, delegates to the resilience layer (if configured), reads the response
// body, and classifies errors.
func (c *Client) exec(ctx context.Context, method string, reqURL *url.URL, headers map[string]string, body io.Reader) (*response, error) {
	op := func(ctx context.Context) (*response, error) {
		req, err := http.NewRequestWithContext(ctx, method, reqURL.String(), body)
		if err != nil {
			return nil, fmt.Errorf("azurenh: failed to create request: %w", err)
		}

		req.Header.Set("Authorization", c.generateSASToken())
		for k, v := range headers {
			req.Header.Set(k, v)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("azurenh: request failed: %w", err)
		}
		defer resp.Body.Close()

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("azurenh: failed to read response body: %w", err)
		}

		if apiErr := classifyHTTPResponse(resp.StatusCode, resp.Header, respBody); apiErr != nil {
			return nil, apiErr
		}

		return &response{
			statusCode: resp.StatusCode,
			headers:    resp.Header,
			body:       respBody,
		}, nil
	}

	if c.resilience != nil {
		return c.resilience.execute(ctx, op)
	}
	return op(ctx)
}
