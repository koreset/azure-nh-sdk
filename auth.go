package azurenh

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/url"
	"strings"
	"time"
)

// parseConnectionString parses an Azure NH connection string and returns
// the endpoint URL, shared access key name, and shared access key value.
// Connection string format:
//
//	Endpoint=sb://<namespace>.servicebus.windows.net/;SharedAccessKeyName=<name>;SharedAccessKey=<key>
func parseConnectionString(connectionString string) (endpoint *url.URL, keyName, keyValue string, err error) {
	parts := make(map[string]string)
	for _, segment := range strings.Split(connectionString, ";") {
		segment = strings.TrimSpace(segment)
		if segment == "" {
			continue
		}
		idx := strings.Index(segment, "=")
		if idx < 0 {
			continue
		}
		key := segment[:idx]
		value := segment[idx+1:]
		parts[key] = value
	}

	rawEndpoint, ok := parts["Endpoint"]
	if !ok || rawEndpoint == "" {
		return nil, "", "", fmt.Errorf("%w: missing Endpoint", ErrInvalidConnectionString)
	}

	keyName, ok = parts["SharedAccessKeyName"]
	if !ok || keyName == "" {
		return nil, "", "", fmt.Errorf("%w: missing SharedAccessKeyName", ErrInvalidConnectionString)
	}

	keyValue, ok = parts["SharedAccessKey"]
	if !ok || keyValue == "" {
		return nil, "", "", fmt.Errorf("%w: missing SharedAccessKey", ErrInvalidConnectionString)
	}

	endpoint, err = url.Parse(rawEndpoint)
	if err != nil {
		return nil, "", "", fmt.Errorf("%w: invalid Endpoint URL: %v", ErrInvalidConnectionString, err)
	}

	return endpoint, keyName, keyValue, nil
}

// generateSASToken generates a Shared Access Signature token for Azure Service Bus.
func (c *Client) generateSASToken() string {
	now := c.clock.Now()
	expiry := now.Add(time.Duration(defaultSASTokenTTL) * time.Second)
	expiryStr := fmt.Sprintf("%d", expiry.Unix())

	targetURI := strings.ToLower(c.hubURL.String())
	encodedURI := url.QueryEscape(targetURI)

	toSign := encodedURI + "\n" + expiryStr

	mac := hmac.New(sha256.New, []byte(c.sasKeyValue))
	mac.Write([]byte(toSign))
	signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	return fmt.Sprintf(
		"SharedAccessSignature sr=%s&sig=%s&se=%s&skn=%s",
		encodedURI,
		url.QueryEscape(signature),
		expiryStr,
		c.sasKeyName,
	)
}
