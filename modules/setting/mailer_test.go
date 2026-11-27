package setting

import "testing"

func TestMailerTransportConfiguration(t *testing.T) {
	for _, tc := range []struct {
		name, mode, extra, protocol string
		wantErr                     bool
	}{
		{"default_tls", "production", "", "smtps", false},
		{"mailpit", "development", "SMTP_PROTOCOL = smtp\n", "smtp", false},
		{"plaintext_production", "production", "SMTP_PROTOCOL = smtp\n", "", true},
		{"plaintext_credentials", "development", "SMTP_PROTOCOL = smtp\nUSERNAME = user\nPASSWORD = secret\n", "", true},
		{"unknown_protocol", "development", "SMTP_PROTOCOL = other\n", "", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := NewConfigProviderFromData("[mailer]\nSMTP_ADDRESS = 127.0.0.1:1025\nFROM = security@bdspro.test\n" + tc.extra)
			if err != nil {
				t.Fatal(err)
			}
			if err := validateKnownKeys(cfg); err != nil {
				t.Fatal(err)
			}
			got, err := loadMailerFrom(cfg, tc.mode)
			if (err != nil) != tc.wantErr {
				t.Fatalf("unexpected error: %v", err)
			}
			if !tc.wantErr && got.Protocol != tc.protocol {
				t.Fatalf("protocol=%q", got.Protocol)
			}
		})
	}
}
