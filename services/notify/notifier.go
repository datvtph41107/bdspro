package notify

import "context"

// Notifier receives internal application notifications.
//
// Implementations own their side effects.
// No persistence or queue delivery is implied.
type Notifier interface {
	NotificationCountChange(
		ctx context.Context,
		accountID int64,
	)
}

// NullNotifier provides no-op implementations.
// Specialized notifiers embed it and override
// only the methods they consume.
type NullNotifier struct{}

func (NullNotifier) NotificationCountChange(
	context.Context,
	int64,
) {
}
