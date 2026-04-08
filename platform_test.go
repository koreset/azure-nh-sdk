package azurenh

import "testing"

func TestPlatform_IsValid(t *testing.T) {
	valid := []Platform{PlatformAPNS, PlatformFCMV1, PlatformWNS, PlatformADM, PlatformBaidu}
	for _, p := range valid {
		if !p.IsValid() {
			t.Errorf("expected platform %q to be valid", p)
		}
	}

	invalid := []Platform{"gcm", "ios", "", "unknown"}
	for _, p := range invalid {
		if p.IsValid() {
			t.Errorf("expected platform %q to be invalid", p)
		}
	}
}

func TestNotificationFormat_IsValid(t *testing.T) {
	valid := []NotificationFormat{FormatApple, FormatFCMV1, FormatWindows, FormatADM, FormatBaidu, FormatTemplate}
	for _, f := range valid {
		if !f.IsValid() {
			t.Errorf("expected format %q to be valid", f)
		}
	}

	if NotificationFormat("unknown").IsValid() {
		t.Error("expected unknown format to be invalid")
	}
}

func TestNotificationFormat_ContentType(t *testing.T) {
	tests := []struct {
		format   NotificationFormat
		expected string
	}{
		{FormatApple, "application/json"},
		{FormatFCMV1, "application/json"},
		{FormatWindows, "application/xml"},
		{FormatADM, "application/json"},
		{FormatBaidu, "application/json"},
		{FormatTemplate, "application/json"},
	}
	for _, tt := range tests {
		if got := tt.format.ContentType(); got != tt.expected {
			t.Errorf("ContentType(%q) = %q, want %q", tt.format, got, tt.expected)
		}
	}
}

func TestNotificationFormat_ServiceBusHeader(t *testing.T) {
	if got := FormatApple.ServiceBusHeader(); got != "apple" {
		t.Errorf("ServiceBusHeader() = %q, want %q", got, "apple")
	}
}
