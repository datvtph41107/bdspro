package notify

import (
	"context"
	"fmt"
	"runtime/debug"

	"github.com/datvtph41107/bdspro/modules/log"
)

// Dispatcher routes typed notifications to registered consumers.
type Dispatcher struct {
	notifiers []Notifier
}

// NewDispatcher snapshots the notifier registry.
// Registration belongs to application startup.
func NewDispatcher(notifiers ...Notifier) (*Dispatcher, error) {
	copied := make([]Notifier, 0, len(notifiers))

	for i, notifier := range notifiers {
		if notifier == nil {
			return nil, fmt.Errorf("notify: nil notifier at index %d", i)
		}
		copied = append(copied, notifier)
	}

	return &Dispatcher{notifiers: copied}, nil
}

// NotificationCountChange broadcasts a state-change signal.
//
// Failures in one notifier must not prevent later
// notifiers from observing the same event.
func (d *Dispatcher) NotificationCountChange(
	ctx context.Context,
	accountID int64,
) {
	if d == nil || ctx == nil || ctx.Err() != nil || accountID <= 0 {
		return
	}

	for _, notifier := range d.notifiers {
		func(n Notifier) {
			defer func() {
				if panicValue := recover(); panicValue != nil {
					log.Error(
						"notify: callback panicked: %v\n%s",
						panicValue,
						debug.Stack(),
					)
				}
			}()

			n.NotificationCountChange(ctx, accountID)
		}(notifier)
	}
}
