package log

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Empty means all components. The reserved default label matches events without a component.
func parseComponents(value string) (map[string]struct{}, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	components := make(map[string]struct{})
	for _, part := range strings.Split(value, ",") {
		name := strings.TrimSpace(part)
		if name == "" || len(name) > 64 {
			return nil, fmt.Errorf("invalid log.console.COMPONENTS entry %q", name)
		}
		for i, c := range name {
			if !(c >= 'a' && c <= 'z' || i > 0 && (c >= '0' && c <= '9' || c == '_' || c == '-')) {
				return nil, fmt.Errorf("invalid log.console.COMPONENTS entry %q", name)
			}
		}
		components[name] = struct{}{}
	}
	return components, nil
}

// Fail open on malformed events: a console filter must not hide an unexpected payload.
func componentAllowed(components map[string]struct{}, p []byte) bool {
	if len(components) == 0 {
		return true
	}
	var event struct {
		Component string `json:"component"`
	}
	if err := json.Unmarshal(p, &event); err != nil {
		return true
	}
	if event.Component == "" {
		event.Component = "default"
	}
	_, ok := components[event.Component]
	return ok
}
