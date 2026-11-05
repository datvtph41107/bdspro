package worker

import (
	"context"
	"errors"
	"testing"
	"time"

	db "github.com/datvtph41107/bdspro/models/db"
)

type historyRetentionStoreFake struct {
	rows    []int64
	calls   int
	cutoffs []time.Time
	takes   []int32
	err     error
}

func (f *historyRetentionStoreFake) DeleteExpiredAccountConnectionHistories(_ context.Context, arg db.DeleteExpiredAccountConnectionHistoriesParams) (int64, error) {
	f.cutoffs = append(f.cutoffs, arg.Cutoff)
	f.takes = append(f.takes, arg.Take)
	f.calls++
	if f.err != nil {
		return 0, f.err
	}
	if f.calls <= len(f.rows) {
		return f.rows[f.calls-1], nil
	}
	return 0, nil
}

func TestConnectionHistoryRetentionCutoffAndBatches(t *testing.T) {
	store := &historyRetentionStoreFake{rows: []int64{1000, 1000, 8}}
	now := time.Date(2026, 11, 5, 5, 0, 0, 0, time.UTC)
	n, err := NewConnectionHistoryRetention(store).Sweep(context.Background(), now)
	if err != nil || n != 2008 || store.calls != 3 {
		t.Fatalf("unexpected retention: deleted=%d calls=%d err=%v", n, store.calls, err)
	}
	for i := range store.cutoffs {
		if want := now.Add(-90 * 24 * time.Hour); !store.cutoffs[i].Equal(want) {
			t.Fatalf("wrong retention cutoff: %s want %s", store.cutoffs[i], want)
		}
		if store.takes[i] != 1000 {
			t.Fatalf("wrong batch size: %d", store.takes[i])
		}
	}
}

func TestConnectionHistoryRetentionStopsOnErrorAndCancel(t *testing.T) {
	failure := errors.New("postgres offline")
	store := &historyRetentionStoreFake{err: failure}
	_, err := NewConnectionHistoryRetention(store).Sweep(context.Background(), time.Now())
	if !errors.Is(err, failure) || store.calls != 1 {
		t.Fatalf("failed sweep should propagate database error: %v calls=%d", err, store.calls)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	store.calls = 0
	_, err = NewConnectionHistoryRetention(store).Sweep(ctx, time.Now())
	if !errors.Is(err, context.Canceled) || store.calls != 0 {
		t.Fatalf("cancelled sweep should not write: %v calls=%d", err, store.calls)
	}
}

func TestConnectionHistoryRetentionCapsWorkPerSweep(t *testing.T) {
	store := &historyRetentionStoreFake{rows: make([]int64, connectionHistoryMaxBatches)}
	for i := range store.rows {
		store.rows[i] = int64(connectionHistoryBatchSize)
	}
	n, err := NewConnectionHistoryRetention(store).Sweep(context.Background(), time.Now())
	if err != nil || store.calls != connectionHistoryMaxBatches || n != int64(connectionHistoryBatchSize*connectionHistoryMaxBatches) {
		t.Fatalf("unbounded retention sweep: deleted=%d calls=%d err=%v", n, store.calls, err)
	}
}
