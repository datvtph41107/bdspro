package setting

import "testing"

func TestQueueSettings(t *testing.T) {
	cfg, err := NewConfigProviderFromData("[queue]\nCONCURRENCY = 5\nCRITICAL_WEIGHT = 3\n")
	if err != nil {
		t.Fatal(err)
	}
	if err := loadQueueFrom(cfg); err != nil {
		t.Fatal(err)
	}
	if Queue.Concurrency != 5 || Queue.CriticalWeight != 3 || Queue.DefaultWeight != 5 {
		t.Fatalf("queue policy = %+v", Queue)
	}
	for _, raw := range []string{"0", "257", "abc"} {
		provider, err := NewConfigProviderFromData("[queue]\nCONCURRENCY = " + raw + "\n")
		if err != nil {
			t.Fatal(err)
		}
		if err := loadQueueFrom(provider); err == nil {
			t.Errorf("accepted concurrency %q", raw)
		}
	}
}
