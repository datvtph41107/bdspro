package phonedelivery

import (
	"context"
	"errors"
)

type message struct {
	Phone      string
	Code       string
	TTLMinutes int
}

// receipt acknowledges provider acceptance, not delivery to the recipient.
type receipt struct {
	Provider  string
	MessageID string
}

var ErrUnavailable = errors.New("phone code delivery unavailable")

// Failure contains only safe operational fields; never upstream response text.
type Failure struct {
	Reason       string
	HTTPStatus   int
	ProviderCode string
}

func (e *Failure) Error() string { return e.Reason }
func (e *Failure) Unwrap() error { return ErrUnavailable }

// sender returns when the provider accepts the message, not when it is delivered.
// Implementations must not log message content or credentials, or retry implicitly.
type sender interface {
	Send(context.Context, message) (receipt, error)
}
