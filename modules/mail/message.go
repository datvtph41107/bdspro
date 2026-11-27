package mail

import (
	"fmt"
	"net/mail"
	"strings"
)

func plainMessage(from, to, subject, body string) (string, error) {
	if strings.ContainsAny(subject, "\r\n") {
		return "", ErrUnavailable
	}
	for _, address := range []string{from, to} {
		parsed, err := mail.ParseAddress(address)
		if err != nil || parsed.Address != address || strings.ContainsAny(address, "\r\n") {
			return "", ErrUnavailable
		}
	}
	return fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s", from, to, subject, body), nil
}
