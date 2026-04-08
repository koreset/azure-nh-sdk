package azurenh

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestParseConnectionString_Valid(t *testing.T) {
	cs := "Endpoint=sb://test-ns.servicebus.windows.net/;SharedAccessKeyName=DefaultFullSharedAccessSignature;SharedAccessKey=dGVzdGtleQ=="
	endpoint, keyName, keyValue, err := parseConnectionString(cs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if endpoint.Host != "test-ns.servicebus.windows.net" {
		t.Errorf("endpoint host = %q, want %q", endpoint.Host, "test-ns.servicebus.windows.net")
	}
	if keyName != "DefaultFullSharedAccessSignature" {
		t.Errorf("keyName = %q, want %q", keyName, "DefaultFullSharedAccessSignature")
	}
	if keyValue != "dGVzdGtleQ==" {
		t.Errorf("keyValue = %q, want %q", keyValue, "dGVzdGtleQ==")
	}
}

func TestParseConnectionString_MissingEndpoint(t *testing.T) {
	cs := "SharedAccessKeyName=name;SharedAccessKey=key"
	_, _, _, err := parseConnectionString(cs)
	if !errors.Is(err, ErrInvalidConnectionString) {
		t.Errorf("expected ErrInvalidConnectionString, got %v", err)
	}
}

func TestParseConnectionString_MissingKeyName(t *testing.T) {
	cs := "Endpoint=sb://test.servicebus.windows.net/;SharedAccessKey=key"
	_, _, _, err := parseConnectionString(cs)
	if !errors.Is(err, ErrInvalidConnectionString) {
		t.Errorf("expected ErrInvalidConnectionString, got %v", err)
	}
}

func TestParseConnectionString_MissingKey(t *testing.T) {
	cs := "Endpoint=sb://test.servicebus.windows.net/;SharedAccessKeyName=name"
	_, _, _, err := parseConnectionString(cs)
	if !errors.Is(err, ErrInvalidConnectionString) {
		t.Errorf("expected ErrInvalidConnectionString, got %v", err)
	}
}

type fixedClock struct {
	t time.Time
}

func (fc *fixedClock) Now() time.Time { return fc.t }

func TestGenerateSASToken(t *testing.T) {
	c := &Client{
		hubURL:      mustParseURL("https://test-ns.servicebus.windows.net/myhub"),
		sasKeyName:  "DefaultFullSharedAccessSignature",
		sasKeyValue: "dGVzdGtleQ==",
		clock:       &fixedClock{t: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)},
	}

	token := c.generateSASToken()

	if !strings.HasPrefix(token, "SharedAccessSignature ") {
		t.Errorf("token should start with 'SharedAccessSignature ', got %q", token)
	}
	if !strings.Contains(token, "sr=") {
		t.Error("token should contain 'sr='")
	}
	if !strings.Contains(token, "sig=") {
		t.Error("token should contain 'sig='")
	}
	if !strings.Contains(token, "se=") {
		t.Error("token should contain 'se='")
	}
	if !strings.Contains(token, "skn=DefaultFullSharedAccessSignature") {
		t.Error("token should contain key name")
	}
}
