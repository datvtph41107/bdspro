package setting

import (
	"fmt"
	"strconv"
)

var Mailer struct {
	EmailSenderName     string
	EmailSenderAddress  string
	EmailSenderPassword string
	SMTPAddress         string
	SMTPAuth            bool
	VerifyEmailURL      string
}

func loadMailerFrom(rootCfg ConfigProvider) error {
	sec := rootCfg.Section("mailer")
	auth, err := strconv.ParseBool(stringOrDefault(sec.Key("SMTP_AUTH"), "false"))
	if err != nil {
		return fmt.Errorf("SMTP_AUTH: %w", err)
	}
	Mailer.EmailSenderName = sec.Key("EMAIL_SENDER_NAME").String()
	Mailer.EmailSenderAddress = sec.Key("EMAIL_SENDER_ADDRESS").String()
	Mailer.EmailSenderPassword = sec.Key("EMAIL_SENDER_PASSWORD").String()
	Mailer.SMTPAddress = sec.Key("SMTP_ADDRESS").String()
	Mailer.SMTPAuth = auth
	Mailer.VerifyEmailURL = sec.Key("VERIFY_EMAIL_URL").String()
	return nil
}

func LoadMailerSetting() error {
	return loadMailerFrom(CfgProvider)
}
