package setting

import (
	"fmt"
	"strconv"
	"time"
)

const (
	MaxQueueConcurrency = 256
	MaxQueueWeight      = 100
	MaxQueueTotalWeight = 1000
)

type QueueSettings struct {
	Concurrency     int
	CriticalWeight  int
	DefaultWeight   int
	ShutdownTimeout time.Duration
}

var Queue QueueSettings

// DefaultQueueSettings is shared with modules/queue, not duplicated there.
func DefaultQueueSettings() QueueSettings {
	return QueueSettings{
		Concurrency:     10,
		CriticalWeight:  10,
		DefaultWeight:   5,
		ShutdownTimeout: 8 * time.Second,
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

	timeoutKey := sec.Key("SHUTDOWN_TIMEOUT")

	if timeoutKey.IsSet() {
		timeout, err := time.ParseDuration(
			timeoutKey.String(),
		)
		if err != nil || timeout <= 0 {
			return fmt.Errorf(
				"queue.SHUTDOWN_TIMEOUT must be a positive duration",
			)
		}

		cfg.ShutdownTimeout = timeout
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
