package notifications

import (
    "context"
    "errors"
    "fmt"

    notification_model "github.com/datvtph41107/bdspro/models/notification"
    "github.com/datvtph41107/bdspro/models/db"
    "github.com/datvtph41107/bdspro/services/notify"
)

var (
    ErrInvalidAccountID = errors.New(
        "notifications: invalid account ID",
    )
    ErrInvalidNotificationID = errors.New(
        "notifications: invalid notification ID",
    )
    ErrOpenTransaction = errors.New(
        "notifications: status service requires a committed DB boundary",
    )
    ErrNoDispatcher = errors.New(
        "notifications: notify dispatcher is required",
    )
)

// Service coordinates persisted status mutations
// with typed realtime notifications.
type Service struct {
    dispatcher *notify.Dispatcher
}

func New(
    dispatcher *notify.Dispatcher,
) (*Service, error) {
    if dispatcher == nil {
        return nil, ErrNoDispatcher
    }

    return &Service{
        dispatcher: dispatcher,
    }, nil
}

func (s *Service) validate(
    ctx context.Context,
    accountID int64,
) error {
    if s == nil || s.dispatcher == nil {
        return ErrNoDispatcher
    }
    if ctx == nil {
        return errors.New("notifications: nil context")
    }
    if err := ctx.Err(); err != nil {
        return err
    }
    if accountID <= 0 {
        return ErrInvalidAccountID
    }
    if db.InTransaction(ctx) {
        return ErrOpenTransaction
    }
    return nil
}

// MarkRead performs one status mutation and emits
// a count-change signal only if a row changed.
func (s *Service) MarkRead(
    ctx context.Context,
    accountID, notificationID int64,
) (bool, error) {
    if err := s.validate(ctx, accountID); err != nil {
        return false, err
    }
    if notificationID <= 0 {
        return false, ErrInvalidNotificationID
    }

    changed, err := notification_model.MarkRead(
        ctx,
        accountID,
        notificationID,
    )
    if err != nil {
        return false, fmt.Errorf(
            "notifications: mark read: %w",
            err,
        )
    }

    if changed {
        s.dispatcher.NotificationCountChange(
            ctx,
            accountID,
        )
    }

    return changed, nil
}

// MarkAllRead updates the inbox and notifies once,
// regardless of the number of rows changed.
func (s *Service) MarkAllRead(
    ctx context.Context,
    accountID int64,
) (int64, error) {
    if err := s.validate(ctx, accountID); err != nil {
        return 0, err
    }

    affected, err := notification_model.MarkAllRead(
        ctx,
        accountID,
    )
    if err != nil {
        return 0, fmt.Errorf(
            "notifications: mark all read: %w",
            err,
        )
    }

    if affected > 0 {
        s.dispatcher.NotificationCountChange(
            ctx,
            accountID,
        )
    }

    return affected, nil
}
