package setting

import (
	"fmt"
	"strconv"
)

// QueueSettings is the worker's validated Asynq execution policy.
// Redis connectivity is owned separately by the worker's Redis manager.
type QueueSettings struct {
	Concurrency    int
	CriticalWeight int
	DefaultWeight  int
}

// DefaultQueueSettings is the single source of defaults for the worker and
// modules/queue. A missing [queue] section preserves the current behavior.
func DefaultQueueSettings() QueueSettings {
	return QueueSettings{
		Concurrency:    10,
		CriticalWeight: 10,
		DefaultWeight:  5,
	}
}

// LoadQueueSettings reads optional Asynq execution options from app.ini.
// Malformed overrides fail before any worker opens PostgreSQL or Redis.
func LoadQueueSettings(provider *ConfigProvider) (QueueSettings, error) {
	if provider == nil {
		return QueueSettings{}, fmt.Errorf("queue configuration provider is nil")
	}
	cfg := DefaultQueueSettings()
	for _, option := range []struct {
		key    string
		target *int
	}{
		{"CONCURRENCY", &cfg.Concurrency},
		{"CRITICAL_WEIGHT", &cfg.CriticalWeight},
		{"DEFAULT_WEIGHT", &cfg.DefaultWeight},
	} {
		value, defined := provider.Lookup("queue", option.key)
		if !defined {
			continue
		}
		n, err := strconv.Atoi(value)
		if err != nil || n < 1 {
			return QueueSettings{}, fmt.Errorf("queue.%s must be a positive integer", option.key)
		}
		*option.target = n
	}
	return cfg, nil
}
