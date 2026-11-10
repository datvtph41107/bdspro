package queue

import (
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
