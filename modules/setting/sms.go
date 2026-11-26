package setting

import (
	"errors"
	"regexp"
	"strings"
)

type SMSSettings struct {
	AccountSID string
	AuthToken  string
	SenderID   string
}

var SMS SMSSettings

func (s SMSSettings) Enabled() bool { return s.AccountSID != "" }

var twilioAccountSID = regexp.MustCompile(`^AC[0-9a-fA-F]{32}$`)
var smsSenderID = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9 ]{0,10}$`)

func loadSMSFrom(cfg ConfigProvider) (SMSSettings, error) {
	sec := cfg.Section("sms")
	s := SMSSettings{
		AccountSID: sec.Key("ACCOUNT_SID").String(),
		AuthToken:  sec.Key("AUTH_TOKEN").String(),
		SenderID:   sec.Key("SENDER_ID").String(),
	}
	if s == (SMSSettings{}) {
		return s, nil
	}
	if !twilioAccountSID.MatchString(s.AccountSID) || len(s.AuthToken) < 16 ||
		strings.ContainsAny(s.AuthToken, " \r\n\t") || !smsSenderID.MatchString(s.SenderID) ||
		!strings.ContainsAny(s.SenderID, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ") ||
		strings.TrimSpace(s.SenderID) != s.SenderID {
		return SMSSettings{}, errors.New("invalid sms configuration: account SID, auth token and registered sender ID required")
	}
	return s, nil
}
