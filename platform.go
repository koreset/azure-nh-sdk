package azurenh

// Platform represents the push notification service platform.
type Platform string

const (
	PlatformAPNS  Platform = "apns"
	PlatformFCMV1 Platform = "fcmv1"
	PlatformWNS   Platform = "wns"
	PlatformADM   Platform = "adm"
	PlatformBaidu Platform = "baidu"
)

// IsValid returns true if the platform is a recognized value.
func (p Platform) IsValid() bool {
	switch p {
	case PlatformAPNS, PlatformFCMV1, PlatformWNS, PlatformADM, PlatformBaidu:
		return true
	}
	return false
}

// NotificationFormat identifies the PNS-specific format for a notification.
type NotificationFormat string

const (
	FormatApple    NotificationFormat = "apple"
	FormatFCMV1    NotificationFormat = "fcmv1"
	FormatWindows  NotificationFormat = "windows"
	FormatADM      NotificationFormat = "adm"
	FormatBaidu    NotificationFormat = "baidu"
	FormatTemplate NotificationFormat = "template"
)

// ContentType returns the HTTP Content-Type for the notification format.
func (f NotificationFormat) ContentType() string {
	if f == FormatWindows {
		return "application/xml"
	}
	return "application/json"
}

// IsValid returns true if the format is a recognized value.
func (f NotificationFormat) IsValid() bool {
	switch f {
	case FormatApple, FormatFCMV1, FormatWindows, FormatADM, FormatBaidu, FormatTemplate:
		return true
	}
	return false
}

// ServiceBusHeader returns the ServiceBusNotification-Format header value.
func (f NotificationFormat) ServiceBusHeader() string {
	return string(f)
}
