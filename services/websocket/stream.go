package websocket

import (
	"context"
	"errors"

	"github.com/datvtph41107/bdspro/services/pubsub"
)

var (
	ErrStreamDependency = errors.New(
		"websocket: missing stream dependency",
	)
	ErrStreamIdentity = errors.New(
		"websocket: invalid stream identity",
	)
	ErrSubscriptionClosed = errors.New(
		"websocket: subscription closed",
	)
)

// Snapshot builds an initial broker-format event
// using committed, authoritative state.
type Snapshot func(
	ctx context.Context,
	accountID int64,
) ([]byte, error)

// Send writes client JSON to a connection.
// Its implementation belongs to the HTTP/WebSocket adapter.
type Send func(
	ctx context.Context,
	payload []byte,
) error

// Stream coordinates an authenticated account subscription.
// It does not authenticate identities or own network connections.
type Stream struct {
	broker   pubsub.Broker
	snapshot Snapshot
}

func NewStream(
	broker pubsub.Broker,
	snapshot Snapshot,
) (*Stream, error) {
	if broker == nil || snapshot == nil {
		return nil, ErrStreamDependency
	}
	return &Stream{
		broker:   broker,
		snapshot: snapshot,
	}, nil
}

// Serve subscribes before reading the initial snapshot.
//
// accountID and sessionID MUST come from a trusted
// authenticated actor, never URL parameters.
func (s *Stream) Serve(
	ctx context.Context,
	accountID int64,
	sessionID string,
	send Send,
) error {
	if s == nil || s.broker == nil ||
		s.snapshot == nil || send == nil {
		return ErrStreamDependency
	}
	if ctx == nil {
		return errors.New("websocket: nil context")
	}
	if accountID <= 0 || sessionID == "" {
		return ErrStreamIdentity
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	// Subscribe before snapshot to avoid a gap
	// where no subscriber exists.
	ch, cancel := s.broker.Subscribe(
		pubsub.UserTopic(accountID),
	)
	defer cancel()

	// Never read the subscription before the event loop: doing so
	// discards an event that arrives immediately after Subscribe.
	// A closed subscription is detected by the loop below.

	initial, err := s.snapshot(ctx, accountID)
	if err != nil {
		return err
	}

	payload, accepted := EventForSession(
		initial, sessionID,
	)
	if !accepted {
		return ErrEventEnvelope
	}
	if err := send(ctx, payload); err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case event, ok := <-ch:
			if !ok {
				return ErrSubscriptionClosed
			}

			payload, accepted := EventForSession(
				event, sessionID,
			)
			if !accepted {
				continue
			}

			if err := send(ctx, payload); err != nil {
				return err
			}
		}
	}
}
