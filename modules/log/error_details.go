package log

import "github.com/rs/zerolog"

// ErrorDetails enriches an existing error event without emitting another event.
// The caller controls whether to include the current application stack.
func ErrorDetails(event *zerolog.Event, err error, stack bool) *zerolog.Event {
	if !event.Enabled() {
		return event
	}
	nodes, truncated := describeError(err)
	if len(nodes) > 0 {
		event.Str("error_type", nodes[0].Type).Interface("error_chain", nodes)
	}
	if truncated {
		event.Bool("error_chain_truncated", true)
	}
	if stack {
		return StackDetails(event)
	}
	return event
}

// StackDetails is also used at the panic boundary before rethrowing.
func StackDetails(event *zerolog.Event) *zerolog.Event {
	if event.Enabled() {
		if frames := applicationStack(); len(frames) != 0 {
			event.Strs("stack_frames", frames)
		}
	}
	return event
}
