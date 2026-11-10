package queue

import (
	"fmt"
	"testing"

	"github.com/datvtph41107/bdspro/modules/setting"
)

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

func TestQueueDefaultsFollowCanonicalSetting(t *testing.T) {
	want := setting.DefaultQueueSettings()
	got := DefaultConfig()
	if got.Concurrency != want.Concurrency ||
		got.Queues[QueueCritical] != want.CriticalWeight ||
		got.Queues[QueueDefault] != want.DefaultWeight {
		t.Fatalf("queue backend defaults diverged from app.ini policy: %+v vs %+v", got, want)
	}
}

func TestDirectProcessorConfigRejectsUnboundedResources(t *testing.T) {
	for _, tc := range []struct {
		name string
		cfg  Config
	}{
		{"excessive_concurrency", Config{Concurrency: 257, Queues: map[string]int{QueueCritical: 1}}},
		{"excessive_weight", Config{Concurrency: 1, Queues: map[string]int{QueueCritical: 1000000000, QueueDefault: 1}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.cfg.Validate(); err == nil {
				t.Fatalf("unsafe direct queue configuration accepted: %+v", tc.cfg)
			}
		})
	}
	manyQueues := map[string]int{}
	for i := 0; i < 11; i++ {
		manyQueues[fmt.Sprintf("queue_%d", i)] = setting.MaxQueueWeight
	}
	if err := (Config{Concurrency: 1, Queues: manyQueues}).Validate(); err == nil {
		t.Fatal("unbounded total priority expansion accepted")
	}
}

func TestDirectProcessorConfigAcceptsResourceBounds(t *testing.T) {
	cfg := Config{Concurrency: setting.MaxQueueConcurrency, Queues: map[string]int{
		QueueCritical: setting.MaxQueueWeight,
		QueueDefault:  setting.MaxQueueWeight,
	}}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("valid bounded queue rejected: %v", err)
	}
}
