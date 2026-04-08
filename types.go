package azurenh

import (
	"encoding/json"
	"time"
)

// Installation represents a device installation in Azure Notification Hubs.
type Installation struct {
	InstallationID     string                          `json:"installationId"`
	LastActiveOn       *time.Time                      `json:"lastActiveOn,omitempty"`
	ExpirationTime     *time.Time                      `json:"expirationTime,omitempty"`
	LastUpdate         *time.Time                      `json:"lastUpdate,omitempty"`
	Platform           Platform                        `json:"platform"`
	PushChannel        string                          `json:"pushChannel"`
	ExpiredPushChannel bool                            `json:"expiredPushChannel,omitempty"`
	Tags               []string                        `json:"tags,omitempty"`
	Templates          map[string]InstallationTemplate `json:"templates,omitempty"`
}

// InstallationTemplate defines a notification template on an installation.
type InstallationTemplate struct {
	Body    string            `json:"body"`
	Headers map[string]string `json:"headers,omitempty"`
	Expiry  *time.Time        `json:"expiry,omitempty"`
	Tags    []string          `json:"tags,omitempty"`
}

// InstallationPatch represents a JSON Patch operation on an installation.
type InstallationPatch struct {
	Op    PatchOp         `json:"op"`
	Path  string          `json:"path"`
	Value json.RawMessage `json:"value,omitempty"`
}

// PatchOp is the JSON Patch operation type.
type PatchOp string

const (
	PatchAdd     PatchOp = "add"
	PatchRemove  PatchOp = "remove"
	PatchReplace PatchOp = "replace"
)

// Registration represents a legacy device registration in Azure Notification Hubs.
type Registration struct {
	RegistrationID string             `json:"registrationId,omitempty"`
	ETag           string             `json:"etag,omitempty"`
	DeviceToken    string             `json:"deviceToken,omitempty"`
	ExpirationTime *time.Time         `json:"expirationTime,omitempty"`
	Tags           []string           `json:"tags,omitempty"`
	Platform       Platform           `json:"platform,omitempty"`
	Format         NotificationFormat `json:"format,omitempty"`
	Template       string             `json:"template,omitempty"`
}

// RegistrationFeed is a collection of registration results from a list operation.
type RegistrationFeed struct {
	Entries           []Registration `json:"entries"`
	ContinuationToken string         `json:"continuationToken,omitempty"`
}

// NotificationOutcome represents delivery outcome counters for a platform.
type NotificationOutcome struct {
	Name  string `xml:"Name" json:"name"`
	Count int    `xml:"Count" json:"count"`
}

// NotificationDetails contains detailed telemetry for a sent notification.
type NotificationDetails struct {
	NotificationID  string              `xml:"NotificationId" json:"notificationId"`
	State           NotificationState   `xml:"State" json:"state"`
	EnqueueTime     string              `xml:"EnqueueTime" json:"enqueueTime"`
	StartTime       string              `xml:"StartTime" json:"startTime"`
	EndTime         string              `xml:"EndTime" json:"endTime"`
	Body            string              `xml:"NotificationBody" json:"body"`
	TargetPlatforms string              `xml:"TargetPlatforms" json:"targetPlatforms"`
	APNSOutcomes    []NotificationOutcome `xml:"ApnsOutcomeCounts>Outcome" json:"apnsOutcomes,omitempty"`
	FCMV1Outcomes   []NotificationOutcome `xml:"FcmV1OutcomeCounts>Outcome" json:"fcmv1Outcomes,omitempty"`
	WNSOutcomes     []NotificationOutcome `xml:"WnsOutcomeCounts>Outcome" json:"wnsOutcomes,omitempty"`
	ADMOutcomes     []NotificationOutcome `xml:"AdmOutcomeCounts>Outcome" json:"admOutcomes,omitempty"`
}

// NotificationState is the lifecycle state of a notification.
type NotificationState string

const (
	StateEnqueued      NotificationState = "Enqueued"
	StateProcessing    NotificationState = "Processing"
	StateCompleted     NotificationState = "Completed"
	StateAbandoned     NotificationState = "Abandoned"
	StateCanceled      NotificationState = "Canceled"
	StateNoTargetFound NotificationState = "NoTargetFound"
	StateScheduled     NotificationState = "Scheduled"
	StateUnknown       NotificationState = "Unknown"
)

// SendResult is returned from send operations.
type SendResult struct {
	NotificationID string // Extracted from Location header
	TrackingURL    string // Full Location header URL
	CorrelationID  string // x-ms-correlation-request-id
}
