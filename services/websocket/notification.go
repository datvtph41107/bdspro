package websocket

import (
	"context"
	"errors"
	"fmt"

	"github.com/datvtph41107/bdspro/modules/log"
	"github.com/datvtph41107/bdspro/services/notify"
	"github.com/datvtph41107/bdspro/services/pubsub"
)

const EventNotificationCount = "notification-count"

// UnreadCounter is the smallest read contract needed
// by the realtime notification adapter.
type UnreadCounter interface {
	CountUnread(ctx context.Context, accountID int64) (int64, error)
}

// Notifier converts an internal notification-count
// change into a best-effort realtime event.
type Notifier struct {
	counter UnreadCounter
	broker  pubsub.Broker
}

var _ notify.Notifier = (*Notifier)(nil)

func NewNotifier(
	counter UnreadCounter,
	broker pubsub.Broker,
) (*Notifier, error) {
	if counter == nil {
		return nil, errors.New("websocket: nil unread counter")
	}
	if broker == nil {
		return nil, errors.New("websocket: nil pubsub broker")
	}

	return &Notifier{
		counter: counter,
		broker:  broker,
	}, nil
}

// NotificationCountChange is a best-effort callback.
// A failure here must not undo a committed DB mutation.
func (n *Notifier) NotificationCountChange(
	ctx context.Context,
	accountID int64,
) {
	if n == nil || ctx == nil ||
		ctx.Err() != nil || accountID <= 0 {
		return
	}

	topic := pubsub.UserTopic(accountID)

	// For MemoryBroker this avoids an unnecessary DB read.
	// RedisBroker returns true conservatively because
	// subscribers may exist on another process.
	if !n.broker.HasTopicSubscribers(topic) {
		return
	}

	payload, err := n.NotificationCountEvent(ctx, accountID)
	if err != nil {
		log.Error(
			"websocket: notification count event: %v",
			err,
		)
		return
	}

	n.broker.Publish(topic, payload)
}

// NotificationCountEvent builds the same event used
// for realtime updates and initial connection snapshot.
//
// The leading newline represents an empty session target
// in Gitea's internal broker envelope.
func (n *Notifier) NotificationCountEvent(
	ctx context.Context,
	accountID int64,
) ([]byte, error) {
	if n == nil || n.counter == nil {
		return nil, errors.New("websocket: nil notifier counter")
	}
	if ctx == nil {
		return nil, errors.New("websocket: nil context")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if accountID <= 0 {
		return nil, errors.New("websocket: invalid account ID")
	}

	count, err := n.counter.CountUnread(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf(
			"count unread notifications: %w", err,
		)
	}
	if count < 0 {
		return nil, errors.New(
			"websocket: negative unread count",
		)
	}

	return EncodeEvent(
		EventNotificationCount,
		"", // Broadcast đến mọi session thuộc account topic.
		struct {
			Count int64 `json:"count"`
		}{
			Count: count,
		},
	)
}
