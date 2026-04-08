// Package azurenh provides a Go SDK for Azure Notification Hubs.
//
// It supports device installation management, legacy registrations,
// push notification delivery (broadcast, direct, batch, scheduled),
// and delivery telemetry across all major platforms: APNS (iOS),
// FCM v1 (Android), WNS (Windows), ADM (Amazon), and Baidu.
//
// # Quick Start
//
//	client, err := azurenh.NewClient(connectionString, hubName)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	notification, err := azurenh.NewAPNSNotification().
//	    Alert("Hello", "World").
//	    Badge(1).
//	    Sound("default").
//	    Build()
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	result, err := client.Send(ctx, notification, azurenh.WithTagExpression("user:123"))
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println("Notification ID:", result.NotificationID)
//
// # Resilience
//
// The client supports optional retry with exponential backoff, circuit breaker,
// and rate limiting via functional options:
//
//	client, err := azurenh.NewClient(connectionString, hubName,
//	    azurenh.WithRetryPolicy(azurenh.DefaultRetryPolicy()),
//	    azurenh.WithCircuitBreaker(azurenh.CircuitBreakerConfig{
//	        FailureThreshold: 5,
//	        ResetTimeout:     30 * time.Second,
//	    }),
//	    azurenh.WithRateLimiter(azurenh.NewTokenBucketLimiter(100, 20)),
//	)
package azurenh
