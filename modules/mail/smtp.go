package mail

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/mail"
	"net/smtp"
	"strings"
	"time"

	"github.com/datvtph41107/bdspro/modules/setting"
)

var ErrUnavailable = errors.New("mail delivery unavailable")

func SendPasswordReset(ctx context.Context, destination, proof string) error {
	message, err := resetMessage(setting.Mailer.From, destination, proof)
	if err != nil {
		return err
	}
	return sendMessage(ctx, destination, message)
}

func SendRegistrationCode(ctx context.Context, destination, code string) error {
	if len(code) != 8 || strings.IndexFunc(code, func(r rune) bool { return r < '0' || r > '9' }) >= 0 {
		return ErrUnavailable
	}
	message, err := plainMessage(setting.Mailer.From, destination, "BDSPro email verification",
		"Enter this verification code in BDSPro: "+code+"\r\nIt expires in 5 minutes. If you did not request this, ignore this message.\r\n")
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
	conn, err := (&tls.Dialer{Config: &tls.Config{
		ServerName: host, MinVersion: tls.VersionTLS12,
	}}).DialContext(ctx, "tcp", m.Address)
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

func resetMessage(from, to, proof string) (string, error) {
	if len(proof) != 43 || strings.ContainsAny(proof, "\r\n") {
		return "", ErrUnavailable
	}
	return plainMessage(from, to, "BDSPro password recovery", "Enter this recovery token in BDSPro: "+proof+"\r\nIt expires in 15 minutes. If you did not request this, ignore this message.\r\n")
}

func plainMessage(from, to, subject, body string) (string, error) {
	for _, address := range []string{from, to} {
		parsed, err := mail.ParseAddress(address)
		if err != nil || parsed.Address != address || strings.ContainsAny(address, "\r\n") {
			return "", ErrUnavailable
		}
	}
	return fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s", from, to, subject, body), nil
}

func SendPasswordSetProof(ctx context.Context, destination, proof string) error {
	if len(proof) != 43 || strings.ContainsAny(proof, "\r\n") {
		return ErrUnavailable
	}
	message, err := plainMessage(setting.Mailer.From, destination, "BDSPro password setup",
		"Enter this proof only in your signed-in BDSPro password setup: "+proof+"\r\nIt expires in 10 minutes. Do not share it.\r\n")
	if err != nil {
		return err
	}
	return sendMessage(ctx, destination, message)
}

func SendCredentialNotice(ctx context.Context, destination string) error {
	message, err := plainMessage(setting.Mailer.From, destination, "BDSPro account security change",
		"A sign-in method on your BDSPro account changed. If you did not make this change, contact support.\r\n")
	if err != nil {
		return err
	}
	return sendMessage(ctx, destination, message)
}

func SendTeamInvitation(ctx context.Context, destination, proof string, expiresAt time.Time) error {
	if len(proof) != 43 || strings.ContainsAny(proof, "\r\n") {
		return ErrUnavailable
	}
	message, err := plainMessage(setting.Mailer.From, destination, "BDSPro team invitation",
		"Sign in to BDSPro with your invited account and enter this invitation token: "+proof+"\r\nExpires at "+expiresAt.UTC().Format(time.RFC3339)+".\r\nDo not share it.\r\n")
	if err != nil {
		return err
	}
	return sendMessage(ctx, destination, message)
}
