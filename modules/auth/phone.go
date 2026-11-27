package auth

import (
	"errors"
	"regexp"
	"strings"
)

var ErrPhoneFormat = errors.New("invalid phone number")
var e164 = regexp.MustCompile(`^\+[1-9][0-9]{7,14}$`)

func NormalizePhone(raw, country string) (string, error) {
	phone := strings.TrimSpace(raw)
	phone = strings.ReplaceAll(phone, " ", "")
	phone = strings.ReplaceAll(phone, "-", "")
	if !strings.HasPrefix(phone, "+") && strings.EqualFold(country, "VN") && len(phone) == 10 && strings.HasPrefix(phone, "0") {
		phone = "+84" + phone[1:]
	}
	if !e164.MatchString(phone) {
		return "", ErrPhoneFormat
	}
	return phone, nil
}
