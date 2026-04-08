package azurenh

import (
	"context"
	"encoding/xml"
	"fmt"
	"strings"
)

// ListRegistrationsOption configures list registration queries.
type ListRegistrationsOption func(*listRegistrationsParams)

type listRegistrationsParams struct {
	tag               string
	top               int
	continuationToken string
}

// WithTag filters registrations by tag.
func WithTag(tag string) ListRegistrationsOption {
	return func(p *listRegistrationsParams) {
		p.tag = tag
	}
}

// WithTop limits the number of registrations returned.
func WithTop(n int) ListRegistrationsOption {
	return func(p *listRegistrationsParams) {
		p.top = n
	}
}

// WithContinuationToken sets the continuation token for pagination.
func WithContinuationToken(token string) ListRegistrationsOption {
	return func(p *listRegistrationsParams) {
		p.continuationToken = token
	}
}

// CreateRegistration creates a new device registration.
// For template registrations, set the Template field on Registration.
func (c *Client) CreateRegistration(ctx context.Context, reg Registration) (*Registration, error) {
	if err := validateRegistration(reg); err != nil {
		return nil, err
	}

	xmlBody, err := buildRegistrationXML(reg)
	if err != nil {
		return nil, err
	}

	resp, err := c.exec(ctx, "POST", c.buildURL("registrations"), map[string]string{
		"Content-Type": "application/atom+xml;type=entry;charset=utf-8",
	}, strings.NewReader(xmlBody))
	if err != nil {
		return nil, err
	}

	return parseRegistrationResponse(resp.body)
}

// UpdateRegistration updates an existing registration. The RegistrationID must be set.
func (c *Client) UpdateRegistration(ctx context.Context, reg Registration) (*Registration, error) {
	if reg.RegistrationID == "" {
		return nil, &ValidationError{Field: "registrationId", Message: "registration ID is required for update"}
	}
	if err := validateRegistration(reg); err != nil {
		return nil, err
	}

	xmlBody, err := buildRegistrationXML(reg)
	if err != nil {
		return nil, err
	}

	etag := reg.ETag
	if etag == "" {
		etag = "*"
	}

	resp, err := c.exec(ctx, "PUT", c.buildURL("registrations", reg.RegistrationID), map[string]string{
		"Content-Type": "application/atom+xml;type=entry;charset=utf-8",
		"If-Match":     etag,
	}, strings.NewReader(xmlBody))
	if err != nil {
		return nil, err
	}

	return parseRegistrationResponse(resp.body)
}

// GetRegistration retrieves a single registration by ID.
func (c *Client) GetRegistration(ctx context.Context, registrationID string) (*Registration, error) {
	if registrationID == "" {
		return nil, &ValidationError{Field: "registrationId", Message: "registration ID is required"}
	}

	resp, err := c.exec(ctx, "GET", c.buildURL("registrations", registrationID), nil, nil)
	if err != nil {
		return nil, err
	}

	return parseRegistrationResponse(resp.body)
}

// ListRegistrations retrieves registrations for the hub.
func (c *Client) ListRegistrations(ctx context.Context, opts ...ListRegistrationsOption) (*RegistrationFeed, error) {
	params := &listRegistrationsParams{}
	for _, opt := range opts {
		opt(params)
	}

	u := c.buildURL("registrations")
	q := u.Query()
	if params.tag != "" {
		q.Set("$filter", fmt.Sprintf("Tags eq '%s'", params.tag))
	}
	if params.top > 0 {
		q.Set("$top", fmt.Sprintf("%d", params.top))
	}
	if params.continuationToken != "" {
		q.Set("ContinuationToken", params.continuationToken)
	}
	u.RawQuery = q.Encode()

	resp, err := c.exec(ctx, "GET", u, nil, nil)
	if err != nil {
		return nil, err
	}

	feed, err := parseRegistrationFeed(resp.body)
	if err != nil {
		return nil, err
	}

	if ct := resp.headers.Get("X-MS-ContinuationToken"); ct != "" {
		feed.ContinuationToken = ct
	}
	return feed, nil
}

