package azurenh_test

import (
	"context"
	"fmt"
	"log"
	"time"

	azurenh "github.com/koreset/azure-nh-sdk"
)

func Example_basicUsage() {
	// Create a client with resilience options.
	client, err := azurenh.NewClient(
		"Endpoint=sb://myhub-ns.servicebus.windows.net/;SharedAccessKeyName=DefaultFullSharedAccessSignature;SharedAccessKey=base64key==",
		"myhub",
		azurenh.WithRetryPolicy(azurenh.DefaultRetryPolicy()),
		azurenh.WithRateLimiter(azurenh.NewTokenBucketLimiter(100, 20)),
	)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	// Register a device installation.
	err = client.CreateOrUpdateInstallation(ctx, azurenh.Installation{
		InstallationID: "device-001",
		Platform:       azurenh.PlatformAPNS,
		PushChannel:    "apns-device-token-hex",
		Tags:           []string{"user:123", "premium"},
		Templates: map[string]azurenh.InstallationTemplate{
			"genericAlert": {
				Body: `{"aps":{"alert":{"title":"$(title)","body":"$(message)"}}}`,
			},
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	// Send an APNS notification to premium users.
	notification, err := azurenh.NewAPNSNotification().
		Alert("Flash Sale", "50% off all items!").
		Badge(1).
		Sound("default").
		Category("SALE").
		Custom("sale_id", "sale-456").
		Build()
	if err != nil {
		log.Fatal(err)
	}

	result, err := client.Send(ctx, notification, azurenh.WithTagExpression("premium"))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Sent notification:", result.NotificationID)

	// Send an FCM v1 data-only notification.
	fcmNotification, err := azurenh.NewFCMV1Notification().
		Data("type", "sync").
		Data("resource", "/api/v1/orders/latest").
		Build()
	if err != nil {
		log.Fatal(err)
	}

	_, err = client.Send(ctx, fcmNotification, azurenh.WithTagExpression("user:123"))
	if err != nil {
		log.Fatal(err)
	}

	// Send a cross-platform template notification.
	tmplNotification, err := azurenh.NewTemplateNotification(map[string]string{
		"title":   "Order Shipped",
		"message": "Your order #789 has shipped!",
	})
	if err != nil {
		log.Fatal(err)
	}

	_, err = client.Send(ctx, tmplNotification, azurenh.WithTagExpression("user:123"))
	if err != nil {
		log.Fatal(err)
	}

	// Schedule a notification for later.
	_, err = client.Schedule(ctx, notification, time.Now().Add(2*time.Hour),
		azurenh.WithTagExpression("premium"),
	)
	if err != nil {
		log.Fatal(err)
	}

	// Update installation tags.
	err = client.PatchInstallation(ctx, "device-001", []azurenh.InstallationPatch{
		azurenh.PatchAddTag("opted-in-marketing"),
		azurenh.PatchRemoveTag("trial"),
	})
	if err != nil {
		log.Fatal(err)
	}
}
