package setting

import (
	"errors"
	"net"
	"net/mail"
	"strconv"
	"strings"
)

type MailerSettings struct {
	Address  string
	From     string
	Username string
	Password string
}

var Mailer MailerSettings

func (m MailerSettings) Enabled() bool { return m.Address != "" }

func loadMailerFrom(cfg ConfigProvider) (MailerSettings, error) {
	sec := cfg.Section("mailer")
	m := MailerSettings{
		Address:  sec.Key("SMTP_ADDRESS").String(),
		From:     sec.Key("FROM").String(),
		Username: sec.Key("USERNAME").String(),
		Password: sec.Key("PASSWORD").String(),
	}
	if !m.Enabled() {
		if m.From != "" || m.Username != "" || m.Password != "" {
			return MailerSettings{}, errors.New("mailer SMTP_ADDRESS is required when mailer is configured")
		}
		return m, nil
	}
	host, port, err := net.SplitHostPort(m.Address)
	n, portErr := strconv.Atoi(port)
	if err != nil || host == "" || portErr != nil || n < 1 || n > 65535 || strings.ContainsAny(host, " \r\n\t") {
		return MailerSettings{}, errors.New("invalid mailer SMTP_ADDRESS")
	}
	address, err := mail.ParseAddress(m.From)
	if err != nil || address.Address != m.From || strings.ContainsAny(m.From, "\r\n") {
		return MailerSettings{}, errors.New("mailer FROM must be a plain email address")
	}
	if (m.Username == "") != (m.Password == "") || strings.ContainsAny(m.Username, "\r\n") {
		return MailerSettings{}, errors.New("mailer credentials must be supplied together")
	}
	return m, nil
}
