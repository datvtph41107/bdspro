package setting

import (
	"strings"
	"testing"
)

func TestPhoneDeliveryConfiguration(t *testing.T) {
	mailer := MailerSettings{Address: "127.0.0.1:1025", Protocol: "smtp"}
	for _, tc := range []struct {
		name, mode, ini string
		wantErr         bool
	}{
		{"disabled_default", "production", "", false},
		{"local_inbox", "development", "[phone_delivery]\nMODE=mailpit\nCHANNELS=sms,zalo,telegram\n", false},
		{"production_inbox", "production", "[phone_delivery]\nMODE=mailpit\n", true},
		{"unknown_channel", "development", "[phone_delivery]\nMODE=mailpit\nCHANNELS=whatsapp\n", true},
		{"duplicate_channel", "development", "[phone_delivery]\nMODE=mailpit\nCHANNELS=sms,sms\n", true},
		{"default_not_enabled", "development", "[phone_delivery]\nMODE=mailpit\nCHANNELS=zalo\n", true},
		{"missing_sms_config", "production", "[phone_delivery]\nMODE=live\n", true},
		{"missing_zalo_config", "production", "[phone_delivery]\nMODE=live\nCHANNELS=zalo\nDEFAULT_CHANNEL=zalo\n", true},
		{"gateway", "production", "[phone_delivery]\nMODE=live\nCHANNELS=telegram\nDEFAULT_CHANNEL=telegram\n[telegram_gateway]\nACCESS_TOKEN=01234567890123456789\n", false},
		{"zalo", "production", "[phone_delivery]\nMODE=live\nCHANNELS=zalo\nDEFAULT_CHANNEL=zalo\n[zalo]\nACCESS_TOKEN=01234567890123456789\nTEMPLATE_ID=approved123\nOTP_PARAM=otp\n", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := NewConfigProviderFromData(tc.ini)
			if err != nil {
				t.Fatal(err)
			}
			if err := validateKnownKeys(cfg); err != nil {
				t.Fatal(err)
			}
			_, err = loadPhoneDeliveryFrom(cfg, tc.mode, mailer, SMSSettings{})
			if (err != nil) != tc.wantErr {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
	cfg, _ := NewConfigProviderFromData("[phone_delivery]\nMODE=mailpit\n")
	for _, address := range []string{"smtp.example.com:25", "127.0.0.1:1025"} {
		m := MailerSettings{Address: address, Protocol: "smtps"}
		if _, err := loadPhoneDeliveryFrom(cfg, "development", m, SMSSettings{}); err == nil {
			t.Fatal("non-Mailpit transport accepted")
		}
	}
	legacy, _ := NewConfigProviderFromData("")
	got, err := loadPhoneDeliveryFrom(legacy, "production", MailerSettings{}, SMSSettings{AccountSID: "AC" + strings.Repeat("a", 32)})
	if err != nil || got.Mode != "live" || !got.Allows("sms") {
		t.Fatalf("legacy SMS config broken: %+v %v", got, err)
	}
}

func TestPhoneDeliveryConfigCopiesChannels(t *testing.T) {
	cfg := PhoneDeliverySettings{Channels: []string{"sms"}}
	copy := cfg.clone()
	copy.Channels[0] = "zalo"
	if cfg.Channels[0] != "sms" {
		t.Fatal("mutable configuration shared")
	}
}
