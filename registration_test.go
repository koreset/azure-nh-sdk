package azurenh

import (
	"net/http"
	"strings"
	"testing"
)

const appleRegistrationResponseXML = `<?xml version="1.0" encoding="utf-8"?>
<entry xmlns="http://www.w3.org/2005/Atom">
  <content type="application/xml">
    <AppleRegistrationDescription xmlns:i="http://www.w3.org/2001/XMLSchema-instance" xmlns="http://schemas.microsoft.com/netservices/2010/10/servicebus/connect">
      <RegistrationId>reg-123</RegistrationId>
      <ETag>1</ETag>
      <Tags>tag1,tag2</Tags>
      <DeviceToken>abc123</DeviceToken>
    </AppleRegistrationDescription>
  </content>
</entry>`

const fcmV1RegistrationResponseXML = `<?xml version="1.0" encoding="utf-8"?>
<entry xmlns="http://www.w3.org/2005/Atom">
  <content type="application/xml">
    <FcmV1RegistrationDescription xmlns:i="http://www.w3.org/2001/XMLSchema-instance" xmlns="http://schemas.microsoft.com/netservices/2010/10/servicebus/connect">
      <RegistrationId>reg-456</RegistrationId>
      <ETag>2</ETag>
      <Tags>android</Tags>
      <FcmV1RegistrationId>fcm-token-xyz</FcmV1RegistrationId>
    </FcmV1RegistrationDescription>
  </content>
</entry>`

