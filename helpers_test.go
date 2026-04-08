package azurenh

import (
	"bytes"
	"io"
	"net/http"
	"net/url"
)

const testConnectionString = "Endpoint=sb://test-ns.servicebus.windows.net/;SharedAccessKeyName=DefaultFullSharedAccessSignature;SharedAccessKey=dGVzdGtleQ=="
const testHubName = "myhub"

type mockHTTPDoer struct {
	doFunc func(req *http.Request) (*http.Response, error)
}

func (m *mockHTTPDoer) Do(req *http.Request) (*http.Response, error) {
	return m.doFunc(req)
}

func newTestClient(doFunc func(*http.Request) (*http.Response, error), opts ...Option) *Client {
	allOpts := append([]Option{
		WithHTTPClient(&mockHTTPDoer{doFunc: doFunc}),
		WithClock(&fixedClock{}),
	}, opts...)
	c, _ := NewClient(testConnectionString, testHubName, allOpts...)
	return c
}

func mockResponse(statusCode int, body string, headers map[string]string) *http.Response {
	h := http.Header{}
	for k, v := range headers {
		h.Set(k, v)
	}
	return &http.Response{
		StatusCode: statusCode,
		Header:     h,
		Body:       io.NopCloser(bytes.NewBufferString(body)),
	}
}

func mustParseURL(raw string) *url.URL {
	u, err := url.Parse(raw)
	if err != nil {
		panic(err)
	}
	return u
}
