// Package phonedelivery implements provider HTTP protocols for phone OTP transport.
// Channel policy and application orchestration belong to services/phonedelivery.
package phonedelivery

import "errors"

type Message struct {
	Phone string
	Code  string
}

// Receipt acknowledges provider acceptance, not delivery to the recipient.
type Receipt struct {
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
