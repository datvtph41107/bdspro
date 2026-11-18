package notifications

import (
	"context"
	"errors"
	"testing"

	"github.com/datvtph41107/bdspro/services/notify"
)

func TestNewRequiresDispatcher(t *testing.T) {
	_, err := New(nil)
	if !errors.Is(err, ErrNoDispatcher) {
		t.Fatalf("expected ErrNoDispatcher, got %v", err)
	}
}

func TestMarkReadRejectsInvalidIDs(t *testing.T) {
	dispatcher, err := notify.NewDispatcher()
	if err != nil {
		t.Fatal(err)
	}

	service, err := New(dispatcher)
	if err != nil {
		t.Fatal(err)
	}

	_, err = service.MarkRead(context.Background(), 0, 10)
	if !errors.Is(err, ErrInvalidAccountID) {
		t.Fatalf("invalid account: %v", err)
	}

	_, err = service.MarkRead(context.Background(), 42, 0)
	if !errors.Is(err, ErrInvalidNotificationID) {
		t.Fatalf("invalid notification: %v", err)
	}
}

func TestMarkAllReadRejectsCancelledContext(t *testing.T) {
	dispatcher, err := notify.NewDispatcher()
	if err != nil {
		t.Fatal(err)
	}

	service, err := New(dispatcher)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = service.MarkAllRead(ctx, 42)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected canceled context, got %v", err)
	}
}
