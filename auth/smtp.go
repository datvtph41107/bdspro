package auth

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"time"
)

// SMTPMailer requires STARTTLS; it will never transmit verification
// tokens over a plaintext SMTP channel. It does not store the token.
type SMTPMailer struct{ Addr, User, Password, From, VerificationURL string }

// NewSMTPMailer creates the optional SMTP sender for application startup.
func NewSMTPMailer(addr, user, password, from, verificationURL string) Mailer {
	if addr == "" {
		return nil
	}
	return SMTPMailer{
		Addr: addr, User: user, Password: password,
		From: from, VerificationURL: verificationURL,
	}
}

func (m SMTPMailer) SendVerification(ctx context.Context, to, token string) error {
	if m.Addr == "" || m.From == "" || m.VerificationURL == "" {
		return ErrMailerUnavailable
	}
	host, _, err := net.SplitHostPort(m.Addr)
	if err != nil {
		return err
	}
	dialer := net.Dialer{}
	conn, err := dialer.DialContext(ctx, "tcp", m.Addr)
	if err != nil {
		return err
	}
	// DialContext limits only connection establishment, not later SMTP reads/writes.
	deadline := time.Now().Add(15 * time.Second)
	if requestDeadline, ok := ctx.Deadline(); ok && requestDeadline.Before(deadline) {
		deadline = requestDeadline
	}
	if err := conn.SetDeadline(deadline); err != nil {
		_ = conn.Close()
		return err
	}
	c, err := smtp.NewClient(conn, host)
	if err != nil {
		_ = conn.Close()
		return err
	}
	defer c.Close()
	if ok, _ := c.Extension("STARTTLS"); !ok {
		return fmt.Errorf("smtp server requires STARTTLS")
	}
	if err := c.StartTLS(&tls.Config{ServerName: host, MinVersion: tls.VersionTLS12}); err != nil {
		return err
	}
	if m.User != "" {
		if err := c.Auth(smtp.PlainAuth("", m.User, m.Password, host)); err != nil {
			return err
		}
	}
	if err := c.Mail(m.From); err != nil {
		return err
	}
	if err := c.Rcpt(to); err != nil {
		return err
	}
	url := m.VerificationURL
	if strings.Contains(url, "?") {
		url += "&token=" + token
	} else {
		url += "?token=" + token
	}
	body := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: Verify your BDSPro email\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\nVerify your email:\r\n%s\r\n", m.From, to, url)
	return writeSMTPData(c, body)
}

// writeSMTPData must check Close: the SMTP server's final DATA acceptance
// arrives only after the message terminator, not after Write succeeds.
func writeSMTPData(c *smtp.Client, body string) error {
	w, err := c.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write([]byte(body)); err != nil {
		_ = w.Close()
		return err
	}
	return w.Close()
}
