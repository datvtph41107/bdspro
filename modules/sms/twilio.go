package sms

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/datvtph41107/bdspro/modules/setting"
)

var ErrUnavailable = errors.New("SMS delivery unavailable")

var slots = make(chan struct{}, 4)
var client = &http.Client{
	Timeout: 5 * time.Second,
	CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	},
}

func SendCode(ctx context.Context, phone, code string) error {
	cfg := setting.SMS
	if !cfg.Enabled() {
		return ErrUnavailable
	}
	select {
	case slots <- struct{}{}:
		defer func() { <-slots }()
	default:
		return ErrUnavailable
	}
	return sendCode(ctx, client, cfg, phone, code)
}

func sendCode(ctx context.Context, httpClient *http.Client, cfg setting.SMSSettings, phone, code string) error {
	if len(code) != 6 || strings.IndexFunc(code, func(r rune) bool { return r < '0' || r > '9' }) >= 0 {
		return ErrUnavailable
	}
	form := url.Values{
		"To":   {phone},
		"From": {cfg.SenderID},
		"Body": {"[" + cfg.SenderID + "] Your verification code is " + code + ". It expires in 5 minutes."},
	}
	endpoint := "https://api.twilio.com/2010-04-01/Accounts/" + cfg.AccountSID + "/Messages.json"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return ErrUnavailable
	}
	req.SetBasicAuth(cfg.AccountSID, cfg.AuthToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := httpClient.Do(req)
	if err != nil {
		return ErrUnavailable
	}
	defer resp.Body.Close()

	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
	if resp.StatusCode != http.StatusCreated {
		return ErrUnavailable
	}
	return nil
}
