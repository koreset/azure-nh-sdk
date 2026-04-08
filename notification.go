package azurenh

import (
	"encoding/json"
	"fmt"
)

// Notification represents a push notification to send.
type Notification struct {
	Format  NotificationFormat
	Payload []byte
}

// NewRawNotification creates a notification from raw bytes and format.
// Use this when you have a pre-built payload.
func NewRawNotification(format NotificationFormat, payload []byte) (*Notification, error) {
	if !format.IsValid() {
		return nil, &ValidationError{Field: "format", Message: fmt.Sprintf("invalid format: %q", format)}
	}
	if len(payload) == 0 {
		return nil, &ValidationError{Field: "payload", Message: "payload cannot be empty"}
	}
	return &Notification{Format: format, Payload: payload}, nil
}

// NewTemplateNotification creates a cross-platform template notification
// with the given property substitutions.
func NewTemplateNotification(properties map[string]string) (*Notification, error) {
	if len(properties) == 0 {
		return nil, &ValidationError{Field: "properties", Message: "template properties cannot be empty"}
	}
	payload, err := json.Marshal(properties)
	if err != nil {
		return nil, fmt.Errorf("azurenh: failed to marshal template properties: %w", err)
	}
	return &Notification{Format: FormatTemplate, Payload: payload}, nil
}

// --- APNS Builder ---

// APNSBuilder builds APNS notification payloads for iOS.
type APNSBuilder struct {
	alert          *apnsAlert
	badge          *int
	sound          string
	contentAvail   bool
	mutableContent bool
	category       string
	threadID       string
	customData     map[string]any
}

type apnsAlert struct {
	Title        string   `json:"title,omitempty"`
	Subtitle     string   `json:"subtitle,omitempty"`
	Body         string   `json:"body,omitempty"`
	LaunchImage  string   `json:"launch-image,omitempty"`
	TitleLocKey  string   `json:"title-loc-key,omitempty"`
	TitleLocArgs []string `json:"title-loc-args,omitempty"`
	LocKey       string   `json:"loc-key,omitempty"`
	LocArgs      []string `json:"loc-args,omitempty"`
}

// NewAPNSNotification starts building an APNS notification.
func NewAPNSNotification() *APNSBuilder {
	return &APNSBuilder{}
}

// Alert sets the notification title and body.
func (b *APNSBuilder) Alert(title, body string) *APNSBuilder {
	if b.alert == nil {
		b.alert = &apnsAlert{}
	}
	b.alert.Title = title
	b.alert.Body = body
	return b
}

// Subtitle sets the notification subtitle.
func (b *APNSBuilder) Subtitle(subtitle string) *APNSBuilder {
	if b.alert == nil {
		b.alert = &apnsAlert{}
	}
	b.alert.Subtitle = subtitle
	return b
}

// Badge sets the badge count on the app icon.
func (b *APNSBuilder) Badge(n int) *APNSBuilder {
	b.badge = &n
	return b
}

// Sound sets the notification sound name.
func (b *APNSBuilder) Sound(sound string) *APNSBuilder {
	b.sound = sound
	return b
}

// ContentAvailable marks this as a background/silent notification (content-available: 1).
func (b *APNSBuilder) ContentAvailable() *APNSBuilder {
	b.contentAvail = true
	return b
}

// MutableContent enables Notification Service Extension processing (mutable-content: 1).
func (b *APNSBuilder) MutableContent() *APNSBuilder {
	b.mutableContent = true
	return b
}

// Category sets the notification category for actionable notifications.
func (b *APNSBuilder) Category(category string) *APNSBuilder {
	b.category = category
	return b
}

// ThreadID sets the thread identifier for notification grouping.
func (b *APNSBuilder) ThreadID(id string) *APNSBuilder {
	b.threadID = id
	return b
}

// Custom adds a custom key-value pair to the payload root.
func (b *APNSBuilder) Custom(key string, value any) *APNSBuilder {
	if b.customData == nil {
		b.customData = make(map[string]any)
	}
	b.customData[key] = value
	return b
}

// Build validates and serializes the APNS notification payload.
func (b *APNSBuilder) Build() (*Notification, error) {
	if b.alert == nil && !b.contentAvail {
		return nil, &ValidationError{Field: "alert", Message: "either alert or content-available is required"}
	}

	aps := make(map[string]any)
	if b.alert != nil {
		aps["alert"] = b.alert
	}
	if b.badge != nil {
		aps["badge"] = *b.badge
	}
	if b.sound != "" {
		aps["sound"] = b.sound
	}
	if b.contentAvail {
		aps["content-available"] = 1
	}
	if b.mutableContent {
		aps["mutable-content"] = 1
	}
	if b.category != "" {
		aps["category"] = b.category
	}
	if b.threadID != "" {
		aps["thread-id"] = b.threadID
	}

	payload := map[string]any{"aps": aps}
	for k, v := range b.customData {
		payload[k] = v
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("azurenh: failed to marshal APNS payload: %w", err)
	}
	return &Notification{Format: FormatApple, Payload: data}, nil
}

// --- FCM v1 Builder ---

// FCMV1Builder builds FCM v1 notification payloads for Android.
type FCMV1Builder struct {
	title    string
	body     string
	image    string
	data     map[string]string
	priority string
	ttl      string
}

// NewFCMV1Notification starts building an FCM v1 notification.
func NewFCMV1Notification() *FCMV1Builder {
	return &FCMV1Builder{}
}

// Title sets the notification title.
func (b *FCMV1Builder) Title(title string) *FCMV1Builder {
	b.title = title
	return b
}

// Body sets the notification body.
func (b *FCMV1Builder) Body(body string) *FCMV1Builder {
	b.body = body
	return b
}

