package azurenh

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestCreateOrUpdateInstallation(t *testing.T) {
	var capturedBody []byte
	c := newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != "PUT" {
			t.Errorf("method = %q, want PUT", req.Method)
		}
		if req.Header.Get("Content-Type") != "application/json" {
			t.Error("expected application/json content type")
		}
		buf := make([]byte, 4096)
		n, _ := req.Body.Read(buf)
		capturedBody = buf[:n]
		return mockResponse(200, "", nil), nil
	})

	err := c.CreateOrUpdateInstallation(ctx(), Installation{
		InstallationID: "inst-123",
		Platform:       PlatformAPNS,
		PushChannel:    "device-token-abc",
		Tags:           []string{"user:123", "premium"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var inst Installation
	json.Unmarshal(capturedBody, &inst)
	if inst.InstallationID != "inst-123" {
		t.Errorf("installationId = %q, want inst-123", inst.InstallationID)
	}
}

func TestCreateOrUpdateInstallation_Validation(t *testing.T) {
	c := newTestClient(nil)

	err := c.CreateOrUpdateInstallation(ctx(), Installation{})
	if err == nil {
		t.Error("expected validation error for empty installation ID")
	}

	err = c.CreateOrUpdateInstallation(ctx(), Installation{
		InstallationID: "id",
		Platform:       "invalid",
		PushChannel:    "token",
	})
	if err == nil {
		t.Error("expected validation error for invalid platform")
	}

	err = c.CreateOrUpdateInstallation(ctx(), Installation{
		InstallationID: "id",
		Platform:       PlatformAPNS,
	})
	if err == nil {
		t.Error("expected validation error for empty push channel")
	}
}

func TestGetInstallation(t *testing.T) {
	c := newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != "GET" {
			t.Errorf("method = %q, want GET", req.Method)
		}
		return mockResponse(200, `{"installationId":"inst-1","platform":"apns","pushChannel":"token"}`, nil), nil
	})

	inst, err := c.GetInstallation(ctx(), "inst-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inst.InstallationID != "inst-1" {
		t.Errorf("installationId = %q, want inst-1", inst.InstallationID)
	}
	if inst.Platform != PlatformAPNS {
		t.Errorf("platform = %q, want apns", inst.Platform)
	}
}

func TestPatchInstallation(t *testing.T) {
	c := newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != "PATCH" {
			t.Errorf("method = %q, want PATCH", req.Method)
		}
		if req.Header.Get("Content-Type") != "application/json-patch+json" {
			t.Error("expected json-patch content type")
		}
		return mockResponse(200, "", nil), nil
	})

	err := c.PatchInstallation(ctx(), "inst-1", []InstallationPatch{
		PatchAddTag("new-tag"),
		PatchSetPushChannel("new-token"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteInstallation(t *testing.T) {
	c := newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != "DELETE" {
			t.Errorf("method = %q, want DELETE", req.Method)
		}
		return mockResponse(200, "", nil), nil
	})

	err := c.DeleteInstallation(ctx(), "inst-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPatchHelpers(t *testing.T) {
	p := PatchAddTag("tag1")
	if p.Op != PatchAdd || p.Path != "/tags" {
		t.Errorf("PatchAddTag: op=%q path=%q", p.Op, p.Path)
	}

	p = PatchRemoveTag("tag1")
	if p.Op != PatchRemove || p.Path != "/tags/tag1" {
		t.Errorf("PatchRemoveTag: op=%q path=%q", p.Op, p.Path)
	}

	p = PatchSetTags([]string{"a", "b"})
	if p.Op != PatchReplace || p.Path != "/tags" {
		t.Errorf("PatchSetTags: op=%q path=%q", p.Op, p.Path)
	}

	p = PatchSetPushChannel("token")
	if p.Op != PatchReplace || p.Path != "/pushChannel" {
		t.Errorf("PatchSetPushChannel: op=%q path=%q", p.Op, p.Path)
	}

	tmpl := InstallationTemplate{Body: `{"aps":{"alert":"$(message)"}}`}
	p = PatchAddTemplate("mytemplate", tmpl)
	if p.Op != PatchAdd || p.Path != "/templates/mytemplate" {
		t.Errorf("PatchAddTemplate: op=%q path=%q", p.Op, p.Path)
	}

	p = PatchRemoveTemplate("mytemplate")
	if p.Op != PatchRemove || p.Path != "/templates/mytemplate" {
		t.Errorf("PatchRemoveTemplate: op=%q path=%q", p.Op, p.Path)
	}

	p = PatchSetTemplateBody("mytemplate", `{"aps":{}}`)
	if p.Op != PatchReplace || p.Path != "/templates/mytemplate/body" {
		t.Errorf("PatchSetTemplateBody: op=%q path=%q", p.Op, p.Path)
	}
}

func ctx() context.Context {
	return context.Background()
}
