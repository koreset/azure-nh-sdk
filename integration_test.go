//go:build integration

package azurenh_test

import (
	"context"
	"os"
	"testing"
	"time"

	azurenh "github.com/koreset/azure-nh-sdk"
)

// Integration tests run against a real Azure Notification Hub.
// Set these environment variables before running:
//
//	AZURE_NH_CONNECTION_STRING - Connection string from Azure portal
//	AZURE_NH_HUB_NAME         - Hub name from Azure portal
//
// Run with: go test -tags=integration -v ./...

func getClient(t *testing.T) *azurenh.Client {
	t.Helper()
	connStr := os.Getenv("AZURE_NH_CONNECTION_STRING")
	hubName := os.Getenv("AZURE_NH_HUB_NAME")
	if connStr == "" || hubName == "" {
		t.Skip("AZURE_NH_CONNECTION_STRING and AZURE_NH_HUB_NAME must be set")
	}

	client, err := azurenh.NewClient(connStr, hubName,
		azurenh.WithRetryPolicy(azurenh.DefaultRetryPolicy()),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	return client
}

func TestIntegration_InstallationLifecycle(t *testing.T) {
	client := getClient(t)
	ctx := context.Background()
	installationID := "integration-test-" + time.Now().Format("20060102150405")

	// Create.
	err := client.CreateOrUpdateInstallation(ctx, azurenh.Installation{
		InstallationID: installationID,
		Platform:       azurenh.PlatformFCMV1,
		PushChannel:    "fake-fcm-token-for-integration-test",
		Tags:           []string{"integration-test"},
	})
	if err != nil {
		t.Fatalf("CreateOrUpdateInstallation: %v", err)
	}

	// Get.
	inst, err := client.GetInstallation(ctx, installationID)
	if err != nil {
		t.Fatalf("GetInstallation: %v", err)
	}
	if inst.InstallationID != installationID {
		t.Errorf("installationId = %q, want %q", inst.InstallationID, installationID)
	}

	// Patch: add tag.
	err = client.PatchInstallation(ctx, installationID, []azurenh.InstallationPatch{
		azurenh.PatchAddTag("patched-tag"),
	})
	if err != nil {
		t.Fatalf("PatchInstallation: %v", err)
	}

	// Delete.
	err = client.DeleteInstallation(ctx, installationID)
	if err != nil {
		t.Fatalf("DeleteInstallation: %v", err)
	}

	// Verify deleted.
	_, err = client.GetInstallation(ctx, installationID)
	if !azurenh.IsNotFound(err) {
		t.Errorf("expected NotFound after delete, got %v", err)
	}
}

func TestIntegration_SendBroadcast(t *testing.T) {
	client := getClient(t)
	ctx := context.Background()

	notification, err := azurenh.NewFCMV1Notification().
		Title("Integration Test").
		Body("This is an integration test notification").
		Data("test", "true").
		Build()
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	// Send to a tag that likely has no real devices.
	result, err := client.Send(ctx, notification, azurenh.WithTagExpression("integration-test-no-devices"))
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	t.Logf("NotificationID: %s, CorrelationID: %s", result.NotificationID, result.CorrelationID)
}

func TestIntegration_ScheduleAndCancel(t *testing.T) {
	client := getClient(t)
	ctx := context.Background()

	notification, err := azurenh.NewAPNSNotification().
		Alert("Scheduled Test", "This should be cancelled").
		Build()
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	deliverAt := time.Now().Add(24 * time.Hour)
	result, err := client.Schedule(ctx, notification, deliverAt, azurenh.WithTagExpression("integration-test-no-devices"))
	if err != nil {
		t.Fatalf("Schedule: %v", err)
	}
	t.Logf("Scheduled NotificationID: %s", result.NotificationID)

	if result.NotificationID != "" {
		err = client.CancelScheduledNotification(ctx, result.NotificationID)
		if err != nil {
			t.Fatalf("CancelScheduledNotification: %v", err)
		}
		t.Log("Successfully cancelled scheduled notification")
	}
}
