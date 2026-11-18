package notifications

import (
	"context"
	"errors"
	"testing"
)

type fakeStore struct {
	changed bool
	count   int64
	err     error
}

func (f *fakeStore) MarkRead(
	context.Context, int64, int64,
) (bool, error) {
	return f.changed, f.err
}

func (f *fakeStore) MarkAllRead(
	context.Context, int64,
) (int64, error) {
	return f.count, f.err
}

type spyNotifier struct {
	calls []int64
}

func (s *spyNotifier) NotificationCountChange(
	_ context.Context,
	accountID int64,
) {
	s.calls = append(s.calls, accountID)
}

func TestMarkRead(t *testing.T) {
	tests := []struct {
		name      string
		changed   bool
		storeErr  error
		wantCalls int
	}{
		{"changed", true, nil, 1},
		{"unchanged", false, nil, 0},
		{"db failure", false, errors.New("db down"), 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			notifier := &spyNotifier{}
			service, err := New(
				&fakeStore{
					changed: tt.changed,
					err:     tt.storeErr,
				},
				notifier,
			)
			if err != nil {
				t.Fatal(err)
			}

			changed, err := service.MarkRead(
				context.Background(), 42, 99,
			)

			if (err != nil) != (tt.storeErr != nil) {
				t.Fatalf("error = %v", err)
			}
			if tt.storeErr == nil && changed != tt.changed {
				t.Fatalf("changed = %v", changed)
			}
			if len(notifier.calls) != tt.wantCalls {
				t.Fatalf("notifications = %v", notifier.calls)
			}
		})
	}
}

func TestMarkAllReadNotifiesOnce(t *testing.T) {
	notifier := &spyNotifier{}
	service, err := New(
		&fakeStore{count: 100},
		notifier,
	)
	if err != nil {
		t.Fatal(err)
	}

	n, err := service.MarkAllRead(context.Background(), 42)
	if err != nil {
		t.Fatal(err)
	}
	if n != 100 || len(notifier.calls) != 1 {
		t.Fatalf("rows=%d notifications=%v", n, notifier.calls)
	}
}

func TestInvalidAccountDoesNotNotify(t *testing.T) {
	notifier := &spyNotifier{}
	service, err := New(&fakeStore{changed: true}, notifier)
	if err != nil {
		t.Fatal(err)
	}

	_, err = service.MarkRead(context.Background(), 0, 99)
	if !errors.Is(err, ErrInvalidAccount) {
		t.Fatalf("error = %v", err)
	}
	if len(notifier.calls) != 0 {
		t.Fatalf("unexpected notify: %v", notifier.calls)
	}
}
