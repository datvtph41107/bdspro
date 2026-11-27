// Package phonedelivery applies channel policy and coordinates OTP delivery.
// OTP issuance and verification remain in services/auth.
package phonedelivery

import (
	"net/http"
	"slices"
	"time"

	"github.com/datvtph41107/bdspro/modules/setting"
)

const (
	maxConcurrentSends = 4
	providerTimeout    = 5 * time.Second
)

// Service owns an immutable channel configuration and its own concurrency budget.
type Service struct {
	mode           string
	defaultChannel string
	channels       []string
	senders        map[Channel]sender
	slots          chan struct{}
}

// New constructs adapters once at startup from validated application settings.
func New(cfg setting.PhoneDeliverySettings, sms setting.SMSSettings, runMode string) (*Service, error) {
	s := &Service{
		mode:           cfg.Mode,
		defaultChannel: cfg.DefaultChannel,
		channels:       append([]string(nil), cfg.Channels...),
		senders:        make(map[Channel]sender),
		slots:          make(chan struct{}, maxConcurrentSends),
	}
	if cfg.Mode == "" || cfg.Mode == "disabled" {
		s.channels = nil
		return s, nil
	}
	if cfg.Mode != "live" && cfg.Mode != "mailpit" {
		return nil, &Failure{Reason: "delivery_not_configured"}
	}
	if cfg.Mode == "mailpit" && runMode != "development" {
		return nil, &Failure{Reason: "development_transport_forbidden"}
	}
	client := &http.Client{
		Timeout:       providerTimeout,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	for _, raw := range s.channels {
		channel := Channel(raw)
		switch channel {
		case SMS, Zalo, Telegram:
		default:
			return nil, ErrInvalidChannel
		}
		if cfg.Mode == "mailpit" {
			s.senders[channel] = mailpitSender{channel: channel}
			continue
		}
		switch channel {
		case SMS:
			s.senders[channel] = twilioSender{
				client:     client,
				accountSID: sms.AccountSID,
				authToken:  sms.AuthToken,
				senderID:   sms.SenderID,
			}
		case Zalo:
			s.senders[channel] = zaloSender{
				client:     client,
				token:      cfg.ZaloAccessToken,
				templateID: cfg.ZaloTemplateID,
				otpParam:   cfg.ZaloOTPParam,
			}
		case Telegram:
			s.senders[channel] = telegramSender{client: client, token: cfg.TelegramAccessToken}
		}
	}
	if !slices.Contains(s.channels, s.defaultChannel) {
		return nil, ErrInvalidChannel
	}
	return s, nil
}
