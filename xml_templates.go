package azurenh

import (
	"fmt"
	"strings"
)

// XML registration templates for the Azure NH legacy Registration API.
// Azure NH uses Atom XML feeds for registration management.

const atomEntryHeader = `<?xml version="1.0" encoding="utf-8"?>
<entry xmlns="http://www.w3.org/2005/Atom">
  <content type="application/xml">`

const atomEntryFooter = `
  </content>
</entry>`

const appleNativeRegistration = `
    <AppleRegistrationDescription xmlns:i="http://www.w3.org/2001/XMLSchema-instance" xmlns="http://schemas.microsoft.com/netservices/2010/10/servicebus/connect">
      <Tags>%s</Tags>
      <DeviceToken>%s</DeviceToken>
    </AppleRegistrationDescription>`

const appleTemplateRegistration = `
    <AppleTemplateRegistrationDescription xmlns:i="http://www.w3.org/2001/XMLSchema-instance" xmlns="http://schemas.microsoft.com/netservices/2010/10/servicebus/connect">
      <Tags>%s</Tags>
      <DeviceToken>%s</DeviceToken>
      <BodyTemplate><![CDATA[%s]]></BodyTemplate>
    </AppleTemplateRegistrationDescription>`

const fcmV1NativeRegistration = `
    <FcmV1RegistrationDescription xmlns:i="http://www.w3.org/2001/XMLSchema-instance" xmlns="http://schemas.microsoft.com/netservices/2010/10/servicebus/connect">
      <Tags>%s</Tags>
      <FcmV1RegistrationId>%s</FcmV1RegistrationId>
    </FcmV1RegistrationDescription>`

const fcmV1TemplateRegistration = `
    <FcmV1TemplateRegistrationDescription xmlns:i="http://www.w3.org/2001/XMLSchema-instance" xmlns="http://schemas.microsoft.com/netservices/2010/10/servicebus/connect">
      <Tags>%s</Tags>
      <FcmV1RegistrationId>%s</FcmV1RegistrationId>
      <BodyTemplate><![CDATA[%s]]></BodyTemplate>
    </FcmV1TemplateRegistrationDescription>`

const wnsNativeRegistration = `
    <WindowsRegistrationDescription xmlns:i="http://www.w3.org/2001/XMLSchema-instance" xmlns="http://schemas.microsoft.com/netservices/2010/10/servicebus/connect">
      <Tags>%s</Tags>
      <ChannelUri>%s</ChannelUri>
    </WindowsRegistrationDescription>`

const wnsTemplateRegistration = `
    <WindowsTemplateRegistrationDescription xmlns:i="http://www.w3.org/2001/XMLSchema-instance" xmlns="http://schemas.microsoft.com/netservices/2010/10/servicebus/connect">
      <Tags>%s</Tags>
      <ChannelUri>%s</ChannelUri>
      <BodyTemplate><![CDATA[%s]]></BodyTemplate>
    </WindowsTemplateRegistrationDescription>`

const admNativeRegistration = `
    <AdmRegistrationDescription xmlns:i="http://www.w3.org/2001/XMLSchema-instance" xmlns="http://schemas.microsoft.com/netservices/2010/10/servicebus/connect">
      <Tags>%s</Tags>
      <AdmRegistrationId>%s</AdmRegistrationId>
    </AdmRegistrationDescription>`

const admTemplateRegistration = `
    <AdmTemplateRegistrationDescription xmlns:i="http://www.w3.org/2001/XMLSchema-instance" xmlns="http://schemas.microsoft.com/netservices/2010/10/servicebus/connect">
      <Tags>%s</Tags>
      <AdmRegistrationId>%s</AdmRegistrationId>
      <BodyTemplate><![CDATA[%s]]></BodyTemplate>
    </AdmTemplateRegistrationDescription>`

const baiduNativeRegistration = `
    <BaiduRegistrationDescription xmlns:i="http://www.w3.org/2001/XMLSchema-instance" xmlns="http://schemas.microsoft.com/netservices/2010/10/servicebus/connect">
      <Tags>%s</Tags>
      <BaiduUserId>%s</BaiduUserId>
      <BaiduChannelId>%s</BaiduChannelId>
    </BaiduRegistrationDescription>`

const baiduTemplateRegistration = `
    <BaiduTemplateRegistrationDescription xmlns:i="http://www.w3.org/2001/XMLSchema-instance" xmlns="http://schemas.microsoft.com/netservices/2010/10/servicebus/connect">
      <Tags>%s</Tags>
      <BaiduUserId>%s</BaiduUserId>
      <BaiduChannelId>%s</BaiduChannelId>
      <BodyTemplate><![CDATA[%s]]></BodyTemplate>
    </BaiduTemplateRegistrationDescription>`

// buildRegistrationXML constructs the Atom XML body for a registration request.
func buildRegistrationXML(reg Registration) (string, error) {
	tags := strings.Join(reg.Tags, ",")

	var xmlBody string
	isTemplate := reg.Template != ""

	switch reg.Platform {
	case PlatformAPNS:
		if isTemplate {
			xmlBody = fmt.Sprintf(appleTemplateRegistration, tags, reg.DeviceToken, reg.Template)
		} else {
			xmlBody = fmt.Sprintf(appleNativeRegistration, tags, reg.DeviceToken)
		}
	case PlatformFCMV1:
		if isTemplate {
			xmlBody = fmt.Sprintf(fcmV1TemplateRegistration, tags, reg.DeviceToken, reg.Template)
		} else {
			xmlBody = fmt.Sprintf(fcmV1NativeRegistration, tags, reg.DeviceToken)
		}
	case PlatformWNS:
		if isTemplate {
			xmlBody = fmt.Sprintf(wnsTemplateRegistration, tags, reg.DeviceToken, reg.Template)
		} else {
			xmlBody = fmt.Sprintf(wnsNativeRegistration, tags, reg.DeviceToken)
		}
	case PlatformADM:
		if isTemplate {
			xmlBody = fmt.Sprintf(admTemplateRegistration, tags, reg.DeviceToken, reg.Template)
		} else {
			xmlBody = fmt.Sprintf(admNativeRegistration, tags, reg.DeviceToken)
		}
	default:
		return "", &ValidationError{Field: "platform", Message: fmt.Sprintf("unsupported registration platform: %q", reg.Platform)}
	}

	return atomEntryHeader + xmlBody + atomEntryFooter, nil
}
