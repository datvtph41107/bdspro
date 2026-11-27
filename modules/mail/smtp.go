// Package mail sends plain email through the configured SMTP transport.
package mail

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"net/smtp"
	"time"

	"github.com/datvtph41107/bdspro/modules/setting"
)

var ErrUnavailable = errors.New("mail delivery unavailable")

// SendPlain sends a plain-text email using the configured sender and SMTP transport.
func SendPlain(ctx context.Context, destination, subject, body string) error {
	message, err := plainMessage(setting.Mailer.From, destination, subject, body)
	if err != nil {
		return err
	}
	return sendMessage(ctx, destination, message)
}

func sendMessage(ctx context.Context, destination, message string) error {
	m := setting.Mailer
	if !m.Enabled() {
		return ErrUnavailable
	}
	host, _, err := net.SplitHostPort(m.Address)
	if err != nil {
		return ErrUnavailable
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	var conn net.Conn
	switch m.Protocol {
	case "smtp":
		if setting.Common.RunMode != "development" || m.Username != "" || m.Password != "" {
			return ErrUnavailable
		}
		conn, err = (&net.Dialer{}).DialContext(ctx, "tcp", m.Address)
	case "", "smtps":
		conn, err = (&tls.Dialer{Config: &tls.Config{
			ServerName: host, MinVersion: tls.VersionTLS12,
		}}).DialContext(ctx, "tcp", m.Address)
	default:
		return ErrUnavailable
	}
	if err != nil {
		return ErrUnavailable
	}
	defer conn.Close()
	deadline, _ := ctx.Deadline()
	if err := conn.SetDeadline(deadline); err != nil {
		return ErrUnavailable
	}
	stop := context.AfterFunc(ctx, func() { _ = conn.Close() })
	defer stop()
	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return ErrUnavailable
	}
	defer client.Close()
	if m.Username != "" {
		if err := client.Auth(smtp.PlainAuth("", m.Username, m.Password, host)); err != nil {
			return ErrUnavailable
		}
	}
	if err := client.Mail(m.From); err != nil {
		return ErrUnavailable
	}
	if err := client.Rcpt(destination); err != nil {
		return ErrUnavailable
	}
	writer, err := client.Data()
	if err != nil {
		return ErrUnavailable
	}
	if _, err := writer.Write([]byte(message)); err != nil {
		return ErrUnavailable
	}
	if err := writer.Close(); err != nil {
		return ErrUnavailable
	}
	return nil
}
