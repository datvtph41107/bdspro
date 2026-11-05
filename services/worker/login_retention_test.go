package worker

import (
	"context"
	"errors"
	"testing"
	"time"

	db "github.com/datvtph41107/bdspro/models/db"
)

type loginRetentionStoreFake struct {
	connectionRows  []int64
	rejectionRows   []int64
	connectionCalls int
	rejectionCalls  int
	cutoffs         []time.Time
	takes           []int32
	connectionErr   error
	rejectionErr    error
}

func (f *loginRetentionStoreFake) DeleteExpiredAccountConnectionHistories(_ context.Context, arg db.DeleteExpiredAccountConnectionHistoriesParams) (int64, error) {
	f.cutoffs = append(f.cutoffs, arg.Cutoff)
	f.takes = append(f.takes, arg.Take)
	f.connectionCalls++
	if f.connectionErr != nil {
		return 0, f.connectionErr
	}
	if f.connectionCalls <= len(f.connectionRows) {
		return f.connectionRows[f.connectionCalls-1], nil
	}
	return 0, nil
}

func (f *loginRetentionStoreFake) DeleteOldAuditEvents(_ context.Context, arg db.DeleteOldAuditEventsParams) (int64, error) {
	f.cutoffs = append(f.cutoffs, arg.Cutoff)
	f.takes = append(f.takes, arg.Take)
	f.rejectionCalls++
	if f.rejectionErr != nil {
		return 0, f.rejectionErr
	}
	if f.rejectionCalls <= len(f.rejectionRows) {
		return f.rejectionRows[f.rejectionCalls-1], nil
	}
	return 0, nil
}

func TestLoginRetentionCutoffAndBatches(t *testing.T) {
	store := &loginRetentionStoreFake{
		connectionRows: []int64{1000, 1000, 8},
		rejectionRows:  []int64{1000, 5},
	}
	now := time.Date(2026, 11, 5, 5, 0, 0, 0, time.UTC)
	n, err := NewLoginRetention(store, 90).Sweep(context.Background(), now)
	if err != nil || n != 3013 || store.connectionCalls != 3 || store.rejectionCalls != 2 {
		t.Fatalf("unexpected retention: deleted=%d connections=%d rejections=%d err=%v", n, store.connectionCalls, store.rejectionCalls, err)
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

func TestLoginRetentionStopsOnErrorAndCancel(t *testing.T) {
	failure := errors.New("postgres offline")
	store := &loginRetentionStoreFake{connectionErr: failure, rejectionRows: []int64{2}}
	n, err := NewLoginRetention(store, 90).Sweep(context.Background(), time.Now())
	if !errors.Is(err, failure) || n != 2 || store.connectionCalls != 1 || store.rejectionCalls != 1 {
		t.Fatalf("failure in connection cleanup must not starve rejected-proof cleanup: deleted=%d err=%v calls=%d/%d", n, err, store.connectionCalls, store.rejectionCalls)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	store.connectionCalls = 0
	_, err = NewLoginRetention(store, 90).Sweep(ctx, time.Now())
	if !errors.Is(err, context.Canceled) || store.connectionCalls != 0 {
		t.Fatalf("cancelled sweep should not write: %v calls=%d", err, store.connectionCalls)
	}
	store = &loginRetentionStoreFake{connectionRows: []int64{2}, rejectionErr: failure}
	n, err = NewLoginRetention(store, 90).Sweep(context.Background(), time.Now())
	if !errors.Is(err, failure) || n != 2 || store.rejectionCalls != 1 {
		t.Fatalf("partial deletion must be visible: deleted=%d err=%v", n, err)
	}
}

func TestLoginRetentionCapsWorkPerSweep(t *testing.T) {
	store := &loginRetentionStoreFake{
		connectionRows: make([]int64, loginRetentionMaxBatches),
		rejectionRows:  make([]int64, loginRetentionMaxBatches),
	}
	for i := 0; i < loginRetentionMaxBatches; i++ {
		store.connectionRows[i] = int64(loginRetentionBatchSize)
		store.rejectionRows[i] = int64(loginRetentionBatchSize)
	}
	n, err := NewLoginRetention(store, 90).Sweep(context.Background(), time.Now())
	if err != nil || store.connectionCalls != loginRetentionMaxBatches ||
		store.rejectionCalls != loginRetentionMaxBatches ||
		n != 2*int64(loginRetentionBatchSize)*loginRetentionMaxBatches {
		t.Fatalf("unbounded retention sweep: deleted=%d calls=%d/%d err=%v", n, store.connectionCalls, store.rejectionCalls, err)
	}
}
