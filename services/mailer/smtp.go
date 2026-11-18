// Package mailer implements SMTP delivery without owning account or queue semantics.
package mailer

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"net/smtp"
	"strconv"
	"strings"
	"time"
)

var (
	ErrInvalidMessage = errors.New("mailer: invalid message")
	ErrUnsafeSMTP     = errors.New("mailer: SMTP requires TLS")
)

type Config struct {
	Address  string
	FromName string
	From     string
	Username string
	Password string
	Auth     bool
	Timeout  time.Duration
}

type Message struct {
	To      string
	Subject string
	Body    string // Plain text only. Templates and verification links belong to services.
}

// Sender is a transport resource, not a domain notifier or durable queue.
type Sender struct{ config Config }

func NewSender(cfg Config) (*Sender, error) {
	host, port, err := net.SplitHostPort(cfg.Address)
	if err != nil || host == "" || port == "" {
		return nil, fmt.Errorf("mailer: invalid SMTP address")
	}
	if _, err := strconv.Atoi(port); err != nil {
		return nil, fmt.Errorf("mailer: invalid SMTP port")
	}
	if err := validAddress(cfg.From); err != nil {
		return nil, fmt.Errorf("mailer: invalid sender address: %w", err)
	}
	if hasControl(cfg.FromName) || cfg.Timeout < 0 {
		return nil, ErrInvalidMessage
	}
	if cfg.Auth && (cfg.Username == "" || cfg.Password == "") {
		return nil, errors.New("mailer: SMTP authentication credentials missing")
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 10 * time.Second
	}
	return &Sender{config: cfg}, nil
}

func (s *Sender) Send(ctx context.Context, m Message) error {
	if s == nil {
		return errors.New("mailer: sender not initialized")
	}
	if ctx == nil {
		return errors.New("mailer: nil context")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := validAddress(m.To); err != nil {
		return fmt.Errorf("%w: recipient address", ErrInvalidMessage)
	}
	if m.Subject == "" || hasControl(m.Subject) || len(m.Subject) > 512 ||
		len(m.Body) > 1024*1024 {
		return ErrInvalidMessage
	}

	payload, err := s.encode(m)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, s.config.Timeout)
	defer cancel()

	dialer := &net.Dialer{}
	conn, err := dialer.DialContext(ctx, "tcp", s.config.Address)
	if err != nil {
		return fmt.Errorf("mailer: connect SMTP: %w", err)
	}
	defer conn.Close()
	// Protect handshake and writes even without a caller deadline.
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}
	// Cancel in-flight socket reads/writes immediately when caller stops.
	stop := context.AfterFunc(ctx, func() { _ = conn.Close() })
	defer stop()

	host, _, _ := net.SplitHostPort(s.config.Address)
	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return fmt.Errorf("mailer: SMTP greeting: %w", err)
	}
	defer client.Close()

	if err := client.Hello("localhost"); err != nil {
		return fmt.Errorf("mailer: SMTP hello: %w", err)
	}
	tlsActive := false
	if ok, _ := client.Extension("STARTTLS"); ok {
		if err := client.StartTLS(&tls.Config{
			MinVersion: tls.VersionTLS12,
			ServerName: host,
		}); err != nil {
			return fmt.Errorf("mailer: STARTTLS: %w", err)
		}
		tlsActive = true
	}
	if !tlsActive && !isLoopback(host) {
		return ErrUnsafeSMTP
	}
	if s.config.Auth {
		// Never send credentials on plaintext SMTP, even to localhost.
		if !tlsActive {
			return ErrUnsafeSMTP
		}
		if err := client.Auth(smtp.PlainAuth("", s.config.Username, s.config.Password, host)); err != nil {
			return fmt.Errorf("mailer: authentication: %w", err)
		}
	}
	if err := client.Mail(s.config.From); err != nil {
		return fmt.Errorf("mailer: MAIL FROM: %w", err)
	}
	if err := client.Rcpt(m.To); err != nil {
		return fmt.Errorf("mailer: RCPT TO: %w", err)
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("mailer: DATA: %w", err)
	}
	if _, err := io.Copy(w, bytes.NewReader(payload)); err != nil {
		_ = w.Close()
		return fmt.Errorf("mailer: write DATA: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("mailer: finish DATA: %w", err)
	}
	if err := client.Quit(); err != nil {
		return fmt.Errorf("mailer: QUIT: %w", err)
	}
	return nil
}

func (s *Sender) encode(m Message) ([]byte, error) {
	var b bytes.Buffer
	from := (&mail.Address{Name: s.config.FromName, Address: s.config.From}).String()
	to := (&mail.Address{Address: m.To}).String()
	b.WriteString("From: " + from + "\r\n")
	b.WriteString("To: " + to + "\r\n")
	b.WriteString("Subject: " + mime.QEncoding.Encode("utf-8", m.Subject) + "\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	b.WriteString("Content-Transfer-Encoding: quoted-printable\r\n")
	b.WriteString("\r\n")
	writer := quotedprintable.NewWriter(&b)
	if _, err := io.WriteString(writer, m.Body); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func validAddress(value string) error {
	if value == "" || len(value) > 254 || hasControl(value) {
		return ErrInvalidMessage
	}
	addr, err := mail.ParseAddress(value)
	if err != nil || addr.Address != value || addr.Name != "" ||
		!strings.Contains(value, "@") {
		return ErrInvalidMessage
	}
	return nil
}

func hasControl(value string) bool {
	return strings.ContainsAny(value, "\r\n\x00")
}

func isLoopback(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
