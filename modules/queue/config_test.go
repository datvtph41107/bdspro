package queue

import "testing"

func TestDefaultQueueConfig(t *testing.T) {
	cfg := DefaultConfig()
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
	cfg.weights()[QueueCritical] = 1
	if cfg.Queues[QueueCritical] != 10 {
		t.Fatal("queue weights are not copied")
	}
}

func TestQueueConfigRejectsInvalidValues(t *testing.T) {
	for _, cfg := range []Config{
		{},
		{Concurrency: 1, Queues: map[string]int{"": 1}},
		{Concurrency: 1, Queues: map[string]int{"default": 0}},
	} {
		if err := cfg.Validate(); err == nil {
			t.Fatalf("accepted invalid config: %+v", cfg)
		}
	}
}
