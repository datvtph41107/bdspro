package phonedelivery

import (
	"context"
	"net/http"
	"slices"
	"time"

	delivery "github.com/datvtph41107/bdspro/modules/mail"
	transport "github.com/datvtph41107/bdspro/modules/phonedelivery"
	"github.com/datvtph41107/bdspro/modules/setting"
)

// New constructs adapters once at startup from validated application settings.
func New(cfg setting.PhoneDeliverySettings, sms setting.SMSSettings, runMode string) (*Service, error) {
	s := &Service{mode: cfg.Mode, defaultChannel: cfg.DefaultChannel,
		channels: append([]string(nil), cfg.Channels...), senders: make(map[Channel]Sender), slots: make(chan struct{}, 4)}
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
		Timeout:       5 * time.Second,
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
			s.senders[channel] = transport.NewTwilio(client, transport.TwilioConfig{
				AccountSID: sms.AccountSID, AuthToken: sms.AuthToken, SenderID: sms.SenderID})
		case Zalo:
			s.senders[channel] = transport.NewZalo(client, cfg.ZaloAccessToken, cfg.ZaloTemplateID, cfg.ZaloOTPParam)
		case Telegram:
			s.senders[channel] = transport.NewTelegramGateway(client, cfg.TelegramAccessToken)
		}
	}
	if !slices.Contains(s.channels, s.defaultChannel) {
		return nil, ErrInvalidChannel
	}
	return s, nil
}

type mailpitSender struct{ channel Channel }

func (s mailpitSender) Send(ctx context.Context, m Message) (Receipt, error) {
	if err := delivery.SendDevelopmentPhoneCode(ctx, m.Phone, m.Code, string(s.channel)); err != nil {
		return Receipt{}, &Failure{Reason: "mailpit_unavailable"}
	}
	return Receipt{Provider: "mailpit"}, nil
}
