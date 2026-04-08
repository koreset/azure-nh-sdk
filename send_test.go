package azurenh

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestSend_Broadcast(t *testing.T) {
	c := newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != "POST" {
			t.Errorf("method = %q, want POST", req.Method)
		}
		if !strings.Contains(req.URL.Path, "/messages") {
			t.Errorf("path = %q, want /messages", req.URL.Path)
		}
		if req.Header.Get("ServiceBusNotification-Format") != "apple" {
			t.Error("expected apple format header")
		}
		return mockResponse(201, "", map[string]string{
			"Location":                        "https://ns.servicebus.windows.net/hub/messages/12345?api-version=2016-07",
			"x-ms-correlation-request-id":     "corr-123",
		}), nil
	})

	n, _ := NewAPNSNotification().Alert("Hello", "World").Build()
	result, err := c.Send(ctx(), n)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.NotificationID != "12345" {
		t.Errorf("notificationId = %q, want 12345", result.NotificationID)
	}
	if result.CorrelationID != "corr-123" {
		t.Errorf("correlationId = %q, want corr-123", result.CorrelationID)
	}
}

func TestSend_WithTagExpression(t *testing.T) {
	c := newTestClient(func(req *http.Request) (*http.Response, error) {
		tagHeader := req.Header.Get("ServiceBusNotification-Tags")
		if tagHeader != "user:123 || premium" {
			t.Errorf("tag header = %q, want %q", tagHeader, "user:123 || premium")
		}
		return mockResponse(201, "", nil), nil
	})

	n, _ := NewFCMV1Notification().Title("Hi").Body("There").Build()
	_, err := c.Send(ctx(), n, WithTagExpression("user:123 || premium"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSend_WithTestSend(t *testing.T) {
	c := newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.URL.Query().Get("test") != "true" {
			t.Error("expected test=true query param")
		}
		return mockResponse(201, "", nil), nil
	})

	n, _ := NewAPNSNotification().Alert("Test", "Send").Build()
	_, err := c.Send(ctx(), n, WithTestSend())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSend_APNSHeaders(t *testing.T) {
	c := newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Header.Get("apns-push-type") != "alert" {
			t.Errorf("apns-push-type = %q, want alert", req.Header.Get("apns-push-type"))
		}
		if req.Header.Get("apns-priority") != "10" {
			t.Errorf("apns-priority = %q, want 10", req.Header.Get("apns-priority"))
		}
		return mockResponse(201, "", nil), nil
	})

	n, _ := NewAPNSNotification().Alert("Title", "Body").Build()
	_, err := c.Send(ctx(), n)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSend_APNSBackgroundHeaders(t *testing.T) {
	c := newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Header.Get("apns-push-type") != "background" {
			t.Errorf("apns-push-type = %q, want background", req.Header.Get("apns-push-type"))
		}
		if req.Header.Get("apns-priority") != "5" {
			t.Errorf("apns-priority = %q, want 5", req.Header.Get("apns-priority"))
		}
		return mockResponse(201, "", nil), nil
	})

	n, _ := NewAPNSNotification().ContentAvailable().Build()
	_, err := c.Send(ctx(), n)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSendDirect(t *testing.T) {
	c := newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Header.Get("ServiceBusNotification-DeviceHandle") != "device-token-123" {
			t.Error("expected device handle header")
		}
		if !strings.Contains(req.URL.RawQuery, "direct") {
			t.Error("expected direct query param")
		}
		return mockResponse(201, "", nil), nil
	})

	n, _ := NewAPNSNotification().Alert("Hello", "Direct").Build()
	_, err := c.SendDirect(ctx(), n, "device-token-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSendDirectBatch(t *testing.T) {
	c := newTestClient(func(req *http.Request) (*http.Response, error) {
		if !strings.Contains(req.URL.Path, "$batch") {
			t.Error("expected $batch in path")
		}
		return mockResponse(201, "", nil), nil
	})

	n, _ := NewAPNSNotification().Alert("Batch", "Send").Build()
	_, err := c.SendDirectBatch(ctx(), n, []string{"token1", "token2"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSendDirectBatch_TooLarge(t *testing.T) {
	c := newTestClient(nil)
	n, _ := NewAPNSNotification().Alert("Batch", "Send").Build()

	handles := make([]string, 1001)
	for i := range handles {
		handles[i] = "token"
	}
	_, err := c.SendDirectBatch(ctx(), n, handles)
	if err != ErrBatchTooLarge {
		t.Errorf("expected ErrBatchTooLarge, got %v", err)
	}
}

func TestSchedule(t *testing.T) {
	now := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)
	clock := &fixedClock{t: now}
	c := newTestClient(func(req *http.Request) (*http.Response, error) {
		if !strings.Contains(req.URL.Path, "schedulednotifications") {
			t.Error("expected schedulednotifications in path")
		}
		schedTime := req.Header.Get("ServiceBusNotification-ScheduleTime")
		if schedTime == "" {
			t.Error("expected schedule time header")
		}
		return mockResponse(201, "", nil), nil
	}, WithClock(clock))

	n, _ := NewAPNSNotification().Alert("Scheduled", "Notification").Build()
	future := now.Add(1 * time.Hour)
	_, err := c.Schedule(ctx(), n, future)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSchedule_InPast(t *testing.T) {
	now := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)
	clock := &fixedClock{t: now}
	c := newTestClient(nil, WithClock(clock))
	n, _ := NewAPNSNotification().Alert("Past", "Notification").Build()
	past := now.Add(-1 * time.Hour)
	_, err := c.Schedule(ctx(), n, past)
	if err != ErrScheduleInPast {
		t.Errorf("expected ErrScheduleInPast, got %v", err)
	}
}

func TestCancelScheduledNotification(t *testing.T) {
	c := newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != "DELETE" {
			t.Errorf("method = %q, want DELETE", req.Method)
		}
		if !strings.Contains(req.URL.Path, "schedulednotifications/12345") {
			t.Error("expected notification ID in path")
		}
		return mockResponse(200, "", nil), nil
	})

	err := c.CancelScheduledNotification(ctx(), "12345")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSend_NilNotification(t *testing.T) {
	c := newTestClient(nil)
	_, err := c.Send(ctx(), nil)
	if err == nil {
		t.Error("expected error for nil notification")
	}
}
