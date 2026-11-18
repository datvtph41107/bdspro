package notification

import (
	"context"

	"github.com/datvtph41107/bdspro/models/db"
)

// MarkRead changes an unread notification to read.
// It returns whether a row was actually changed.
func MarkRead(
	ctx context.Context,
	accountID, notificationID int64,
) (bool, error) {
	queries, err := db.GetQueries(ctx)
	if err != nil {
		return false, err
	}

	affected, err := queries.MarkNotificationRead(
		ctx,
		db.MarkNotificationReadParams{
			AccountID:      accountID,
			NotificationID: notificationID,
		},
	)
	if err != nil {
		return false, err
	}

	return affected > 0, nil
}

// MarkAllRead marks all unread notifications of an account.
// A zero count is a valid, idempotent result.
func MarkAllRead(
	ctx context.Context,
	accountID int64,
) (int64, error) {
	queries, err := db.GetQueries(ctx)
	if err != nil {
		return 0, err
	}

	return queries.MarkAllNotificationsRead(
		ctx,
		accountID,
	)
}

// CountUnread reads the authoritative unread count.
func CountUnread(
	ctx context.Context,
	accountID int64,
) (int64, error) {
	queries, err := db.GetQueries(ctx)
	if err != nil {
		return 0, err
	}

	return queries.CountUnreadNotifications(
		ctx,
		accountID,
	)
}
