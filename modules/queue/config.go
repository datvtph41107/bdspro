package queue

import (
	"fmt"
	"strings"
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
	return Config{
		Queues:      map[string]int{QueueCritical: 10, QueueDefault: 5},
		Concurrency: 10,
	}
}

func (c Config) Validate() error {
	if c.Concurrency < 1 || len(c.Queues) == 0 {
		return fmt.Errorf("queue concurrency and weights must be positive")
	}
	for name, weight := range c.Queues {
		if strings.TrimSpace(name) == "" || weight < 1 {
			return fmt.Errorf("queue name and weight must be valid")
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
