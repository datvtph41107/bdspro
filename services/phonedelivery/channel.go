package phonedelivery

import "github.com/datvtph41107/bdspro/modules/util"

type Channel string

const (
	SMS      Channel = "sms"
	Zalo     Channel = "zalo"
	Telegram Channel = "telegram"
)

var ErrInvalidChannel = util.WithPublic(util.ErrInvalidArgument,
	"phone delivery channel not enabled", "delivery_channel_unavailable",
	"choose an available verification channel")

func (s *Service) ResolveChannel(raw string) (Channel, error) {
	if s == nil || s.mode == "" || s.mode == "disabled" {
		return "", &Failure{Reason: "delivery_not_configured"}
	}
	if raw == "" {
		raw = s.defaultChannel
	}
	if s.senders[Channel(raw)] == nil {
		return "", ErrInvalidChannel
	}
	return Channel(raw), nil
}

// Options contains public capabilities from the same service that sends messages.
type Options struct {
	Channels       []string `json:"channels"`
	DefaultChannel string   `json:"default_channel"`
	Development    bool     `json:"development"`
}

func (s *Service) Options() Options {
	if s == nil || s.mode == "" || s.mode == "disabled" {
		return Options{Channels: []string{}}
	}
	return Options{Channels: append([]string{}, s.channels...), DefaultChannel: s.defaultChannel, Development: s.mode == "mailpit"}
}
