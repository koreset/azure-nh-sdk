# Azure Notification Hubs Go SDK

A Go SDK for [Azure Notification Hubs](https://learn.microsoft.com/en-us/azure/notification-hubs/notification-hubs-push-notification-overview) providing full push notification support across all major platforms.

## Features

- **Multi-platform** -- APNS (iOS), FCM v1 (Android), WNS (Windows), ADM (Amazon), Baidu
- **Device management** -- Installation CRUD with JSON Patch updates, legacy XML registration API
- **Flexible sending** -- Broadcast, direct-to-device, batch (up to 1,000), scheduled, and cancel scheduled
- **Tag expressions** -- Target devices with boolean expressions (`"premium && ios"`, `"user:123 || user:456"`)
- **Notification builders** -- Type-safe, fluent builders for each platform with validation
- **Cross-platform templates** -- Send once, deliver natively to every platform
- **Delivery telemetry** -- Track notification outcomes per platform (Standard tier)
- **Built-in resilience** -- Retry with exponential backoff + jitter, circuit breaker, token bucket rate limiter
- **Testable by design** -- `HTTPDoer` interface (satisfied by `*http.Client`), functional options, injectable clock

## Requirements

- Go 1.22 or later
- An Azure Notification Hub (any tier; Standard tier required for telemetry and scheduled send)

## Installation

```bash
go get github.com/koreset/azure-nh-sdk
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"

    azurenh "github.com/koreset/azure-nh-sdk"
)

func main() {
    // Create a client.
    client, err := azurenh.NewClient(
        "Endpoint=sb://myhub-ns.servicebus.windows.net/;SharedAccessKeyName=DefaultFullSharedAccessSignature;SharedAccessKey=base64key==",
        "myhub",
    )
    if err != nil {
        log.Fatal(err)
    }

    // Build and send an iOS notification.
    notification, err := azurenh.NewAPNSNotification().
        Alert("Hello", "Welcome to our app!").
        Badge(1).
        Sound("default").
        Build()
    if err != nil {
        log.Fatal(err)
    }

    result, err := client.Send(context.Background(), notification)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println("Notification ID:", result.NotificationID)
}
```

## Client Configuration

The client is created with `NewClient` and configured using functional options:

```go
client, err := azurenh.NewClient(connectionString, hubName,
    // Retry failed requests with exponential backoff.
    azurenh.WithRetryPolicy(azurenh.DefaultRetryPolicy()),

    // Trip circuit after 5 consecutive failures, wait 30s before retrying.
    azurenh.WithCircuitBreaker(azurenh.CircuitBreakerConfig{
        FailureThreshold: 5,
        ResetTimeout:     30 * time.Second,
        HalfOpenMax:      1,
    }),

    // Rate limit to 100 requests/sec with burst of 20.
    azurenh.WithRateLimiter(azurenh.NewTokenBucketLimiter(100, 20)),

    // Use a custom HTTP client (e.g., with custom timeouts or transport).
    azurenh.WithHTTPClient(&http.Client{Timeout: 10 * time.Second}),

    // Override the API version (default is "2016-07").
    azurenh.WithAPIVersion("2016-07"),
)
```

### Connection String

Find your connection string in the Azure Portal under your Notification Hub > **Access Policies**. The format is:

```
Endpoint=sb://<namespace>.servicebus.windows.net/;SharedAccessKeyName=<key-name>;SharedAccessKey=<key-value>
```

### Custom Retry Policy

```go
azurenh.WithRetryPolicy(azurenh.RetryPolicy{
    MaxAttempts:  5,                      // Total attempts including the initial request
    InitialDelay: 200 * time.Millisecond, // Delay before first retry
    MaxDelay:     60 * time.Second,       // Cap on retry delay
    Multiplier:   2.0,                    // Exponential backoff multiplier
    JitterFactor: 0.1,                    // 10% jitter to avoid thundering herd
})
```

The retry policy only retries on status codes 408, 429, 500, 503, and 504. It also respects `Retry-After` headers from Azure.

---

## Building Notifications

Each platform has a dedicated builder with a fluent API. All builders validate at `.Build()` time and return `(*Notification, error)`.

### APNS (iOS)

```go
// Standard alert notification.
notification, err := azurenh.NewAPNSNotification().
    Alert("Flash Sale", "50% off all items today!").
    Subtitle("Limited time offer").
    Badge(3).
    Sound("default").
    Category("SALE").                       // For actionable notifications
    ThreadID("promotions").                 // Group in notification center
    MutableContent().                       // Enable Notification Service Extension
    Custom("sale_id", "sale-456").          // Custom data in payload root
    Custom("deep_link", "/sales/456").
    Build()
```

Produces:
```json
{
  "aps": {
    "alert": { "title": "Flash Sale", "subtitle": "Limited time offer", "body": "50% off all items today!" },
    "badge": 3,
    "sound": "default",
    "category": "SALE",
    "thread-id": "promotions",
    "mutable-content": 1
  },
  "sale_id": "sale-456",
  "deep_link": "/sales/456"
}
```

```go
// Silent/background notification.
notification, err := azurenh.NewAPNSNotification().
    ContentAvailable().
    Custom("type", "sync").
    Custom("resource", "/api/v1/orders").
    Build()
```

The SDK automatically sets `apns-push-type: background` and `apns-priority: 5` for background notifications.

### FCM v1 (Android)

```go
// Display notification with custom data.
notification, err := azurenh.NewFCMV1Notification().
    Title("New Message").
    Body("You have a message from Alice").
    Image("https://example.com/avatar.png").
    Data("sender_id", "user-789").
    Data("conversation_id", "conv-123").
    AndroidPriority("high").
    AndroidTTL("86400s").
    Build()
```

Produces:
```json
{
  "message": {
    "notification": { "title": "New Message", "body": "You have a message from Alice", "image": "https://example.com/avatar.png" },
    "data": { "sender_id": "user-789", "conversation_id": "conv-123" },
    "android": { "priority": "high", "ttl": "86400s" }
  }
}
```

```go
// Data-only notification (no visible alert, app handles display).
notification, err := azurenh.NewFCMV1Notification().
    Data("type", "sync").
    Data("resource", "/api/v1/orders").
    Build()
```

### WNS (Windows)

```go
notification, err := azurenh.NewWNSNotification().
    RawXML(`<toast>
        <visual>
            <binding template="ToastText02">
                <text id="1">New Message</text>
                <text id="2">You have a new message from Alice</text>
            </binding>
        </visual>
    </toast>`).
    Build()
```

### ADM (Amazon)

```go
notification, err := azurenh.NewADMNotification().
    Data("title", "Order Update").
    Data("body", "Your order has shipped").
    Data("order_id", "ord-456").
    ConsolidationKey("order-updates").
    Build()
```

### Baidu

```go
notification, err := azurenh.NewBaiduNotification().
    Title("Notification Title").
    Description("Notification body text").
    Custom("action", "open_screen").
    Custom("screen_id", "home").
    Build()
```

### Template (Cross-Platform)

Template notifications deliver natively to every platform using templates registered on each device installation:

```go
notification, err := azurenh.NewTemplateNotification(map[string]string{
    "title":   "Order Shipped",
    "message": "Your order #789 has been shipped!",
    "orderId": "789",
})
```

### Raw Notification

If you have a pre-built payload:

```go
payload := []byte(`{"aps":{"alert":"Hello"}}`)
notification, err := azurenh.NewRawNotification(azurenh.FormatApple, payload)
```

---

## Sending Notifications

### Broadcast

Send to all registered devices, or target with tag expressions:

```go
// Broadcast to everyone.
result, err := client.Send(ctx, notification)

// Target with a single tag.
result, err := client.Send(ctx, notification,
    azurenh.WithTagExpression("premium"),
)

// Complex tag expression with boolean operators.
result, err := client.Send(ctx, notification,
    azurenh.WithTagExpression("(follows_RedSox || follows_Cardinals) && location_Boston"),
)
```

### Direct to Device

Send to a specific device by its push channel token (device token / FCM registration ID):

```go
result, err := client.SendDirect(ctx, notification, "apns-device-token-hex-string")
```

### Batch Direct

Send to up to 1,000 devices at once:

```go
tokens := []string{"token1", "token2", "token3"}
result, err := client.SendDirectBatch(ctx, notification, tokens)
```

Returns `ErrBatchTooLarge` if more than 1,000 handles are provided.

### Scheduled Send

Schedule a notification for future delivery (requires Standard tier):

```go
deliverAt := time.Now().Add(2 * time.Hour)
result, err := client.Schedule(ctx, notification, deliverAt,
    azurenh.WithTagExpression("premium"),
)
fmt.Println("Scheduled notification:", result.NotificationID)
```

### Cancel Scheduled

```go
err := client.CancelScheduledNotification(ctx, result.NotificationID)
```

### Test Send

Debug mode sends to a maximum of 10 devices and returns detailed results:

```go
result, err := client.Send(ctx, notification, azurenh.WithTestSend())
```

### Send Result

All send methods return a `*SendResult`:

```go
type SendResult struct {
    NotificationID string // e.g., "12345"
    TrackingURL    string // Full Location header URL for telemetry
    CorrelationID  string // x-ms-correlation-request-id for Azure support
}
```

### APNS-Specific Options

```go
result, err := client.Send(ctx, notification,
    azurenh.WithAPNSPushType("voip"),                          // Override auto-detected push type
    azurenh.WithAPNSPriority(10),                              // 10 = immediate, 5 = power-saving
    azurenh.WithAPNSExpiry(time.Now().Add(24 * time.Hour)),    // Notification expiration
)
```

The SDK auto-detects `apns-push-type` and `apns-priority` from the payload:
- Payloads with `content-available` -> `background` / priority `5`
- All others -> `alert` / priority `10`

---

## Device Installations (Recommended)

The Installation API is the modern, recommended approach for device management. Installations support templates, tags, and partial updates.

### Create or Update

```go
err := client.CreateOrUpdateInstallation(ctx, azurenh.Installation{
    InstallationID: "device-001",
    Platform:       azurenh.PlatformAPNS,       // or PlatformFCMV1, PlatformWNS, etc.
    PushChannel:    "apns-device-token-hex",
    Tags:           []string{"user:123", "premium", "ios"},
    Templates: map[string]azurenh.InstallationTemplate{
        "genericAlert": {
            Body:    `{"aps":{"alert":{"title":"$(title)","body":"$(message)"}}}`,
            Headers: map[string]string{"apns-push-type": "alert"},
        },
        "silentSync": {
            Body: `{"aps":{"content-available":1},"type":"$(type)"}`,
            Headers: map[string]string{"apns-push-type": "background"},
        },
    },
})
```

### Get

```go
installation, err := client.GetInstallation(ctx, "device-001")
fmt.Printf("Platform: %s, Tags: %v\n", installation.Platform, installation.Tags)
```

### Partial Update (Patch)

Update specific fields without replacing the entire installation:

```go
err := client.PatchInstallation(ctx, "device-001", []azurenh.InstallationPatch{
    azurenh.PatchAddTag("opted-in-marketing"),
    azurenh.PatchRemoveTag("trial"),
    azurenh.PatchSetPushChannel("new-device-token"),
    azurenh.PatchAddTemplate("orderUpdate", azurenh.InstallationTemplate{
        Body: `{"aps":{"alert":"Order $(orderId): $(status)"}}`,
    }),
})
```

Available patch helpers:

| Helper | Description |
|--------|-------------|
| `PatchSetPushChannel(channel)` | Replace the device token |
| `PatchSetTags(tags)` | Replace all tags |
| `PatchAddTag(tag)` | Add a single tag |
| `PatchRemoveTag(tag)` | Remove a single tag |
| `PatchAddTemplate(name, template)` | Add a named template |
| `PatchRemoveTemplate(name)` | Remove a named template |
| `PatchSetTemplateBody(name, body)` | Update a template's body |

### Delete

```go
err := client.DeleteInstallation(ctx, "device-001")
```

---

## Legacy Registrations

The Registration API is the older XML-based approach. Use Installations (above) for new projects.

### Create

```go
// Native registration.
reg, err := client.CreateRegistration(ctx, azurenh.Registration{
    Platform:    azurenh.PlatformAPNS,
    DeviceToken: "apns-device-token-hex",
    Tags:        []string{"user:123", "ios"},
})
fmt.Println("Registration ID:", reg.RegistrationID)

// Template registration.
reg, err := client.CreateRegistration(ctx, azurenh.Registration{
    Platform:    azurenh.PlatformFCMV1,
    DeviceToken: "fcm-registration-token",
    Tags:        []string{"user:456", "android"},
    Template:    `{"message":{"notification":{"title":"$(title)","body":"$(message)"}}}`,
})
```

### Update

```go
reg.Tags = append(reg.Tags, "premium")
updated, err := client.UpdateRegistration(ctx, *reg)
```

### Get

```go
reg, err := client.GetRegistration(ctx, "registration-id")
```

### List with Pagination

```go
feed, err := client.ListRegistrations(ctx,
    azurenh.WithTag("premium"),
    azurenh.WithTop(100),
)
for _, reg := range feed.Entries {
    fmt.Printf("ID: %s, Platform: %s, Token: %s\n", reg.RegistrationID, reg.Platform, reg.DeviceToken)
}

// Paginate.
if feed.ContinuationToken != "" {
    nextPage, err := client.ListRegistrations(ctx,
        azurenh.WithTag("premium"),
        azurenh.WithTop(100),
        azurenh.WithContinuationToken(feed.ContinuationToken),
    )
    // ...
}
```

### Delete

```go
// Unconditional delete.
err := client.DeleteRegistration(ctx, "registration-id", "*")

// Conditional delete (optimistic concurrency).
err := client.DeleteRegistration(ctx, reg.RegistrationID, reg.ETag)
```

---

## Delivery Telemetry

Track notification delivery outcomes (requires Standard tier):

```go
result, err := client.Send(ctx, notification)
if err != nil {
    log.Fatal(err)
}

// Wait a moment for processing, then check delivery details.
details, err := client.GetNotificationDetails(ctx, result.NotificationID)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("State: %s\n", details.State)
fmt.Printf("Target Platforms: %s\n", details.TargetPlatforms)

for _, outcome := range details.APNSOutcomes {
    fmt.Printf("  APNS %s: %d\n", outcome.Name, outcome.Count)
}
for _, outcome := range details.FCMV1Outcomes {
    fmt.Printf("  FCM %s: %d\n", outcome.Name, outcome.Count)
}
```

Possible states: `Enqueued`, `Processing`, `Completed`, `Abandoned`, `Canceled`, `NoTargetFound`, `Scheduled`.

---

## Error Handling

The SDK provides structured errors for both client-side validation and API responses.

### Sentinel Errors

```go
// Check for specific error conditions with errors.Is().
if errors.Is(err, azurenh.ErrBatchTooLarge) {
    // Split into smaller batches
}
if errors.Is(err, azurenh.ErrCircuitOpen) {
    // Back off, circuit breaker has tripped
}
if errors.Is(err, azurenh.ErrScheduleInPast) {
    // Deliver time must be in the future
}
```

| Sentinel Error | Meaning |
|----------------|---------|
| `ErrInvalidConnectionString` | Malformed or missing connection string fields |
| `ErrInvalidPayload` | Notification payload failed validation |
| `ErrInvalidPlatform` | Unrecognized platform value |
| `ErrCircuitOpen` | Circuit breaker is open (too many failures) |
| `ErrRateLimited` | Rate limiter context cancelled while waiting |
| `ErrBatchTooLarge` | Batch exceeds 1,000 device limit |
| `ErrScheduleInPast` | Scheduled delivery time is in the past |

### API Errors

```go
// Type-assert to *APIError for detailed Azure error info.
var apiErr *azurenh.APIError
if errors.As(err, &apiErr) {
    fmt.Printf("Code: %s, Status: %d, Message: %s\n", apiErr.Code, apiErr.StatusCode, apiErr.Message)
    fmt.Printf("Request ID: %s\n", apiErr.RequestID)
    fmt.Printf("Retryable: %v\n", apiErr.IsRetryable())
    if d := apiErr.RetryAfter(); d > 0 {
        fmt.Printf("Retry after: %s\n", d)
    }
}
```

### Convenience Helpers

```go
if azurenh.IsNotFound(err) {
    // Resource doesn't exist (404)
}
if azurenh.IsUnauthorized(err) {
    // Bad credentials (401/403)
}
if azurenh.IsThrottled(err) {
    // Too many requests (429)
}
```

### Validation Errors

Client-side validation errors (before any HTTP request is made):

```go
var valErr *azurenh.ValidationError
if errors.As(err, &valErr) {
    fmt.Printf("Field: %s, Message: %s\n", valErr.Field, valErr.Message)
}
```

---

## Resilience

All resilience features are optional and composable. If not configured, requests execute directly without any wrapping.

### Retry with Exponential Backoff

```go
azurenh.WithRetryPolicy(azurenh.RetryPolicy{
    MaxAttempts:  5,
    InitialDelay: 500 * time.Millisecond,
    MaxDelay:     30 * time.Second,
    Multiplier:   2.0,
    JitterFactor: 0.1,
})
```

Only retries on transient errors (408, 429, 500, 503, 504). Respects `Retry-After` headers. Non-retryable errors (400, 401, 404, etc.) fail immediately.

### Circuit Breaker

Prevents cascading failures by stopping requests after repeated failures:

```go
azurenh.WithCircuitBreaker(azurenh.CircuitBreakerConfig{
    FailureThreshold: 5,              // Trip after 5 consecutive failures
    ResetTimeout:     30 * time.Second, // Wait 30s before trying again
    HalfOpenMax:      1,              // Allow 1 probe request in half-open state
})
```

States: **Closed** (normal) -> **Open** (all requests fail with `ErrCircuitOpen`) -> **Half-Open** (probe requests) -> **Closed** (on success).

### Rate Limiter

Token bucket rate limiting to stay within Azure NH quotas:

```go
azurenh.WithRateLimiter(azurenh.NewTokenBucketLimiter(
    100, // 100 requests per second
    20,  // Burst up to 20 requests
))
```

### Execution Order

When all three are configured, each request flows through:

1. **Rate limiter** -- blocks until a token is available
2. **Circuit breaker** -- fails fast if the circuit is open
3. **HTTP request** -- executes the actual API call
4. **On failure** -- records with circuit breaker, computes backoff, retries (if retryable)
5. **On success** -- records with circuit breaker, returns result

---

## Supported Platforms

| Platform | Constant | Format | Builder |
|----------|----------|--------|---------|
| Apple Push Notification Service | `PlatformAPNS` | `FormatApple` | `NewAPNSNotification()` |
| Firebase Cloud Messaging v1 | `PlatformFCMV1` | `FormatFCMV1` | `NewFCMV1Notification()` |
| Windows Notification Service | `PlatformWNS` | `FormatWindows` | `NewWNSNotification()` |
| Amazon Device Messaging | `PlatformADM` | `FormatADM` | `NewADMNotification()` |
| Baidu Cloud Push | `PlatformBaidu` | `FormatBaidu` | `NewBaiduNotification()` |
| Cross-platform Template | -- | `FormatTemplate` | `NewTemplateNotification()` |

---

## Testing

### Running Unit Tests

```bash
go test ./... -v
```

### Running Integration Tests

Integration tests run against a real Azure Notification Hub and are gated behind a build tag:

```bash
export AZURE_NH_CONNECTION_STRING="Endpoint=sb://..."
export AZURE_NH_HUB_NAME="myhub"
go test -tags=integration -v ./...
```

### Mocking in Your Tests

The SDK is designed for testability. Implement the `HTTPDoer` interface to mock HTTP responses:

```go
type mockHTTP struct {
    doFunc func(req *http.Request) (*http.Response, error)
}

func (m *mockHTTP) Do(req *http.Request) (*http.Response, error) {
    return m.doFunc(req)
}

func TestMyService(t *testing.T) {
    client, _ := azurenh.NewClient(connectionString, hubName,
        azurenh.WithHTTPClient(&mockHTTP{
            doFunc: func(req *http.Request) (*http.Response, error) {
                return &http.Response{
                    StatusCode: 201,
                    Header:     http.Header{"Location": []string{"https://ns.servicebus.windows.net/hub/messages/123"}},
                    Body:       io.NopCloser(strings.NewReader("")),
                }, nil
            },
        }),
    )
    // Use client in your tests...
}
```

Since `*http.Client` already satisfies `HTTPDoer`, you can also pass it directly:

```go
azurenh.WithHTTPClient(&http.Client{Timeout: 5 * time.Second})
```

---

## API Reference

### Client Methods

| Method | Description |
|--------|-------------|
| `Send(ctx, notification, ...SendOption)` | Broadcast or tag-targeted send |
| `SendDirect(ctx, notification, deviceHandle)` | Send to a specific device |
| `SendDirectBatch(ctx, notification, deviceHandles)` | Send to up to 1,000 devices |
| `Schedule(ctx, notification, deliverAt, ...SendOption)` | Schedule for future delivery |
| `CancelScheduledNotification(ctx, notificationID)` | Cancel a scheduled notification |
| `CreateOrUpdateInstallation(ctx, installation)` | Create or replace a device installation |
| `GetInstallation(ctx, installationID)` | Get an installation by ID |
| `PatchInstallation(ctx, installationID, patches)` | Partially update an installation |
| `DeleteInstallation(ctx, installationID)` | Delete an installation |
| `CreateRegistration(ctx, registration)` | Create a legacy registration |
| `UpdateRegistration(ctx, registration)` | Update a legacy registration |
| `GetRegistration(ctx, registrationID)` | Get a registration by ID |
| `ListRegistrations(ctx, ...ListRegistrationsOption)` | List registrations with filtering |
| `DeleteRegistration(ctx, registrationID, etag)` | Delete a registration |
| `GetNotificationDetails(ctx, notificationID)` | Get delivery telemetry |

### Client Options

| Option | Description |
|--------|-------------|
| `WithHTTPClient(doer)` | Custom HTTP client |
| `WithRetryPolicy(policy)` | Retry with exponential backoff |
| `WithCircuitBreaker(config)` | Circuit breaker for fault tolerance |
| `WithRateLimiter(limiter)` | Token bucket rate limiting |
| `WithAPIVersion(version)` | Override API version (default `2016-07`) |
| `WithClock(clock)` | Override clock for testing |

### Send Options

| Option | Description |
|--------|-------------|
| `WithTagExpression(expr)` | Tag expression for targeting |
| `WithTestSend()` | Debug mode (max 10 devices) |
| `WithAPNSExpiry(time)` | APNS notification expiration |
| `WithAPNSPriority(int)` | APNS priority (5 or 10) |
| `WithAPNSPushType(string)` | Override auto-detected push type |

## License

MIT
