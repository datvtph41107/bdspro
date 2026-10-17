package auth

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewSMTPMailer(t *testing.T) {
	require.Nil(t, NewSMTPMailer("", "", "", "", ""))

	mailer := NewSMTPMailer("smtp.example.test:587", "mail-user", "secret", "hello@example.test", "https://example.test/verify")
	require.Equal(t, SMTPMailer{
		Addr: "smtp.example.test:587",
		User: "mail-user",
		Password: "secret",
		From: "hello@example.test",
		VerificationURL: "https://example.test/verify",
	}, mailer)
}
