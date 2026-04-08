package azurenh

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"
)

// SendOption configures a send operation.
type SendOption func(*sendParams)

type sendParams struct {
	tagExpression string
	testSend      bool
	apnsExpiry    *time.Time
	apnsPriority  *int
	apnsPushType  string
}

// WithTagExpression sets the tag expression for targeting.
// Examples: "tag1", "tag1 || tag2", "(tag1 && tag2) || tag3"
func WithTagExpression(expr string) SendOption {
	return func(p *sendParams) {
		p.tagExpression = expr
	}
}

// WithTestSend enables test send mode (debug, limited to 10 devices).
func WithTestSend() SendOption {
	return func(p *sendParams) {
		p.testSend = true
	}
}

// WithAPNSExpiry sets the APNS expiration timestamp.
func WithAPNSExpiry(expiry time.Time) SendOption {
	return func(p *sendParams) {
		p.apnsExpiry = &expiry
	}
}

// WithAPNSPriority sets the APNS priority (5 for background, 10 for alert).
func WithAPNSPriority(priority int) SendOption {
	return func(p *sendParams) {
		p.apnsPriority = &priority
	}
}

// WithAPNSPushType sets the apns-push-type header ("alert", "background", "voip", etc.).
func WithAPNSPushType(pushType string) SendOption {
	return func(p *sendParams) {
		p.apnsPushType = pushType
	}
}

// Send broadcasts a notification to all devices matching the optional tag expression.
func (c *Client) Send(ctx context.Context, notification *Notification, opts ...SendOption) (*SendResult, error) {
	if notification == nil {
		return nil, &ValidationError{Field: "notification", Message: "notification is required"}
	}

	params := &sendParams{}
	for _, opt := range opts {
		opt(params)
	}

	u := c.buildURL("messages")
	if params.testSend {
		q := u.Query()
		q.Set("test", "true")
		u.RawQuery = q.Encode()
	}

	headers := c.buildSendHeaders(notification, params)

	resp, err := c.exec(ctx, "POST", u, headers, bytes.NewReader(notification.Payload))
	if err != nil {
		return nil, err
	}

	return extractSendResult(resp), nil
}

// SendDirect sends a notification directly to a specific device handle (push channel token).
func (c *Client) SendDirect(ctx context.Context, notification *Notification, deviceHandle string) (*SendResult, error) {
	if notification == nil {
		return nil, &ValidationError{Field: "notification", Message: "notification is required"}
	}
	if deviceHandle == "" {
		return nil, &ValidationError{Field: "deviceHandle", Message: "device handle is required"}
	}

	u := c.buildURL("messages")
	q := u.Query()
	q.Set(directParam, "")
	u.RawQuery = q.Encode()

	headers := c.buildSendHeaders(notification, &sendParams{})
	headers["ServiceBusNotification-DeviceHandle"] = deviceHandle

	resp, err := c.exec(ctx, "POST", u, headers, bytes.NewReader(notification.Payload))
	if err != nil {
		return nil, err
	}

	return extractSendResult(resp), nil
}

// SendDirectBatch sends a notification to multiple device handles (max 1000).
func (c *Client) SendDirectBatch(ctx context.Context, notification *Notification, deviceHandles []string) (*SendResult, error) {
	if notification == nil {
		return nil, &ValidationError{Field: "notification", Message: "notification is required"}
	}
	if len(deviceHandles) == 0 {
		return nil, &ValidationError{Field: "deviceHandles", Message: "at least one device handle is required"}
	}
	if len(deviceHandles) > maxBatchSize {
		return nil, ErrBatchTooLarge
	}

	u := c.buildURL("messages", "$batch")
	q := u.Query()
	q.Set(directParam, "")
	u.RawQuery = q.Encode()

	headers := c.buildSendHeaders(notification, &sendParams{})

	// Batch uses newline-separated device handles in the body alongside the notification.
	batchBody := strings.Join(deviceHandles, "\n") + "\n\n" + string(notification.Payload)

	resp, err := c.exec(ctx, "POST", u, headers, strings.NewReader(batchBody))
	if err != nil {
		return nil, err
	}

	return extractSendResult(resp), nil
}

// Schedule schedules a notification for future delivery.
func (c *Client) Schedule(ctx context.Context, notification *Notification, deliverAt time.Time, opts ...SendOption) (*SendResult, error) {
	if notification == nil {
		return nil, &ValidationError{Field: "notification", Message: "notification is required"}
	}
	if deliverAt.Before(c.clock.Now()) {
		return nil, ErrScheduleInPast
	}

	params := &sendParams{}
	for _, opt := range opts {
		opt(params)
	}

	u := c.buildURL("schedulednotifications")

	headers := c.buildSendHeaders(notification, params)
	headers["ServiceBusNotification-ScheduleTime"] = deliverAt.UTC().Format(time.RFC3339)

	resp, err := c.exec(ctx, "POST", u, headers, bytes.NewReader(notification.Payload))
	if err != nil {
		return nil, err
	}

	return extractSendResult(resp), nil
}

// CancelScheduledNotification cancels a previously scheduled notification.
func (c *Client) CancelScheduledNotification(ctx context.Context, notificationID string) error {
	if notificationID == "" {
		return &ValidationError{Field: "notificationId", Message: "notification ID is required"}
	}

	_, err := c.exec(ctx, "DELETE", c.buildURL("schedulednotifications", notificationID), nil, nil)
	return err
}

func (c *Client) buildSendHeaders(notification *Notification, params *sendParams) map[string]string {
	headers := map[string]string{
		"Content-Type":                     notification.Format.ContentType(),
		"ServiceBusNotification-Format":    notification.Format.ServiceBusHeader(),
	}

	if params.tagExpression != "" {
		headers["ServiceBusNotification-Tags"] = params.tagExpression
	}

	// APNS-specific headers.
	if notification.Format == FormatApple {
		pushType := params.apnsPushType
		if pushType == "" {
			pushType = inferAPNSPushType(notification.Payload)
		}
		headers["apns-push-type"] = pushType

		if params.apnsPriority != nil {
			headers["apns-priority"] = fmt.Sprintf("%d", *params.apnsPriority)
		} else if pushType == "background" {
			headers["apns-priority"] = "5"
		} else {
			headers["apns-priority"] = "10"
		}

		if params.apnsExpiry != nil {
			headers["apns-expiration"] = fmt.Sprintf("%d", params.apnsExpiry.Unix())
		}
	}

	return headers
}

// inferAPNSPushType checks if the payload is a background notification.
func inferAPNSPushType(payload []byte) string {
	// Quick heuristic: if content-available is in the payload, it's background.
	if bytes.Contains(payload, []byte(`"content-available"`)) {
		return "background"
	}
	return "alert"
}

func extractSendResult(resp *response) *SendResult {
	result := &SendResult{}
	if resp.headers != nil {
		result.TrackingURL = resp.headers.Get("Location")
		result.CorrelationID = resp.headers.Get("x-ms-correlation-request-id")
		result.NotificationID = parseNotificationIDFromLocation(result.TrackingURL)
	}
	return result
}
