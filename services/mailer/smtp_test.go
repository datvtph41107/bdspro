package mailer

import (
	"bufio"
	"context"
	"errors"
	"io"
	"net"
	"strings"
	"testing"
	"time"
)

// smtpFixture is a minimal SMTP server that verifies the wire protocol.
// It does not depend on real SMTP credentials or an external provider.
func smtpFixture(t *testing.T) (address string, result <-chan string, closeServer func()) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	resultCh := make(chan string, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		reader := bufio.NewReader(conn)
		writer := bufio.NewWriter(conn)
		reply := func(s string) {
			_, _ = writer.WriteString(s)
			_ = writer.Flush()
		}
		reply("220 local.example ESMTP ready\r\n")
		var body strings.Builder
		inData := false
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				return
			}
			if inData {
				if line == ".\r\n" {
					inData = false
					resultCh <- body.String()
					reply("250 2.0.0 message accepted\r\n")
					continue
				}
				body.WriteString(line)
				continue
			}
			switch {
			case strings.HasPrefix(line, "EHLO "):
				reply("250-local.example\r\n250 8BITMIME\r\n")
			case strings.HasPrefix(line, "HELO "):
				reply("250 local.example\r\n")
			case strings.HasPrefix(line, "MAIL FROM:"):
				reply("250 ok\r\n")
			case strings.HasPrefix(line, "RCPT TO:"):
				reply("250 ok\r\n")
			case strings.HasPrefix(line, "DATA"):
				inData = true
				reply("354 end with .\r\n")
			case strings.HasPrefix(line, "QUIT"):
				reply("221 closing\r\n")
				return
			default:
				reply("500 unexpected command\r\n")
				return
			}
		}
	}()
	return listener.Addr().String(), resultCh, func() { _ = listener.Close() }
}

func TestSMTPDeliveryToLocalDisposableServer(t *testing.T) {
	address, messages, stop := smtpFixture(t)
	defer stop()
	sender, err := NewSender(Config{
		Address: address, From: "no-reply@example.invalid",
		FromName: "BDSPro", Timeout: 2 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := sender.Send(context.Background(), Message{
		To: "recipient@example.invalid",
		Subject: "Verify your BDSPro account",
		Body: "Click the verification link.\nThis is a test.",
	}); err != nil {
		t.Fatalf("send: %v", err)
	}
	select {
	case got := <-messages:
		for _, expected := range []string{
			"From: BDSPro <no-reply@example.invalid>",
			"To: <recipient@example.invalid>",
			"Content-Type: text/plain; charset=UTF-8",
			"Verify your BDSPro account",
			"Click the verification link.",
		} {
			if !strings.Contains(got, expected) {
				t.Fatalf("missing %q in SMTP data %q", expected, got)
			}
		}
	case <-time.After(2 * time.Second):
		t.Fatal("SMTP message not delivered")
	}
}

func TestSMTPRejectsHeaderInjectionAndBadAddresses(t *testing.T) {
	if _, err := NewSender(Config{Address: "127.0.0.1:1025", From: "sender@example.com\r\nBcc: bad@example.com"}); err == nil {
		t.Fatal("injected sender accepted")
	}
	sender, err := NewSender(Config{Address: "127.0.0.1:1025", From: "sender@example.com"})
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range []Message{
		{To: "a@example.com\r\nBcc: attacker@example.com", Subject: "Hi", Body: "text"},
		{To: "recipient@example.com", Subject: "subject\nBcc: attacker@example.com", Body: "text"},
		{To: "recipient@example.com", Subject: "", Body: "text"},
	} {
		if err := sender.Send(context.Background(), m); !errors.Is(err, ErrInvalidMessage) {
			t.Fatalf("invalid message accepted: %v", err)
		}
	}
}

func TestSMTPRejectsPlaintextPasswordAuth(t *testing.T) {
	address, _, stop := smtpFixture(t)
	defer stop()
	sender, err := NewSender(Config{
		Address: address, From: "sender@example.com",
		Auth: true, Username: "user", Password: "secret",
		Timeout: time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	err = sender.Send(context.Background(), Message{
		To: "recipient@example.com", Subject: "Test", Body: "body",
	})
	if !errors.Is(err, ErrUnsafeSMTP) {
		t.Fatalf("expected fail closed without STARTTLS, got %v", err)
	}
}

func TestSMTPHonorsCanceledContext(t *testing.T) {
	sender, err := NewSender(Config{Address: "127.0.0.1:1025", From: "sender@example.com"})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err = sender.Send(ctx, Message{To: "recipient@example.com", Subject: "test", Body: "ok"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancellation, got %v", err)
	}
}

var _ = io.EOF
