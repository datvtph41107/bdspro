package log

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

// prepareConsoleEvent modifies only the console's decoded event, never the
// original JSON bytes shared with the file sink.
func prepareConsoleEvent(event map[string]any) error {
	prepareConsoleCaller(event)
	if id, ok := event["request_id"].(string); ok && id != "" {
		if len(id) > 8 {
			id = id[:8]
		}
		event["rid"] = id
		delete(event, "request_id")
	}

	switch event["message"] {
	case "http.request":
		if !consoleHasFields(event, "method", "path", "status", "duration_ms") {
			break
		}
		event["message"] = fmt.Sprintf("router: completed %s %s %s %sms",
			consoleToken(event["method"]), consoleToken(event["path"]),
			consoleToken(event["status"]), consoleToken(event["duration_ms"]))
		for _, field := range []string{"component", "method", "path", "status", "duration_ms", "bytes"} {
			delete(event, field)
		}
		appendConsoleHandler(event)
	case "http.started", "http.slow", "http.failed":
		if !consoleHasFields(event, "method", "path") {
			break
		}
		kind := strings.TrimPrefix(event["message"].(string), "http.")
		message := fmt.Sprintf("router: %s %s %s", kind,
			consoleToken(event["method"]), consoleToken(event["path"]))
		if duration, ok := event["duration_ms"]; ok {
			message += " " + consoleToken(duration) + "ms"
		}
		event["message"] = message
		for _, field := range []string{"component", "method", "path", "duration_ms"} {
			delete(event, field)
		}
		appendConsoleHandler(event)
	case "db.query":
		if !consoleHasFields(event, "query", "duration_ms") {
			break
		}
		event["message"] = fmt.Sprintf("sql: %s %sms",
			consoleToken(event["query"]), consoleToken(event["duration_ms"]))
		if name, ok := event["query"].(string); ok && name != "" && name != "unnamed" {
			delete(event, "query_id")
		}
		for _, field := range []string{"component", "query", "duration_ms"} {
			delete(event, field)
		}
	case "phone.delivery":
		if !consoleHasFields(event, "channel", "accepted", "duration_ms") {
			break
		}
		event["message"] = fmt.Sprintf("delivery: %s accepted=%s %sms",
			consoleToken(event["channel"]), consoleToken(event["accepted"]), consoleToken(event["duration_ms"]))
		for _, field := range []string{"component", "channel", "accepted", "duration_ms"} {
			delete(event, field)
		}
	case "admission.denied", "auth.denied":
		event["message"] = "security: " + event["message"].(string)
		delete(event, "component")
	}
	return nil
}

func appendConsoleHandler(event map[string]any) {
	if handler, ok := event["handler"].(string); ok && handler != "" {
		event["message"] = event["message"].(string) + " @ " + consoleToken(handler)
		delete(event, "handler")
		delete(event, "handler_source")
	}
}

// Routine HTTP/SQL results do not need the same instrumentation location
// repeated on every line. Warnings/errors and other events retain the source.
func prepareConsoleCaller(event map[string]any) {
	message, _ := event["message"].(string)
	level, _ := event["level"].(string)
	// Handler metadata is more useful than the router instrumentation call site.
	_, hasHandler := event["handler"].(string)
	if strings.HasPrefix(message, "http.") && (hasHandler || message == "http.started" || message == "http.slow") {
		delete(event, "caller")
		delete(event, "caller_func")
		return
	}
	if (message == "http.request" || message == "db.query") &&
		(level == "trace" || level == "debug" || level == "info") {
		delete(event, "caller")
		delete(event, "caller_func")
		return
	}
	if caller, ok := event["caller"].(string); ok && caller != "" {
		if function, ok := event["caller_func"].(string); ok && function != "" {
			event["caller"] = caller + ":" + function + "()"
			delete(event, "caller_func")
		}
	}
}

func consoleHasFields(event map[string]any, fields ...string) bool {
	for _, field := range fields {
		if value, ok := event[field]; !ok || value == nil {
			return false
		}
	}
	return true
}

// Quote whitespace and control characters in tokens moved into the message.
func consoleToken(value any) string {
	s := fmt.Sprint(value)
	if strings.IndexFunc(s, func(r rune) bool {
		return unicode.IsSpace(r) || !unicode.IsPrint(r)
	}) >= 0 || s == "" {
		return strconv.Quote(s)
	}
	return s
}
