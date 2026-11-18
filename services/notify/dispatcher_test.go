package notify

import (
	"context"
	"testing"
)

type testNotifier struct {
	NullNotifier
	calls  *[]int64
	panics bool
}

func (n *testNotifier) NotificationCountChange(
	_ context.Context,
	accountID int64,
) {
	if n.panics {
		panic("simulated consumer failure")
	}

	*n.calls = append(*n.calls, accountID)
}

func TestDispatcherFanout(t *testing.T) {
	var first, second []int64

	d, err := NewDispatcher(
		&testNotifier{calls: &first},
		&testNotifier{calls: &second},
	)
	if err != nil {
		t.Fatal(err)
	}

	d.NotificationCountChange(context.Background(), 42)

	if len(first) != 1 || first[0] != 42 {
		t.Fatalf("first consumer: %v", first)
	}
	if len(second) != 1 || second[0] != 42 {
		t.Fatalf("second consumer: %v", second)
	}
}

func TestDispatcherIsolatesPanic(t *testing.T) {
	var received []int64

	d, err := NewDispatcher(
		&testNotifier{panics: true},
		&testNotifier{calls: &received},
	)
	if err != nil {
		t.Fatal(err)
	}

	d.NotificationCountChange(context.Background(), 99)

	if len(received) != 1 || received[0] != 99 {
		t.Fatalf("later consumer was skipped: %v", received)
	}
}

func TestDispatcherRejectsNil(t *testing.T) {
	if _, err := NewDispatcher(nil); err == nil {
		t.Fatal("expected registration error")
	}
}

func TestDispatcherSkipsCancelledContext(t *testing.T) {
	var received []int64

	d, err := NewDispatcher(&testNotifier{calls: &received})
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	d.NotificationCountChange(ctx, 42)

	if len(received) != 0 {
		t.Fatalf("unexpected calls: %v", received)
	}
}
