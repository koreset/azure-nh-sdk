package azurenh

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
)

// CreateOrUpdateInstallation creates or updates a device installation.
// If the installation already exists, it is fully replaced.
func (c *Client) CreateOrUpdateInstallation(ctx context.Context, installation Installation) error {
	if installation.InstallationID == "" {
		return &ValidationError{Field: "installationId", Message: "installation ID is required"}
	}
	if !installation.Platform.IsValid() {
		return &ValidationError{Field: "platform", Message: fmt.Sprintf("invalid platform: %q", installation.Platform)}
	}
	if installation.PushChannel == "" {
		return &ValidationError{Field: "pushChannel", Message: "push channel (device token) is required"}
	}

	body, err := json.Marshal(installation)
	if err != nil {
		return fmt.Errorf("azurenh: failed to marshal installation: %w", err)
	}

	_, err = c.exec(ctx, "PUT", c.buildURL("installations", installation.InstallationID), map[string]string{
		"Content-Type": "application/json",
	}, bytes.NewReader(body))
	return err
}

// GetInstallation retrieves a device installation by ID.
func (c *Client) GetInstallation(ctx context.Context, installationID string) (*Installation, error) {
	if installationID == "" {
		return nil, &ValidationError{Field: "installationId", Message: "installation ID is required"}
	}

	resp, err := c.exec(ctx, "GET", c.buildURL("installations", installationID), nil, nil)
	if err != nil {
		return nil, err
	}

	var inst Installation
	if err := json.Unmarshal(resp.body, &inst); err != nil {
		return nil, fmt.Errorf("azurenh: failed to unmarshal installation: %w", err)
	}
	return &inst, nil
}

// PatchInstallation applies partial updates to an installation using JSON Patch operations.
func (c *Client) PatchInstallation(ctx context.Context, installationID string, patches []InstallationPatch) error {
	if installationID == "" {
		return &ValidationError{Field: "installationId", Message: "installation ID is required"}
	}
	if len(patches) == 0 {
		return &ValidationError{Field: "patches", Message: "at least one patch operation is required"}
	}

	body, err := json.Marshal(patches)
	if err != nil {
		return fmt.Errorf("azurenh: failed to marshal patches: %w", err)
	}

	_, err = c.exec(ctx, "PATCH", c.buildURL("installations", installationID), map[string]string{
		"Content-Type": "application/json-patch+json",
	}, bytes.NewReader(body))
	return err
}

// DeleteInstallation removes a device installation.
func (c *Client) DeleteInstallation(ctx context.Context, installationID string) error {
	if installationID == "" {
		return &ValidationError{Field: "installationId", Message: "installation ID is required"}
	}

	_, err := c.exec(ctx, "DELETE", c.buildURL("installations", installationID), nil, nil)
	return err
}

// --- Patch helpers ---

func patchJSON(value any) json.RawMessage {
	data, _ := json.Marshal(value)
	return data
}

// PatchSetPushChannel creates a patch to replace the push channel.
func PatchSetPushChannel(channel string) InstallationPatch {
	return InstallationPatch{Op: PatchReplace, Path: "/pushChannel", Value: patchJSON(channel)}
}

// PatchSetTags creates a patch to replace all tags.
func PatchSetTags(tags []string) InstallationPatch {
	return InstallationPatch{Op: PatchReplace, Path: "/tags", Value: patchJSON(tags)}
}

// PatchAddTag creates a patch to add a single tag.
func PatchAddTag(tag string) InstallationPatch {
	return InstallationPatch{Op: PatchAdd, Path: "/tags", Value: patchJSON(tag)}
}

// PatchRemoveTag creates a patch to remove a single tag.
func PatchRemoveTag(tag string) InstallationPatch {
	return InstallationPatch{Op: PatchRemove, Path: "/tags/" + tag}
}

// PatchAddTemplate creates a patch to add a named template.
func PatchAddTemplate(name string, template InstallationTemplate) InstallationPatch {
	return InstallationPatch{Op: PatchAdd, Path: "/templates/" + name, Value: patchJSON(template)}
}

// PatchRemoveTemplate creates a patch to remove a named template.
func PatchRemoveTemplate(name string) InstallationPatch {
	return InstallationPatch{Op: PatchRemove, Path: "/templates/" + name}
}

// PatchSetTemplateBody creates a patch to update a template body.
func PatchSetTemplateBody(name, body string) InstallationPatch {
	return InstallationPatch{Op: PatchReplace, Path: "/templates/" + name + "/body", Value: patchJSON(body)}
}