// Image sets the notification image URL.
func (b *FCMV1Builder) Image(url string) *FCMV1Builder {
	b.image = url
	return b
}

// Data adds a custom data key-value pair.
func (b *FCMV1Builder) Data(key, value string) *FCMV1Builder {
	if b.data == nil {
		b.data = make(map[string]string)
	}
	b.data[key] = value
	return b
}

// AndroidPriority sets the Android message priority ("normal" or "high").
func (b *FCMV1Builder) AndroidPriority(priority string) *FCMV1Builder {
	b.priority = priority
	return b
}

// AndroidTTL sets the Android message time-to-live (e.g., "3600s").
func (b *FCMV1Builder) AndroidTTL(ttl string) *FCMV1Builder {
	b.ttl = ttl
	return b
}

// Build validates and serializes the FCM v1 notification payload.
func (b *FCMV1Builder) Build() (*Notification, error) {
	if b.title == "" && b.body == "" && len(b.data) == 0 {
		return nil, &ValidationError{Field: "notification", Message: "at least title/body or data is required"}
	}

	msg := make(map[string]any)

	if b.title != "" || b.body != "" || b.image != "" {
		notification := make(map[string]string)
		if b.title != "" {
			notification["title"] = b.title
		}
		if b.body != "" {
			notification["body"] = b.body
		}
		if b.image != "" {
			notification["image"] = b.image
		}
		msg["notification"] = notification
	}

	if len(b.data) > 0 {
		msg["data"] = b.data
	}

	if b.priority != "" || b.ttl != "" {
		android := make(map[string]string)
		if b.priority != "" {
			android["priority"] = b.priority
		}
		if b.ttl != "" {
			android["ttl"] = b.ttl
		}
		msg["android"] = android
	}

	wrapper := map[string]any{"message": msg}
	data, err := json.Marshal(wrapper)
	if err != nil {
		return nil, fmt.Errorf("azurenh: failed to marshal FCM payload: %w", err)
	}
	return &Notification{Format: FormatFCMV1, Payload: data}, nil
}

// --- WNS Builder ---

// WNSBuilder builds Windows Notification Service payloads.
type WNSBuilder struct {
	payload string
}

// NewWNSNotification starts building a WNS notification.
func NewWNSNotification() *WNSBuilder {
	return &WNSBuilder{}
}

// RawXML sets the raw XML payload for WNS (toast, tile, badge, or raw).
func (b *WNSBuilder) RawXML(xml string) *WNSBuilder {
	b.payload = xml
	return b
}

// Build validates and returns the WNS notification.
func (b *WNSBuilder) Build() (*Notification, error) {
	if b.payload == "" {
		return nil, &ValidationError{Field: "payload", Message: "WNS XML payload is required"}
	}
	return &Notification{Format: FormatWindows, Payload: []byte(b.payload)}, nil
}

// --- ADM Builder ---

// ADMBuilder builds Amazon Device Messaging payloads.
type ADMBuilder struct {
	data            map[string]string
	consolidationKey string
	expiresAfter    int
	md5             string
}

// NewADMNotification starts building an ADM notification.
func NewADMNotification() *ADMBuilder {
	return &ADMBuilder{}
}

// Data adds a custom data key-value pair.
func (b *ADMBuilder) Data(key, value string) *ADMBuilder {
	if b.data == nil {
		b.data = make(map[string]string)
	}
	b.data[key] = value
	return b
}

// ConsolidationKey sets the consolidation key for message grouping.
func (b *ADMBuilder) ConsolidationKey(key string) *ADMBuilder {
	b.consolidationKey = key
	return b
}

// Build validates and serializes the ADM notification payload.
func (b *ADMBuilder) Build() (*Notification, error) {
	if len(b.data) == 0 {
		return nil, &ValidationError{Field: "data", Message: "ADM requires at least one data field"}
	}

	payload := map[string]any{"data": b.data}
	if b.consolidationKey != "" {
		payload["consolidationKey"] = b.consolidationKey
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("azurenh: failed to marshal ADM payload: %w", err)
	}
	return &Notification{Format: FormatADM, Payload: data}, nil
}

// --- Baidu Builder ---

// BaiduBuilder builds Baidu Cloud Push payloads.
type BaiduBuilder struct {
	title       string
	description string
	customData  map[string]string
}

// NewBaiduNotification starts building a Baidu notification.
func NewBaiduNotification() *BaiduBuilder {
	return &BaiduBuilder{}
}

// Title sets the notification title.
func (b *BaiduBuilder) Title(title string) *BaiduBuilder {
	b.title = title
	return b
}

// Description sets the notification description.
func (b *BaiduBuilder) Description(desc string) *BaiduBuilder {
	b.description = desc
	return b
}

// Custom adds a custom key-value pair.
func (b *BaiduBuilder) Custom(key, value string) *BaiduBuilder {
	if b.customData == nil {
		b.customData = make(map[string]string)
	}
	b.customData[key] = value
	return b
}

// Build validates and serializes the Baidu notification payload.
func (b *BaiduBuilder) Build() (*Notification, error) {
	if b.title == "" && b.description == "" {
		return nil, &ValidationError{Field: "title", Message: "Baidu requires title or description"}
	}

	payload := make(map[string]any)
	if b.title != "" {
		payload["title"] = b.title
	}
	if b.description != "" {
		payload["description"] = b.description
	}
	if len(b.customData) > 0 {
		payload["custom_content"] = b.customData
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("azurenh: failed to marshal Baidu payload: %w", err)
	}
	return &Notification{Format: FormatBaidu, Payload: data}, nil
}
