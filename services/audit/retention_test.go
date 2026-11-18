package audit

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/datvtph41107/bdspro/services/cron"
)

func TestSweepStopsWhenBatchIsPartial(t *testing.T) {
	now := time.Date(2026, 11, 18, 0, 0, 0, 0, time.UTC)
	var calls int
	n, err := Sweep(context.Background(), 90, func(_ context.Context, cutoff time.Time, batch int32) (int64, error) {
		calls++
		if batch != 1000 {
			t.Fatalf("batch = %d", batch)
		}
		if !cutoff.Equal(now.AddDate(0, 0, -90)) {
			t.Fatalf("cutoff = %s", cutoff)
		}
		if calls == 1 {
			return 1000, nil
		}
		return 17, nil
	}, now)
	if err != nil || n != 1017 || calls != 2 {
		t.Fatalf("n=%d calls=%d err=%v", n, calls, err)
	}
}

func TestSweepHasHardUpperBound(t *testing.T) {
	calls := 0
	n, err := Sweep(context.Background(), 90, func(context.Context, time.Time, int32) (int64, error) {
		calls++
		return 1000, nil
	}, time.Now())
	if err != nil || n != 20000 || calls != 20 {
		t.Fatalf("n=%d calls=%d err=%v", n, calls, err)
	}
}

func TestSweepReturnsPartialCountOnFailure(t *testing.T) {
	sentinel := errors.New("db offline")
	calls := 0
	n, err := Sweep(context.Background(), 90, func(context.Context, time.Time, int32) (int64, error) {
		calls++
		if calls == 1 {
			return 1000, nil
		}
		return 0, sentinel
	}, time.Now())
	if n != 1000 || !errors.Is(err, sentinel) {
		t.Fatalf("n=%d err=%v", n, err)
	}
}

func TestSweepRejectsInvalidConfiguration(t *testing.T) {
	_, err := Sweep(context.Background(), 0, func(context.Context, time.Time, int32) (int64, error) { return 0, nil }, time.Now())
	if err == nil {
		t.Fatal("invalid retention accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = Sweep(ctx, 90, func(context.Context, time.Time, int32) (int64, error) { t.Fatal("must not query"); return 0, nil }, time.Now())
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err=%v", err)
	}
}

func TestRegisterRetentionIsExplicitAndValidatesConfig(t *testing.T) {
	s := cron.New(context.Background())
	if s.HasJobs() {
		t.Fatal("new scheduler started with registered work")
	}
	if err := RegisterRetention(s, 90); err != nil {
		t.Fatalf("register retention: %v", err)
	}
	if !s.HasJobs() {
		t.Fatal("retention task not registered")
	}
	if err := RegisterRetention(s, 90); err == nil {
		t.Fatal("duplicate retention task accepted")
	}
	if err := RegisterRetention(nil, 90); err == nil {
		t.Fatal("nil scheduler accepted")
	}
	if err := RegisterRetention(cron.New(context.Background()), 0); err == nil {
		t.Fatal("invalid retention period accepted")
	}
}
