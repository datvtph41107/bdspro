package pubsub

import "fmt"

const subscriberBuffer = 8

// Broker delivers transient realtime messages.
//
// Publish is best-effort, not a durable enqueue.
// Subscribers must tolerate missed messages.
type Broker interface {
	Subscribe(topic string) (<-chan []byte, func())
	Publish(topic string, msg []byte)
	HasTopicSubscribers(topic string) bool
}

// UserTopic is the logical channel for all live
// connections belonging to an authenticated account.
func UserTopic(accountID int64) string {
	return fmt.Sprintf("user-%d", accountID)
}
