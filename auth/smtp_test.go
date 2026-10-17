package auth

import (
	"bufio"
	"net"
	"net/smtp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewSMTPMailer(t *testing.T) {
	require.Nil(t, NewSMTPMailer("", "", "", "", ""))

	mailer := NewSMTPMailer("smtp.example.test:587", "mail-user", "secret", "hello@example.test", "https://example.test/verify")
	require.Equal(t, SMTPMailer{
		Addr:            "smtp.example.test:587",
		User:            "mail-user",
		Password:        "secret",
		From:            "hello@example.test",
		VerificationURL: "https://example.test/verify",
	}, mailer)
}

func TestWriteSMTPDataChecksFinalServerResponse(t *testing.T) {
	cases := []struct {
		name     string
		response string
		rejected bool
	}{
		{"accepted", "250 queued\\r\\n", false},
		{"rejected", "554 content rejected\\r\\n", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			require.NoError(t, err)
			defer listener.Close()
			done := make(chan error, 1)
			go func() {
				conn, err := listener.Accept()
				if err != nil {
					done <- err
					return
				}
				defer conn.Close()
				r := bufio.NewReader(conn)
				w := bufio.NewWriter(conn)
				reply := func(s string) error {
					if _, err := w.WriteString(s); err != nil {
						return err
					}
					return w.Flush()
				}
				if err := reply("220 fake SMTP ready\\r\\n"); err != nil {
					done <- err
					return
				}
				for {
					line, err := r.ReadString('\\n')
					if err != nil {
						done <- err
						return
					}
					switch {
					case strings.HasPrefix(line, "EHLO "), strings.HasPrefix(line, "HELO "):
						err = reply("250 localhost\\r\\n")
					case strings.HasPrefix(line, "MAIL FROM:"), strings.HasPrefix(line, "RCPT TO:"):
						err = reply("250 accepted\\r\\n")
					case line == "DATA\\r\\n":
						if err = reply("354 send data\\r\\n"); err == nil {
							for {
								data, e := r.ReadString('\\n')
								if e != nil {
									err = e
									break
								}
								if data == ".\\r\\n" {
									break
								}
							}
							if err == nil {
								err = reply(tc.response)
							}
						}
						done <- err
						return
					default:
						err = reply("500 unsupported\\r\\n")
					}
					if err != nil {
						done <- err
						return
					}
				}
			}()
			conn, err := net.Dial("tcp", listener.Addr().String())
			require.NoError(t, err)
			c, err := smtp.NewClient(conn, "localhost")
			require.NoError(t, err)
			defer c.Close()
			require.NoError(t, c.Mail("sender@example.test"))
			require.NoError(t, c.Rcpt("recipient@example.test"))
			err = writeSMTPData(c, "From: sender@example.test\\r\\nTo: recipient@example.test\\r\\nSubject: test\\r\\n\\r\\nHello\\r\\n")
			if tc.rejected {
				require.ErrorContains(t, err, "554")
			} else {
				require.NoError(t, err)
			}
			require.NoError(t, <-done)
		})
	}
}
