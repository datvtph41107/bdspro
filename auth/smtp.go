package auth

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strings"
)

// SMTPMailer requires STARTTLS; it will never transmit verification
// tokens over a plaintext SMTP channel. It does not store the token.
type SMTPMailer struct{ Addr, User, Password, From, VerificationURL string }

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
	w, err := c.Data()
	if err != nil {
		return err
	}
	defer w.Close()
	url := m.VerificationURL
	if strings.Contains(url, "?") {
		url += "&token=" + token
	} else {
		url += "?token=" + token
	}
	body := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: Verify your BDSPro email\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\nVerify your email:\r\n%s\r\n", m.From, to, url)
	_, err = w.Write([]byte(body))
	return err
}
