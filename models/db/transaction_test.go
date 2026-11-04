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

func TestWithTxResultCommitsBeforeReturningValue(t *testing.T) {
	committed := false
	runner := txResultRunner(func(fn func(*Queries) error) error {
		if err := fn(nil); err != nil {
			return err
		}
		committed = true
		return nil
	})
	got, err := WithTxResult(context.Background(), runner, func(_ *Queries) (int64, error) {
		if committed {
			t.Fatal("result callback ran after commit")
		}
		return 37, nil
	})
	if err != nil || got != 37 || !committed {
		t.Fatalf("result=%d committed=%v error=%v", got, committed, err)
	}
}

func TestWithTxResultDiscardsUncommittedResult(t *testing.T) {
	commitErr := errors.New("commit rejected")
	runner := txResultRunner(func(fn func(*Queries) error) error {
		if err := fn(nil); err != nil {
			return err
		}
		return commitErr
	})
	got, err := WithTxResult(context.Background(), runner, func(_ *Queries) (int64, error) {
		return 99, nil
	})
	if got != 0 || !errors.Is(err, commitErr) {
		t.Fatalf("uncommitted result was returned: %d error=%v", got, err)
	}
}

func TestWithTxResultCallbackFailureDoesNotPublishValue(t *testing.T) {
	callbackErr := errors.New("transaction must roll back")
	runner := txResultRunner(func(fn func(*Queries) error) error { return fn(nil) })
	got, err := WithTxResult(context.Background(), runner, func(_ *Queries) (string, error) {
		return "uncommitted secret", callbackErr
	})
	if got != "" || !errors.Is(err, callbackErr) {
		t.Fatalf("failed callback published value=%q error=%v", got, err)
	}
}
