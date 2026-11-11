package setting

import (
	"fmt"
	"strconv"
)

const (
	MaxQueueConcurrency = 256
	MaxQueueWeight      = 100
	MaxQueueTotalWeight = 1000
)

type QueueSettings struct {
	Concurrency    int
	CriticalWeight int
	DefaultWeight  int
}

var Queue QueueSettings

// DefaultQueueSettings is shared with modules/queue, not duplicated there.
func DefaultQueueSettings() QueueSettings {
	return QueueSettings{
		Concurrency:    10,
		CriticalWeight: 10,
		DefaultWeight:  5,
	}
}

func loadQueueFrom(rootCfg ConfigProvider) error {
	cfg := DefaultQueueSettings()
	sec := rootCfg.Section("queue")
	for _, option := range []struct {
		name   string
		target *int
	}{
		{"CONCURRENCY", &cfg.Concurrency},
		{"CRITICAL_WEIGHT", &cfg.CriticalWeight},
		{"DEFAULT_WEIGHT", &cfg.DefaultWeight},
	} {
		key := sec.Key(option.name)
		if !key.IsSet() {
			continue
		}
		n, err := strconv.Atoi(key.String())
		if err != nil || n < 1 {
			return fmt.Errorf("queue.%s must be a positive integer", option.name)
		}
		*option.target = n
	}
	if cfg.Concurrency > MaxQueueConcurrency {
		return fmt.Errorf("queue.CONCURRENCY must not exceed %d", MaxQueueConcurrency)
	}
	if cfg.CriticalWeight > MaxQueueWeight {
		return fmt.Errorf("queue.CRITICAL_WEIGHT must not exceed %d", MaxQueueWeight)
	}
	if cfg.DefaultWeight > MaxQueueWeight {
		return fmt.Errorf("queue.DEFAULT_WEIGHT must not exceed %d", MaxQueueWeight)
	}
	Queue = cfg
	return nil
}

func LoadQueueSetting() error {
	return loadQueueFrom(CfgProvider)
}