// DeleteRegistration removes a registration. Use etag "*" for unconditional delete.
func (c *Client) DeleteRegistration(ctx context.Context, registrationID, etag string) error {
	if registrationID == "" {
		return &ValidationError{Field: "registrationId", Message: "registration ID is required"}
	}
	if etag == "" {
		etag = "*"
	}

	_, err := c.exec(ctx, "DELETE", c.buildURL("registrations", registrationID), map[string]string{
		"If-Match": etag,
	}, nil)
	return err
}

func validateRegistration(reg Registration) error {
	if !reg.Platform.IsValid() {
		return &ValidationError{Field: "platform", Message: fmt.Sprintf("invalid platform: %q", reg.Platform)}
	}
	if reg.DeviceToken == "" {
		return &ValidationError{Field: "deviceToken", Message: "device token is required"}
	}
	return nil
}

// parseRegistrationResponse parses an Atom XML entry response into a Registration.
func parseRegistrationResponse(body []byte) (*Registration, error) {
	var entry struct {
		Content struct {
			AppleReg *struct {
				RegistrationID string `xml:"RegistrationId"`
				ETag           string `xml:"ETag"`
				Tags           string `xml:"Tags"`
				DeviceToken    string `xml:"DeviceToken"`
			} `xml:"AppleRegistrationDescription"`
			AppleTemplateReg *struct {
				RegistrationID string `xml:"RegistrationId"`
				ETag           string `xml:"ETag"`
				Tags           string `xml:"Tags"`
				DeviceToken    string `xml:"DeviceToken"`
				BodyTemplate   string `xml:"BodyTemplate"`
			} `xml:"AppleTemplateRegistrationDescription"`
			FcmV1Reg *struct {
				RegistrationID   string `xml:"RegistrationId"`
				ETag             string `xml:"ETag"`
				Tags             string `xml:"Tags"`
				FcmV1RegistrationId string `xml:"FcmV1RegistrationId"`
			} `xml:"FcmV1RegistrationDescription"`
			FcmV1TemplateReg *struct {
				RegistrationID   string `xml:"RegistrationId"`
				ETag             string `xml:"ETag"`
				Tags             string `xml:"Tags"`
				FcmV1RegistrationId string `xml:"FcmV1RegistrationId"`
				BodyTemplate     string `xml:"BodyTemplate"`
			} `xml:"FcmV1TemplateRegistrationDescription"`
			WnsReg *struct {
				RegistrationID string `xml:"RegistrationId"`
				ETag           string `xml:"ETag"`
				Tags           string `xml:"Tags"`
				ChannelUri     string `xml:"ChannelUri"`
			} `xml:"WindowsRegistrationDescription"`
			WnsTemplateReg *struct {
				RegistrationID string `xml:"RegistrationId"`
				ETag           string `xml:"ETag"`
				Tags           string `xml:"Tags"`
				ChannelUri     string `xml:"ChannelUri"`
				BodyTemplate   string `xml:"BodyTemplate"`
			} `xml:"WindowsTemplateRegistrationDescription"`
			AdmReg *struct {
				RegistrationID    string `xml:"RegistrationId"`
				ETag              string `xml:"ETag"`
				Tags              string `xml:"Tags"`
				AdmRegistrationId string `xml:"AdmRegistrationId"`
			} `xml:"AdmRegistrationDescription"`
			AdmTemplateReg *struct {
				RegistrationID    string `xml:"RegistrationId"`
				ETag              string `xml:"ETag"`
				Tags              string `xml:"Tags"`
				AdmRegistrationId string `xml:"AdmRegistrationId"`
				BodyTemplate      string `xml:"BodyTemplate"`
			} `xml:"AdmTemplateRegistrationDescription"`
		} `xml:"content"`
	}

	if err := xml.Unmarshal(body, &entry); err != nil {
		return nil, fmt.Errorf("azurenh: failed to parse registration response: %w", err)
	}

	reg := &Registration{}
	c := entry.Content

	switch {
	case c.AppleTemplateReg != nil:
		reg.RegistrationID = c.AppleTemplateReg.RegistrationID
		reg.ETag = c.AppleTemplateReg.ETag
		reg.DeviceToken = c.AppleTemplateReg.DeviceToken
		reg.Platform = PlatformAPNS
		reg.Template = c.AppleTemplateReg.BodyTemplate
		reg.Tags = splitTags(c.AppleTemplateReg.Tags)
	case c.AppleReg != nil:
		reg.RegistrationID = c.AppleReg.RegistrationID
		reg.ETag = c.AppleReg.ETag
		reg.DeviceToken = c.AppleReg.DeviceToken
		reg.Platform = PlatformAPNS
		reg.Tags = splitTags(c.AppleReg.Tags)
	case c.FcmV1TemplateReg != nil:
		reg.RegistrationID = c.FcmV1TemplateReg.RegistrationID
		reg.ETag = c.FcmV1TemplateReg.ETag
		reg.DeviceToken = c.FcmV1TemplateReg.FcmV1RegistrationId
		reg.Platform = PlatformFCMV1
		reg.Template = c.FcmV1TemplateReg.BodyTemplate
		reg.Tags = splitTags(c.FcmV1TemplateReg.Tags)
	case c.FcmV1Reg != nil:
		reg.RegistrationID = c.FcmV1Reg.RegistrationID
		reg.ETag = c.FcmV1Reg.ETag
		reg.DeviceToken = c.FcmV1Reg.FcmV1RegistrationId
		reg.Platform = PlatformFCMV1
		reg.Tags = splitTags(c.FcmV1Reg.Tags)
	case c.WnsTemplateReg != nil:
		reg.RegistrationID = c.WnsTemplateReg.RegistrationID
		reg.ETag = c.WnsTemplateReg.ETag
		reg.DeviceToken = c.WnsTemplateReg.ChannelUri
		reg.Platform = PlatformWNS
		reg.Template = c.WnsTemplateReg.BodyTemplate
		reg.Tags = splitTags(c.WnsTemplateReg.Tags)
	case c.WnsReg != nil:
		reg.RegistrationID = c.WnsReg.RegistrationID
		reg.ETag = c.WnsReg.ETag
		reg.DeviceToken = c.WnsReg.ChannelUri
		reg.Platform = PlatformWNS
		reg.Tags = splitTags(c.WnsReg.Tags)
	case c.AdmTemplateReg != nil:
		reg.RegistrationID = c.AdmTemplateReg.RegistrationID
		reg.ETag = c.AdmTemplateReg.ETag
		reg.DeviceToken = c.AdmTemplateReg.AdmRegistrationId
		reg.Platform = PlatformADM
		reg.Template = c.AdmTemplateReg.BodyTemplate
		reg.Tags = splitTags(c.AdmTemplateReg.Tags)
	case c.AdmReg != nil:
		reg.RegistrationID = c.AdmReg.RegistrationID
		reg.ETag = c.AdmReg.ETag
		reg.DeviceToken = c.AdmReg.AdmRegistrationId
		reg.Platform = PlatformADM
		reg.Tags = splitTags(c.AdmReg.Tags)
	default:
		return nil, fmt.Errorf("azurenh: unrecognized registration type in response")
	}

	return reg, nil
}

func parseRegistrationFeed(body []byte) (*RegistrationFeed, error) {
	var atomFeed struct {
		Entries []struct {
			Content struct {
				InnerXML []byte `xml:",innerxml"`
			} `xml:"content"`
		} `xml:"entry"`
	}

	if err := xml.Unmarshal(body, &atomFeed); err != nil {
		return nil, fmt.Errorf("azurenh: failed to parse registration feed: %w", err)
	}

	feed := &RegistrationFeed{}
	for _, entry := range atomFeed.Entries {
		// Wrap each entry's inner content back into a full Atom entry for parseRegistrationResponse.
		wrappedXML := []byte(atomEntryHeader + string(entry.Content.InnerXML) + atomEntryFooter)
		reg, err := parseRegistrationResponse(wrappedXML)
		if err != nil {
			continue // skip unparseable entries
		}
		feed.Entries = append(feed.Entries, *reg)
	}
	return feed, nil
}

func splitTags(tags string) []string {
	if tags == "" {
		return nil
	}
	parts := strings.Split(tags, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}
