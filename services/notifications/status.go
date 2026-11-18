package notifications

import (
	"context"
	"errors"
	"fmt"

	"github.com/datvtph41107/bdspro/models/db"
)

var (
	ErrInvalidAccount = errors.New(
		"notifications: invalid account",
	)
	ErrInvalidNotification = errors.New(
		"notifications: invalid notification",
	)
	ErrTransactionScope = errors.New(
		"notifications: operation requires an outer commit boundary",
	)
	ErrDependency = errors.New(
		"notifications: missing dependency",
	)
)

// StatusStore is defined by the service consumer,
// not by a general-purpose repository framework.
type StatusStore interface {
	MarkRead(context.Context, int64, int64) (bool, error)
	MarkAllRead(context.Context, int64) (int64, error)
}

type CountNotifier interface {
	NotificationCountChange(context.Context, int64)
}

// Service coordinates a committed status mutation
// with a best-effort count-change signal.
type Service struct {
	store    StatusStore
	notifier CountNotifier
}

func New(
	store StatusStore,
	notifier CountNotifier,
) (*Service, error) {
	if store == nil || notifier == nil {
		return nil, ErrDependency
	}
	return &Service{store: store, notifier: notifier}, nil
}

func validate(
	ctx context.Context,
	accountID int64,
) error {
	if ctx == nil {
		return errors.New("notifications: nil context")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if accountID <= 0 {
		return ErrInvalidAccount
	}
	if db.InTransaction(ctx) {
		return ErrTransactionScope
	}
	return nil
}

func (s *Service) MarkRead(
	ctx context.Context,
	accountID, notificationID int64,
) (bool, error) {
	if s == nil || s.store == nil || s.notifier == nil {
		return false, ErrDependency
	}
	if err := validate(ctx, accountID); err != nil {
		return false, err
	}
	if notificationID <= 0 {
		return false, ErrInvalidNotification
	}

	changed, err := s.store.MarkRead(
		ctx, accountID, notificationID,
	)
	if err != nil {
		return false, fmt.Errorf(
			"notifications: mark read: %w", err,
		)
	}

	if changed {
		s.notifier.NotificationCountChange(ctx, accountID)
	}
	return changed, nil
}

func (s *Service) MarkAllRead(
	ctx context.Context,
	accountID int64,
) (int64, error) {
	if s == nil || s.store == nil || s.notifier == nil {
		return 0, ErrDependency
	}
	if err := validate(ctx, accountID); err != nil {
		return 0, err
	}

	n, err := s.store.MarkAllRead(ctx, accountID)
	if err != nil {
		return 0, fmt.Errorf(
			"notifications: mark all read: %w", err,
		)
	}

	if n > 0 {
		s.notifier.NotificationCountChange(ctx, accountID)
	}
	return n, nil
}
