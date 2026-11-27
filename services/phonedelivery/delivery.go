package phonedelivery

import (
	"context"
	"errors"
	"regexp"
	"time"

	token "github.com/datvtph41107/bdspro/modules/auth"
	"github.com/datvtph41107/bdspro/modules/log"
	transport "github.com/datvtph41107/bdspro/modules/phonedelivery"
)

type Message = transport.Message
type Receipt = transport.Receipt
type Failure = transport.Failure

// Sender returns when the provider accepts the message, not when it is delivered.
// Implementations must not log message content or credentials, or retry implicitly.
type Sender interface {
	Send(context.Context, Message) (Receipt, error)
}

var ErrUnavailable = transport.ErrUnavailable

var safeMessageID = regexp.MustCompile(`^[A-Za-z0-9_-]{1,128}$`)

func (s *Service) SendCode(ctx context.Context, channel Channel, phone, code string) (Receipt, error) {
	resolved, err := s.ResolveChannel(string(channel))
	if err != nil {
		return Receipt{}, err
	}
	canonical, err := token.NormalizePhone(phone, "")
	if err != nil || canonical != phone || !token.ValidPhoneCode(code) {
		return Receipt{}, &Failure{Reason: "invalid_delivery_message"}
	}
	select {
	case s.slots <- struct{}{}:
		defer func() { <-s.slots }()
	default:
		return Receipt{}, &Failure{Reason: "delivery_capacity"}
	}
	sender := s.senders[resolved]
	started := time.Now()
	receipt, err := sender.Send(ctx, Message{Phone: phone, Code: code})
	logger := log.For(ctx, "delivery")
	event := logger.Info()
	if err != nil {
		event = logger.Error()
		var failure *Failure
		if errors.As(err, &failure) {
			event.Str("reason", failure.Reason).Int("provider_status", failure.HTTPStatus).
				Str("provider_code", failure.ProviderCode)
		} else {
			event.Str("reason", "delivery_failed")
		}
	}
	if receipt.Provider != "" {
		event.Str("provider", receipt.Provider)
	}
	if safeMessageID.MatchString(receipt.MessageID) {
		event.Str("provider_message_id", receipt.MessageID)
	}
	event.Str("channel", string(resolved)).Str("mode", s.mode).
		Bool("accepted", err == nil).Dur("duration_ms", time.Since(started)).Msg("phone.delivery")
	return receipt, err
}
