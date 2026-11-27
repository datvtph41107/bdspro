package mail

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/datvtph41107/bdspro/modules/setting"
)

func TestRegistrationDeliveryOverDevelopmentSMTP(t *testing.T) {
	previousMailer, previousCommon := setting.Mailer, setting.Common
	t.Cleanup(func() { setting.Mailer, setting.Common = previousMailer, previousCommon })
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	done := make(chan error, 1)
	received := make(chan string, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			done <- err
			return
		}
		defer conn.Close()
		_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
		reader := bufio.NewReader(conn)
		_, _ = fmt.Fprint(conn, "220 mailpit-test ESMTP\r\n")
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				done <- err
				return
			}
			switch {
			case strings.HasPrefix(line, "EHLO "), strings.HasPrefix(line, "HELO "), strings.HasPrefix(line, "MAIL FROM:"), strings.HasPrefix(line, "RCPT TO:"):
				_, _ = fmt.Fprint(conn, "250 OK\r\n")
			case line == "DATA\r\n":
				_, _ = fmt.Fprint(conn, "354 Send message\r\n")
				var body strings.Builder
				for {
					line, err = reader.ReadString('\n')
					if err != nil {
						done <- err
						return
					}
					if line == ".\r\n" {
						break
					}
					body.WriteString(line)
				}
				received <- body.String()
				_, _ = fmt.Fprint(conn, "250 Accepted\r\n")
				done <- nil
				return
			default:
				done <- fmt.Errorf("unexpected SMTP command: %q", line)
				return
			}
		}
	}()
	setting.Common.RunMode = "development"
	setting.Mailer = setting.MailerSettings{Protocol: "smtp", Address: listener.Addr().String(), From: "security@bdspro.test"}
	if err := SendRegistrationCode(context.Background(), "test@example.invalid", "12345678"); err != nil {
		t.Fatal(err)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	body := <-received
	if !strings.Contains(body, "To: test@example.invalid") || !strings.Contains(body, "12345678") {
		t.Fatalf("unexpected delivered email: %q", body)
	}

	setting.Common.RunMode = "production"
	if err := SendRegistrationCode(context.Background(), "test@example.invalid", "12345678"); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("plain SMTP allowed in production: %v", err)
	}
}
