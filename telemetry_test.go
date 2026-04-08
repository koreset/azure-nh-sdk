package azurenh

import (
	"net/http"
	"testing"
)

func TestGetNotificationDetails(t *testing.T) {
	responseXML := `<?xml version="1.0" encoding="utf-8"?>
<NotificationDetails xmlns="http://schemas.microsoft.com/netservices/2010/10/servicebus/connect">
  <NotificationId>12345</NotificationId>
  <State>Completed</State>
  <EnqueueTime>2025-01-01T00:00:00Z</EnqueueTime>
  <StartTime>2025-01-01T00:00:01Z</StartTime>
  <EndTime>2025-01-01T00:00:02Z</EndTime>
  <NotificationBody>{"aps":{"alert":"test"}}</NotificationBody>
  <TargetPlatforms>apple</TargetPlatforms>
  <ApnsOutcomeCounts>
    <Outcome>
      <Name>Success</Name>
      <Count>100</Count>
    </Outcome>
    <Outcome>
      <Name>WrongToken</Name>
      <Count>5</Count>
    </Outcome>
  </ApnsOutcomeCounts>
</NotificationDetails>`

	c := newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != "GET" {
			t.Errorf("method = %q, want GET", req.Method)
		}
		return mockResponse(200, responseXML, nil), nil
	})

	details, err := c.GetNotificationDetails(ctx(), "12345")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if details.NotificationID != "12345" {
		t.Errorf("notificationId = %q, want 12345", details.NotificationID)
	}
	if details.State != StateCompleted {
		t.Errorf("state = %q, want Completed", details.State)
	}
	if details.TargetPlatforms != "apple" {
		t.Errorf("targetPlatforms = %q, want apple", details.TargetPlatforms)
	}
	if len(details.APNSOutcomes) != 2 {
		t.Fatalf("expected 2 APNS outcomes, got %d", len(details.APNSOutcomes))
	}
	if details.APNSOutcomes[0].Name != "Success" || details.APNSOutcomes[0].Count != 100 {
		t.Errorf("first outcome = %+v, want Success:100", details.APNSOutcomes[0])
	}
}

func TestGetNotificationDetails_EmptyID(t *testing.T) {
	c := newTestClient(nil)
	_, err := c.GetNotificationDetails(ctx(), "")
	if err == nil {
		t.Error("expected error for empty notification ID")
	}
}

func TestParseNotificationIDFromLocation(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"https://ns.servicebus.windows.net/hub/messages/12345?api-version=2016-07", "12345"},
		{"https://ns.servicebus.windows.net/hub/messages/abc-def", "abc-def"},
		{"", ""},
		{"no-slash", ""},
	}
	for _, tt := range tests {
		got := parseNotificationIDFromLocation(tt.input)
		if got != tt.want {
			t.Errorf("parseNotificationIDFromLocation(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
