package azurenh

import (
	"errors"
	"testing"
)

func TestNewClient_Valid(t *testing.T) {
	c, err := NewClient(testConnectionString, testHubName)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.hubURL.Scheme != "https" {
		t.Errorf("scheme = %q, want %q", c.hubURL.Scheme, "https")
	}
	if c.hubURL.Path != testHubName {
		t.Errorf("path = %q, want %q", c.hubURL.Path, testHubName)
	}
	if c.apiVersion != defaultAPIVersion {
		t.Errorf("apiVersion = %q, want %q", c.apiVersion, defaultAPIVersion)
	}
}

func TestNewClient_InvalidConnectionString(t *testing.T) {
	_, err := NewClient("invalid", testHubName)
	if !errors.Is(err, ErrInvalidConnectionString) {
		t.Errorf("expected ErrInvalidConnectionString, got %v", err)
	}
}

func TestNewClient_EmptyHubName(t *testing.T) {
	_, err := NewClient(testConnectionString, "")
	if !errors.Is(err, ErrInvalidConnectionString) {
		t.Errorf("expected ErrInvalidConnectionString, got %v", err)
	}
}

func TestNewClient_WithAPIVersion(t *testing.T) {
	c, err := NewClient(testConnectionString, testHubName, WithAPIVersion("2020-06"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.apiVersion != "2020-06" {
		t.Errorf("apiVersion = %q, want %q", c.apiVersion, "2020-06")
	}
}

func TestBuildURL(t *testing.T) {
	c := newTestClient(nil)
	u := c.buildURL("messages")

	if u.Path != testHubName+"/messages" {
		t.Errorf("path = %q, want %q", u.Path, testHubName+"/messages")
	}
	if u.Query().Get("api-version") != defaultAPIVersion {
		t.Errorf("api-version = %q, want %q", u.Query().Get("api-version"), defaultAPIVersion)
	}
}
