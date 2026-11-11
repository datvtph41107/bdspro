package setting

var Phone struct {
	Delivery        string
	SMSGatewayURL   string
	SMSGatewayToken string
	ZaloAccessToken string
	ZaloTemplateID  string
	ZaloCodeField   string
}

func loadPhoneFrom(rootCfg ConfigProvider) {
	sec := rootCfg.Section("phone")
	Phone.Delivery = sec.Key("PHONE_DELIVERY").String()
	Phone.SMSGatewayURL = sec.Key("SMS_GATEWAY_URL").String()
	Phone.SMSGatewayToken = sec.Key("SMS_GATEWAY_TOKEN").String()
	Phone.ZaloAccessToken = sec.Key("ZALO_ACCESS_TOKEN").String()
	Phone.ZaloTemplateID = sec.Key("ZALO_TEMPLATE_ID").String()
	Phone.ZaloCodeField = stringOrDefault(sec.Key("ZALO_CODE_FIELD"), "otp")
}
