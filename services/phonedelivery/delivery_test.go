package phonedelivery

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/datvtph41107/bdspro/modules/setting"
)

type senderFunc func(context.Context, Message) (Receipt, error)

func (f senderFunc) Send(ctx context.Context, m Message) (Receipt, error) { return f(ctx, m) }

func testService(t *testing.T) *Service {
	t.Helper()
	s, err := New(setting.PhoneDeliverySettings{Mode: "mailpit", DefaultChannel: "sms", Channels: []string{"sms", "zalo", "telegram"}}, setting.SMSSettings{}, "development")
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestChannelPolicySnapshotAndProductionGuard(t *testing.T) {
	cfg := setting.PhoneDeliverySettings{Mode: "mailpit", DefaultChannel: "sms", Channels: []string{"sms", "zalo"}}
	s, err := New(cfg, setting.SMSSettings{}, "development")
	if err != nil {
		t.Fatal(err)
	}
	cfg.Channels[0] = "telegram"
	options := s.Options()
	options.Channels[0] = "telegram"
	if got, err := s.ResolveChannel(""); got != SMS || err != nil {
		t.Fatalf("default changed: %s %v", got, err)
	}
	if _, err := s.ResolveChannel("telegram"); !errors.Is(err, ErrInvalidChannel) {
		t.Fatalf("disabled channel: %v", err)
	}
	if _, err := s.ResolveChannel("whatsapp"); !errors.Is(err, ErrInvalidChannel) {
		t.Fatalf("unknown channel: %v", err)
	}
	if s.Options().Channels[0] != "sms" {
		t.Fatal("capabilities expose mutable configuration")
	}
	if _, err := New(cfg, setting.SMSSettings{}, "production"); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("production accepted Mailpit: %v", err)
	}
	disabled, err := New(setting.PhoneDeliverySettings{Mode: "disabled"}, setting.SMSSettings{}, "production")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := disabled.ResolveChannel(""); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("disabled: %v", err)
	}
	if len(disabled.Options().Channels) != 0 {
		t.Fatal("disabled service advertises channels")
	}
	if _, err := FromContext(context.Background()).ResolveChannel(""); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("missing dependency: %v", err)
	}
	if FromContext(WithService(context.Background(), s)) != s {
		t.Fatal("request lost configured instance")
	}
}

func TestSendValidatesBeforeCallingProviderAndDoesNotFallback(t *testing.T) {
	s := testService(t)
	calls := 0
	failure := &Failure{Reason: "provider_transport_error"}
	s.senders[SMS] = senderFunc(func(_ context.Context, m Message) (Receipt, error) {
		calls++
		if m.Phone != "+84912345678" || m.Code != "001234" {
			t.Fatal("OTP or destination changed")
		}
		return Receipt{}, failure
	})
	s.senders[Zalo] = senderFunc(func(context.Context, Message) (Receipt, error) {
		t.Fatal("ambiguous provider failure must not trigger fallback")
		return Receipt{}, nil
	})
	for _, tc := range []struct{ phone, code string }{
		{"0912345678", "001234"}, {"+84912345678", "12345"}, {"+84912345678", "abcd12"},
	} {
		if _, err := s.SendCode(context.Background(), SMS, tc.phone, tc.code); !errors.Is(err, ErrUnavailable) {
			t.Fatalf("invalid message: %v", err)
		}
	}
	if calls != 0 {
		t.Fatal("invalid messages reached provider")
	}
	if _, err := s.SendCode(context.Background(), SMS, "+84912345678", "001234"); !errors.Is(err, failure) {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("automatic retry: %d calls", calls)
	}
	if len(s.slots) != 0 {
		t.Fatal("failed send leaked concurrency slot")
	}
}

func TestCapacityIsSharedAcrossChannelsAndReleasedAfterCancellation(t *testing.T) {
	s := testService(t)
	entered := make(chan struct{}, 4)
	fake := senderFunc(func(ctx context.Context, _ Message) (Receipt, error) {
		entered <- struct{}{}
		<-ctx.Done()
		return Receipt{}, &Failure{Reason: "provider_transport_error"}
	})
	for channel := range s.senders {
		s.senders[channel] = fake
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 4)
	for _, channel := range []Channel{SMS, Zalo, Telegram, SMS} {
		go func(ch Channel) { _, err := s.SendCode(ctx, ch, "+84912345678", "001234"); done <- err }(channel)
	}
	for range 4 {
		select {
		case <-entered:
		case <-time.After(2 * time.Second):
			t.Fatal("sender did not start")
		}
	}
	_, err := s.SendCode(context.Background(), Zalo, "+84912345678", "001234")
	var failure *Failure
	if !errors.As(err, &failure) || failure.Reason != "delivery_capacity" {
		t.Fatalf("capacity bypass: %v", err)
	}
	cancel()
	for range 4 {
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatal("cancellation did not release send")
		}
	}
	if len(s.slots) != 0 {
		t.Fatal("cancelled requests leaked slots")
	}
}
