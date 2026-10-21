package mail

import (
	"fmt"
	"net/smtp"

	"github.com/jordan-wright/email"
)

type EmailSender interface {
	SendEmail(subject, content string, to, cc, bcc, attachFiles []string) error
}

type GmailSender struct {
	name     string
	address  string
	password string
}

func NewGmailSender(name, address, password string) EmailSender {
	return &GmailSender{name: name, address: address, password: password}
}

func (sender *GmailSender) SendEmail(subject, content string, to, cc, bcc, attachFiles []string) error {
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
	return message.Send("smtp.gmail.com:587", smtp.PlainAuth("", sender.address, sender.password, "smtp.gmail.com"))
}
