package phonedelivery

import (
	"context"
	"errors"
	"regexp"
	"time"

	token "github.com/datvtph41107/bdspro/modules/auth"
	"github.com/datvtph41107/bdspro/modules/log"
)

var safeMessageID = regexp.MustCompile(`^[A-Za-z0-9_-]{1,128}$`)

func (s *Service) SendCode(ctx context.Context, channel Channel, phone, code string) error {
	resolved, err := s.ResolveChannel(string(channel))
	if err != nil {
		return err
	}
	canonical, err := token.NormalizePhone(phone, "")
	if err != nil || canonical != phone || !token.ValidPhoneCode(code) {
		return &Failure{Reason: "invalid_delivery_message"}
	}
	select {
	case s.slots <- struct{}{}:
		defer func() { <-s.slots }()
	default:
		return &Failure{Reason: "delivery_capacity"}
	}
	started := time.Now()
	result, err := s.senders[resolved].Send(ctx, message{Phone: phone, Code: code})
	logger := log.For(ctx, "delivery")
	level := log.INFO
	if err != nil {
		level = log.ERROR
	}
	event := logger.WithLevel(level)
	if err != nil {
		var failure *Failure
		if errors.As(err, &failure) {
			event.Str("reason", failure.Reason).Int("provider_status", failure.HTTPStatus).
				Str("provider_code", failure.ProviderCode)
		} else {
			event.Str("reason", "delivery_failed")
		}
	}
	if result.Provider != "" {
		event.Str("provider", result.Provider)
	}
	if safeMessageID.MatchString(result.MessageID) {
		event.Str("provider_message_id", result.MessageID)
	}
	event.Str("channel", string(resolved)).Str("mode", s.mode).
		Bool("accepted", err == nil).Dur("duration_ms", time.Since(started)).Msg("phone.delivery")
	return err
}
