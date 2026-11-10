package queue

import (
	"fmt"
	"strings"

	"github.com/datvtph41107/bdspro/modules/setting"
)

const (
	QueueCritical = "critical"
	QueueDefault  = "default"
)

// Config controls one worker process. Queue weights and concurrency are
// independent of the Redis address supplied to the Asynq client.
type Config struct {
	Queues      map[string]int
	Concurrency int
}

func DefaultConfig() Config {
	defaults := setting.DefaultQueueSettings()
	return Config{
		Queues: map[string]int{
			QueueCritical: defaults.CriticalWeight,
			QueueDefault:  defaults.DefaultWeight,
		},
		Concurrency: defaults.Concurrency,
	}
}

func (c Config) Validate() error {
	if c.Concurrency < 1 || len(c.Queues) == 0 {
		return fmt.Errorf("queue concurrency and weights must be positive")
	}
	if c.Concurrency > setting.MaxQueueConcurrency {
		return fmt.Errorf("queue concurrency must not exceed %d", setting.MaxQueueConcurrency)
	}
	totalWeight := 0
	for name, weight := range c.Queues {
		if strings.TrimSpace(name) == "" || weight < 1 {
			return fmt.Errorf("queue name and weight must be valid")
		}
		if weight > setting.MaxQueueWeight {
			return fmt.Errorf("queue %q weight must not exceed %d", name, setting.MaxQueueWeight)
		}
		totalWeight += weight
		if totalWeight > setting.MaxQueueTotalWeight {
			return fmt.Errorf("total queue weight must not exceed %d", setting.MaxQueueTotalWeight)
		}
	}
	return nil
}

func (c Config) weights() map[string]int {
	weights := make(map[string]int, len(c.Queues))
	for name, weight := range c.Queues {
		weights[name] = weight
	}
	return weights
}
