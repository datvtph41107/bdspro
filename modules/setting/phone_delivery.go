package setting

import (
	"errors"
	"net"
	"regexp"
	"slices"
	"strings"
)

type PhoneDeliverySettings struct {
	Mode                string
	DefaultChannel      string
	Channels            []string
	ZaloAccessToken     string
	ZaloTemplateID      string
	ZaloOTPParam        string
	TelegramAccessToken string
}

var PhoneDelivery PhoneDeliverySettings

func (s PhoneDeliverySettings) Allows(channel string) bool {
	return s.Mode != "disabled" && slices.Contains(s.Channels, channel)
}

func (s PhoneDeliverySettings) clone() PhoneDeliverySettings {
	s.Channels = append([]string(nil), s.Channels...)
	return s
}

func loadPhoneDeliveryFrom(cfg ConfigProvider, runMode string, mailer MailerSettings, sms SMSSettings) (PhoneDeliverySettings, error) {
	sec := cfg.Section("phone_delivery")
	legacyMode := "disabled"
	if sms.Enabled() {
		legacyMode = "live"
	}
	s := PhoneDeliverySettings{
		Mode:                stringOrDefault(sec.Key("MODE"), legacyMode),
		DefaultChannel:      stringOrDefault(sec.Key("DEFAULT_CHANNEL"), "sms"),
		ZaloAccessToken:     cfg.Section("zalo").Key("ACCESS_TOKEN").String(),
		ZaloTemplateID:      cfg.Section("zalo").Key("TEMPLATE_ID").String(),
		ZaloOTPParam:        stringOrDefault(cfg.Section("zalo").Key("OTP_PARAM"), "otp"),
		TelegramAccessToken: cfg.Section("telegram_gateway").Key("ACCESS_TOKEN").String(),
	}
	if s.Mode != "disabled" && s.Mode != "mailpit" && s.Mode != "live" {
		return s, errors.New("phone_delivery MODE must be disabled, mailpit or live")
	}
	for _, channel := range strings.Split(stringOrDefault(sec.Key("CHANNELS"), "sms"), ",") {
		channel = strings.TrimSpace(channel)
		if channel != "sms" && channel != "zalo" && channel != "telegram" {
			return s, errors.New("phone_delivery CHANNELS must contain only sms, zalo or telegram")
		}
		if slices.Contains(s.Channels, channel) {
			return s, errors.New("duplicate phone delivery channel")
		}
		s.Channels = append(s.Channels, channel)
	}
	if !slices.Contains(s.Channels, s.DefaultChannel) {
		return s, errors.New("phone_delivery DEFAULT_CHANNEL must be in CHANNELS")
	}
	if s.Mode == "mailpit" {
		host, _, err := net.SplitHostPort(mailer.Address)
		ip := net.ParseIP(host)
		local := host == "localhost" || host == "mailpit" || ip != nil && ip.IsLoopback()
		if runMode != "development" || !mailer.Enabled() || mailer.Protocol != "smtp" || !local || err != nil {
			return s, errors.New("mailpit phone delivery requires development and local plain SMTP mailer (loopback or mailpit)")
		}
	}
	if s.Mode == "live" {
		if s.Allows("sms") && !sms.Enabled() {
			return s, errors.New("sms channel requires Twilio credentials in [sms]")
		}
		if s.Allows("zalo") && (!validDeliverySecret(s.ZaloAccessToken) ||
			!regexp.MustCompile(`^[A-Za-z0-9_-]{1,128}$`).MatchString(s.ZaloTemplateID) ||
			!regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]{0,63}$`).MatchString(s.ZaloOTPParam)) {
			return s, errors.New("zalo channel requires ACCESS_TOKEN, approved TEMPLATE_ID and OTP_PARAM")
		}
		if s.Allows("telegram") && !validDeliverySecret(s.TelegramAccessToken) {
			return s, errors.New("telegram channel requires Telegram Gateway ACCESS_TOKEN (not a bot token)")
		}
	}
	return s, nil
}

func validDeliverySecret(value string) bool {
	return len(value) >= 16 && len(value) <= 8192 && !strings.ContainsAny(value, " \r\n\t")
}
