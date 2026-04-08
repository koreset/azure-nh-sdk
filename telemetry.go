package azurenh

import (
	"context"
	"encoding/xml"
	"fmt"
	"strings"
)

// GetNotificationDetails retrieves delivery details for a sent notification.
// Requires Standard tier Azure Notification Hub. The notificationID is returned
// in SendResult.NotificationID from any send operation.
func (c *Client) GetNotificationDetails(ctx context.Context, notificationID string) (*NotificationDetails, error) {
	if notificationID == "" {
		return nil, &ValidationError{Field: "notificationId", Message: "notification ID is required"}
	}

	resp, err := c.exec(ctx, "GET", c.buildURL("messages", notificationID), nil, nil)
	if err != nil {
		return nil, err
	}

	var details NotificationDetails
	if err := xml.Unmarshal(resp.body, &details); err != nil {
		return nil, fmt.Errorf("azurenh: failed to parse notification details: %w", err)
	}
	return &details, nil
}

// parseNotificationIDFromLocation extracts the notification ID from the
// Location header URL returned by send operations.
// Example: https://ns.servicebus.windows.net/hub/messages/12345?api-version=2016-07 -> 12345
func parseNotificationIDFromLocation(locationURL string) string {
	if locationURL == "" {
		return ""
	}
	// Strip query string.
	if idx := strings.Index(locationURL, "?"); idx >= 0 {
		locationURL = locationURL[:idx]
	}
	// Get last path segment.
	if idx := strings.LastIndex(locationURL, "/"); idx >= 0 {
		return locationURL[idx+1:]
	}
	return ""
}
