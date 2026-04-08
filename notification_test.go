package azurenh

import (
	"encoding/json"
	"testing"
)

func TestNewRawNotification(t *testing.T) {
	n, err := NewRawNotification(FormatApple, []byte(`{"aps":{}}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n.Format != FormatApple {
		t.Errorf("format = %q, want %q", n.Format, FormatApple)
	}

	_, err = NewRawNotification("invalid", []byte("test"))
	if err == nil {
		t.Error("expected error for invalid format")
	}

	_, err = NewRawNotification(FormatApple, nil)
	if err == nil {
		t.Error("expected error for empty payload")
	}
}

func TestNewTemplateNotification(t *testing.T) {
	n, err := NewTemplateNotification(map[string]string{"title": "Hello", "body": "World"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n.Format != FormatTemplate {
		t.Errorf("format = %q, want %q", n.Format, FormatTemplate)
	}

	var props map[string]string
	if err := json.Unmarshal(n.Payload, &props); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if props["title"] != "Hello" {
		t.Errorf("title = %q, want %q", props["title"], "Hello")
	}

	_, err = NewTemplateNotification(nil)
	if err == nil {
		t.Error("expected error for nil properties")
	}
}

func TestAPNSBuilder_Basic(t *testing.T) {
	n, err := NewAPNSNotification().
		Alert("Title", "Body").
		Badge(5).
		Sound("default").
		Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(n.Payload, &payload); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	aps := payload["aps"].(map[string]any)
	alert := aps["alert"].(map[string]any)
	if alert["title"] != "Title" {
		t.Errorf("title = %v, want %q", alert["title"], "Title")
	}
	if alert["body"] != "Body" {
		t.Errorf("body = %v, want %q", alert["body"], "Body")
	}
	if aps["badge"].(float64) != 5 {
		t.Errorf("badge = %v, want 5", aps["badge"])
	}
	if aps["sound"] != "default" {
		t.Errorf("sound = %v, want %q", aps["sound"], "default")
	}
}

func TestAPNSBuilder_BackgroundNotification(t *testing.T) {
	n, err := NewAPNSNotification().
		ContentAvailable().
		Custom("data-key", "data-value").
		Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var payload map[string]any
	json.Unmarshal(n.Payload, &payload)
	aps := payload["aps"].(map[string]any)
	if aps["content-available"].(float64) != 1 {
		t.Error("expected content-available: 1")
	}
	if payload["data-key"] != "data-value" {
		t.Error("expected custom data")
	}
}

func TestAPNSBuilder_ValidationError(t *testing.T) {
	_, err := NewAPNSNotification().Build()
	if err == nil {
		t.Error("expected validation error when no alert or content-available")
	}
}

func TestAPNSBuilder_MutableContentAndCategory(t *testing.T) {
	n, err := NewAPNSNotification().
		Alert("Test", "Body").
		MutableContent().
		Category("MESSAGE").
		ThreadID("thread-1").
		Subtitle("Sub").
		Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var payload map[string]any
	json.Unmarshal(n.Payload, &payload)
	aps := payload["aps"].(map[string]any)
	if aps["mutable-content"].(float64) != 1 {
		t.Error("expected mutable-content: 1")
	}
	if aps["category"] != "MESSAGE" {
		t.Errorf("category = %v, want MESSAGE", aps["category"])
	}
	if aps["thread-id"] != "thread-1" {
		t.Errorf("thread-id = %v, want thread-1", aps["thread-id"])
	}
	alert := aps["alert"].(map[string]any)
	if alert["subtitle"] != "Sub" {
		t.Errorf("subtitle = %v, want Sub", alert["subtitle"])
	}
}

func TestFCMV1Builder_Basic(t *testing.T) {
	n, err := NewFCMV1Notification().
		Title("Hello").
		Body("World").
		Image("https://example.com/img.png").
		Data("key1", "val1").
		AndroidPriority("high").
		AndroidTTL("3600s").
		Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var payload map[string]any
	json.Unmarshal(n.Payload, &payload)
	msg := payload["message"].(map[string]any)
	notif := msg["notification"].(map[string]any)
	if notif["title"] != "Hello" {
		t.Errorf("title = %v, want Hello", notif["title"])
	}
	if notif["body"] != "World" {
		t.Errorf("body = %v, want World", notif["body"])
	}
	if notif["image"] != "https://example.com/img.png" {
		t.Error("expected image URL")
	}

	data := msg["data"].(map[string]any)
	if data["key1"] != "val1" {
		t.Error("expected data key1=val1")
	}

	android := msg["android"].(map[string]any)
	if android["priority"] != "high" {
		t.Error("expected high priority")
	}
}

func TestFCMV1Builder_DataOnly(t *testing.T) {
	n, err := NewFCMV1Notification().
		Data("key", "value").
		Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var payload map[string]any
	json.Unmarshal(n.Payload, &payload)
	msg := payload["message"].(map[string]any)
	if _, ok := msg["notification"]; ok {
		t.Error("should not have notification block for data-only")
	}
}

func TestFCMV1Builder_ValidationError(t *testing.T) {
	_, err := NewFCMV1Notification().Build()
	if err == nil {
		t.Error("expected validation error")
	}
}

func TestWNSBuilder(t *testing.T) {
	xml := `<toast><visual><binding template="ToastText01"><text id="1">Hello</text></binding></visual></toast>`
	n, err := NewWNSNotification().RawXML(xml).Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n.Format != FormatWindows {
		t.Errorf("format = %q, want windows", n.Format)
	}

	_, err = NewWNSNotification().Build()
	if err == nil {
		t.Error("expected validation error for empty WNS")
	}
}

func TestADMBuilder(t *testing.T) {
	n, err := NewADMNotification().
		Data("title", "Hello").
		ConsolidationKey("group1").
		Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var payload map[string]any
	json.Unmarshal(n.Payload, &payload)
	if payload["consolidationKey"] != "group1" {
		t.Error("expected consolidationKey")
	}

	_, err = NewADMNotification().Build()
	if err == nil {
		t.Error("expected validation error for empty ADM")
	}
}

func TestBaiduBuilder(t *testing.T) {
	n, err := NewBaiduNotification().
		Title("Hello").
		Description("World").
		Custom("key", "val").
		Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var payload map[string]any
	json.Unmarshal(n.Payload, &payload)
	if payload["title"] != "Hello" {
		t.Errorf("title = %v, want Hello", payload["title"])
	}
	cc := payload["custom_content"].(map[string]any)
	if cc["key"] != "val" {
		t.Error("expected custom_content key=val")
	}

	_, err = NewBaiduNotification().Build()
	if err == nil {
		t.Error("expected validation error for empty Baidu")
	}
}
