package mail

import (
	"fmt"
	"net"
	"net/smtp"

	"github.com/jordan-wright/email"
)

// EmailSender is implemented by both the real SMTP adapter and test fakes.
type EmailSender interface {
	SendEmail(subject, content string, to, cc, bcc, attachFiles []string) error
}

// SMTPSender sends the same message to a configured SMTP endpoint.
// Authentication is opt-in so local Mailpit can receive without credentials.
type SMTPSender struct {
	name     string
	address  string
	password string
	smtpAddr string
	auth     bool
}

func NewSMTPSender(name, address, password, smtpAddress string, auth bool) EmailSender {
	return &SMTPSender{
		name: name, address: address, password: password,
		smtpAddr: smtpAddress, auth: auth,
	}
}

// NewGmailSender remains available for Gmail-specific callers.
func NewGmailSender(name, address, password string) EmailSender {
	return NewSMTPSender(name, address, password, "smtp.gmail.com:587", true)
}

func (sender *SMTPSender) SendEmail(subject, content string, to, cc, bcc, attachFiles []string) error {
	message := email.NewEmail()
	message.From = fmt.Sprintf("%s <%s>", sender.name, sender.address)
	message.Subject = subject
	message.HTML = []byte(content)
	message.To = to
	message.Cc = cc
	message.Bcc = bcc
	for _, file := range attachFiles {
		if _, err := message.AttachFile(file); err != nil {
			return fmt.Errorf("attach %s: %w", file, err)
		}
	}
	if sender.smtpAddr == "" {
		return fmt.Errorf("SMTP_ADDRESS is required")
	}
	var auth smtp.Auth
	if sender.auth {
		host, _, err := net.SplitHostPort(sender.smtpAddr)
		if err != nil {
			return fmt.Errorf("invalid SMTP_ADDRESS: %w", err)
		}
		if sender.address == "" || sender.password == "" {
			return fmt.Errorf("SMTP authentication requires sender address and password")
		}
		auth = smtp.PlainAuth("", sender.address, sender.password, host)
	}
	return message.Send(sender.smtpAddr, auth)
}