func TestCreateRegistration_Apple(t *testing.T) {
	var capturedMethod string
	var capturedContentType string
	c := newTestClient(func(req *http.Request) (*http.Response, error) {
		capturedMethod = req.Method
		capturedContentType = req.Header.Get("Content-Type")
		return mockResponse(201, appleRegistrationResponseXML, nil), nil
	})

	reg, err := c.CreateRegistration(ctx(), Registration{
		Platform:    PlatformAPNS,
		DeviceToken: "abc123",
		Tags:        []string{"tag1", "tag2"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedMethod != "POST" {
		t.Errorf("method = %q, want POST", capturedMethod)
	}
	if !strings.Contains(capturedContentType, "atom+xml") {
		t.Errorf("content-type = %q, want atom+xml", capturedContentType)
	}
	if reg.RegistrationID != "reg-123" {
		t.Errorf("registrationId = %q, want reg-123", reg.RegistrationID)
	}
	if reg.Platform != PlatformAPNS {
		t.Errorf("platform = %q, want apns", reg.Platform)
	}
	if len(reg.Tags) != 2 || reg.Tags[0] != "tag1" {
		t.Errorf("tags = %v, want [tag1, tag2]", reg.Tags)
	}
}

func TestCreateRegistration_Validation(t *testing.T) {
	c := newTestClient(nil)

	_, err := c.CreateRegistration(ctx(), Registration{
		Platform: "invalid",
		DeviceToken: "token",
	})
	if err == nil {
		t.Error("expected validation error for invalid platform")
	}

	_, err = c.CreateRegistration(ctx(), Registration{
		Platform: PlatformAPNS,
	})
	if err == nil {
		t.Error("expected validation error for empty device token")
	}
}

func TestUpdateRegistration(t *testing.T) {
	c := newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != "PUT" {
			t.Errorf("method = %q, want PUT", req.Method)
		}
		if req.Header.Get("If-Match") != "etag-1" {
			t.Errorf("If-Match = %q, want etag-1", req.Header.Get("If-Match"))
		}
		return mockResponse(200, appleRegistrationResponseXML, nil), nil
	})

	_, err := c.UpdateRegistration(ctx(), Registration{
		RegistrationID: "reg-123",
		ETag:           "etag-1",
		Platform:       PlatformAPNS,
		DeviceToken:    "abc123",
		Tags:           []string{"tag1", "tag2"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetRegistration(t *testing.T) {
	c := newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != "GET" {
			t.Errorf("method = %q, want GET", req.Method)
		}
		return mockResponse(200, fcmV1RegistrationResponseXML, nil), nil
	})

	reg, err := c.GetRegistration(ctx(), "reg-456")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reg.RegistrationID != "reg-456" {
		t.Errorf("registrationId = %q, want reg-456", reg.RegistrationID)
	}
	if reg.Platform != PlatformFCMV1 {
		t.Errorf("platform = %q, want fcmv1", reg.Platform)
	}
	if reg.DeviceToken != "fcm-token-xyz" {
		t.Errorf("deviceToken = %q, want fcm-token-xyz", reg.DeviceToken)
	}
}

func TestListRegistrations(t *testing.T) {
	feedXML := `<?xml version="1.0" encoding="utf-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <entry>
    <content type="application/xml">
      <AppleRegistrationDescription xmlns:i="http://www.w3.org/2001/XMLSchema-instance" xmlns="http://schemas.microsoft.com/netservices/2010/10/servicebus/connect">
        <RegistrationId>reg-1</RegistrationId>
        <ETag>1</ETag>
        <Tags>tag1</Tags>
        <DeviceToken>token1</DeviceToken>
      </AppleRegistrationDescription>
    </content>
  </entry>
  <entry>
    <content type="application/xml">
      <FcmV1RegistrationDescription xmlns:i="http://www.w3.org/2001/XMLSchema-instance" xmlns="http://schemas.microsoft.com/netservices/2010/10/servicebus/connect">
        <RegistrationId>reg-2</RegistrationId>
        <ETag>2</ETag>
        <Tags>tag2</Tags>
        <FcmV1RegistrationId>token2</FcmV1RegistrationId>
      </FcmV1RegistrationDescription>
    </content>
  </entry>
</feed>`

	c := newTestClient(func(req *http.Request) (*http.Response, error) {
		headers := map[string]string{
			"X-MS-ContinuationToken": "next-token",
		}
		return mockResponse(200, feedXML, headers), nil
	})

	feed, err := c.ListRegistrations(ctx(), WithTag("tag1"), WithTop(10))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(feed.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(feed.Entries))
	}
	if feed.Entries[0].RegistrationID != "reg-1" {
		t.Errorf("first entry registrationId = %q, want reg-1", feed.Entries[0].RegistrationID)
	}
	if feed.ContinuationToken != "next-token" {
		t.Errorf("continuationToken = %q, want next-token", feed.ContinuationToken)
	}
}

func TestDeleteRegistration(t *testing.T) {
	c := newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != "DELETE" {
			t.Errorf("method = %q, want DELETE", req.Method)
		}
		if req.Header.Get("If-Match") != "*" {
			t.Errorf("If-Match = %q, want *", req.Header.Get("If-Match"))
		}
		return mockResponse(200, "", nil), nil
	})

	err := c.DeleteRegistration(ctx(), "reg-123", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildRegistrationXML(t *testing.T) {
	xml, err := buildRegistrationXML(Registration{
		Platform:    PlatformAPNS,
		DeviceToken: "token123",
		Tags:        []string{"tag1", "tag2"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(xml, "AppleRegistrationDescription") {
		t.Error("expected AppleRegistrationDescription")
	}
	if !strings.Contains(xml, "token123") {
		t.Error("expected device token in XML")
	}
	if !strings.Contains(xml, "tag1,tag2") {
		t.Error("expected tags in XML")
	}
}

func TestBuildRegistrationXML_Template(t *testing.T) {
	xml, err := buildRegistrationXML(Registration{
		Platform:    PlatformFCMV1,
		DeviceToken: "fcm-token",
		Tags:        []string{"user:1"},
		Template:    `{"message":{"notification":{"title":"$(title)"}}}`,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(xml, "FcmV1TemplateRegistrationDescription") {
		t.Error("expected FcmV1TemplateRegistrationDescription for template registration")
	}
	if !strings.Contains(xml, "BodyTemplate") {
		t.Error("expected BodyTemplate")
	}
}
