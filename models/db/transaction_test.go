package db

import (
	"context"
	"errors"
	"testing"
)

type txResultRunner func(func(*Queries) error) error

func (run txResultRunner) ExecTx(_ context.Context, fn func(*Queries) error) error {
	return run(fn)
}

func TestWithTx2CommitsBeforeReturningValue(t *testing.T) {
	committed := false
	runner := txResultRunner(func(fn func(*Queries) error) error {
		if err := fn(&Queries{}); err != nil {
			return err
		}
		committed = true
		return nil
	})
	got, err := WithTx2(WithStore(context.Background(), runner), func(txCtx context.Context) (int64, error) {
		if committed {
			t.Fatal("result callback ran after commit")
		}
		return 37, nil
	})
	if err != nil || got != 37 || !committed {
		t.Fatalf("result=%d committed=%v error=%v", got, committed, err)
	}
}

func TestWithTx2DiscardsUncommittedResult(t *testing.T) {
	commitErr := errors.New("commit rejected")
	runner := txResultRunner(func(fn func(*Queries) error) error {
		if err := fn(&Queries{}); err != nil {
			return err
		}
		return commitErr
	})
	got, err := WithTx2(WithStore(context.Background(), runner), func(txCtx context.Context) (int64, error) {
		return 99, nil
	})
	if got != 0 || !errors.Is(err, commitErr) {
		t.Fatalf("uncommitted result was returned: %d error=%v", got, err)
	}
}

func TestWithTx2CallbackFailureDoesNotPublishValue(t *testing.T) {
	callbackErr := errors.New("transaction must roll back")
	runner := txResultRunner(func(fn func(*Queries) error) error { return fn(&Queries{}) })
	got, err := WithTx2(WithStore(context.Background(), runner), func(txCtx context.Context) (string, error) {
		return "uncommitted secret", callbackErr
	})
	if got != "" || !errors.Is(err, callbackErr) {
		t.Fatalf("failed callback published value=%q error=%v", got, err)
	}
}

func TestWithTx2ReusesActiveExecutor(t *testing.T) {
	owner := &contextTestStore{}
	stray := &contextTestStore{}
	ctx := WithStore(context.Background(), owner)

	err := WithTx(ctx, func(txCtx context.Context) error {
		expected, err := TxQueries(txCtx)
		if err != nil {
			return err
		}
		result, err := WithTx2(txCtx, func(txCtx context.Context) (int, error) {
			q, getQueriesErr := GetQueries(txCtx)
			if getQueriesErr != nil {
				var zero int
				return zero, getQueriesErr
			}
			if q != expected {
				t.Fatal("WithTx2 escaped the transaction-bound executor")
			}
			return 17, nil
		})
		if err != nil || result != 17 {
			t.Fatalf("nested result=%d err=%v", result, err)
		}
		return nil
	})
	if err != nil || !owner.committed || owner.transactions != 1 || stray.transactions != 0 {
		t.Fatalf("err=%v owner=%+v stray=%+v", err, owner, stray)
	}
}

func TestWithTx2IgnoredNestedErrorPoisonsOwner(t *testing.T) {
	owner := &contextTestStore{}
	stray := &contextTestStore{}
	sentinel := errors.New("reject nested write")
	ctx := WithStore(context.Background(), owner)

	err := WithTx(ctx, func(txCtx context.Context) error {
		result, nestedErr := WithTx2(txCtx, func(txCtx context.Context) (string, error) {
			return "must not publish", sentinel
		})
		if result != "" || !errors.Is(nestedErr, sentinel) {
			t.Fatalf("nested result=%q err=%v", result, nestedErr)
		}
		// Deliberately ignore the nested error: the parent must still fail.
		return nil
	})
	if !errors.Is(err, sentinel) || owner.committed || owner.transactions != 1 || stray.transactions != 0 {
		t.Fatalf("err=%v owner=%+v stray=%+v", err, owner, stray)
	}
}

func TestWithTx2RejectsOrphanedTransaction(t *testing.T) {
	stray := &contextTestStore{}
	ctx := context.WithValue(context.Background(), queriesContextKey{}, &Queries{})
	result, err := WithTx2(WithStore(ctx, stray), func(txCtx context.Context) (int, error) {
		t.Fatal("orphaned executor must not execute")
		return 1, nil
	})
	if result != 0 || !errors.Is(err, ErrNestedTransaction) || stray.transactions != 0 {
		t.Fatalf("result=%d err=%v stray=%+v", result, err, stray)
	}
}

func TestWithTx2CancellationPreventsCommit(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	committed := false
	runner := txResultRunner(func(fn func(*Queries) error) error {
		if err := fn(&Queries{}); err != nil {
			return err
		}
		committed = true
		return nil
	})
	result, err := WithTx2(WithStore(ctx, runner), func(txCtx context.Context) (int, error) {
		cancel()
		return 41, nil
	})
	if result != 0 || !errors.Is(err, context.Canceled) || committed {
		t.Fatalf("result=%d err=%v committed=%v", result, err, committed)
	}
}
