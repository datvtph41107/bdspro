package notification

import (
	"context"

	"github.com/datvtph41107/bdspro/models/db"
)

// Store provides notification status persistence.
// It owns no realtime side effects.
type Store struct{}

func (Store) MarkRead(
	ctx context.Context,
	accountID, notificationID int64,
) (bool, error) {
	q, err := db.GetQueries(ctx)
	if err != nil {
		return false, err
	}

	n, err := q.MarkNotificationRead(
		ctx,
		db.MarkNotificationReadParams{
			AccountID:      accountID,
			NotificationID: notificationID,
		},
	)
	if err != nil {
		return false, err
	}

	return n > 0, nil
}

func (Store) MarkAllRead(
	ctx context.Context,
	accountID int64,
) (int64, error) {
	q, err := db.GetQueries(ctx)
	if err != nil {
		return 0, err
	}

	return q.MarkAllNotificationsRead(ctx, accountID)
}

func (Store) CountUnread(
	ctx context.Context,
	accountID int64,
) (int64, error) {
	q, err := db.GetQueries(ctx)
	if err != nil {
		return 0, err
	}

	return q.CountUnreadNotifications(ctx, accountID)
}
