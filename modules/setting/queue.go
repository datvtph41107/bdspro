package setting

import (
	"fmt"
	"strconv"
)

// Asynq v0.23 builds a priority-weighted queue-name slice on every fetch.
// Unbounded priorities can exhaust CPU/memory, and unbounded concurrency can
// launch excessive worker goroutines. These are BDSPro operational caps,
// not hard limits imposed by Asynq.
const (
	MaxQueueConcurrency = 256
	MaxQueueWeight      = 100
	MaxQueueTotalWeight = 1000
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
	if cfg.Concurrency > MaxQueueConcurrency {
		return QueueSettings{}, fmt.Errorf("queue.CONCURRENCY must not exceed %d", MaxQueueConcurrency)
	}
	if cfg.CriticalWeight > MaxQueueWeight {
		return QueueSettings{}, fmt.Errorf("queue.CRITICAL_WEIGHT must not exceed %d", MaxQueueWeight)
	}
	if cfg.DefaultWeight > MaxQueueWeight {
		return QueueSettings{}, fmt.Errorf("queue.DEFAULT_WEIGHT must not exceed %d", MaxQueueWeight)
	}
	return cfg, nil
}
