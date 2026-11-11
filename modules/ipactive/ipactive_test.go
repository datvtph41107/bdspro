package ipactive

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestDefaultPolicyParityAndValidation(t *testing.T) {
	policies := DefaultPolicies()
	if len(policies) != 21 {
		t.Fatalf("expected 21 existing route actions, got %d", len(policies))
	}
	for action, policy := range policies {
		if !validAction(action) || policy.Validate() != nil || policy.Mode != ModeCount {
			t.Fatalf("invalid or changed default %s: %+v", action, policy)
		}
	}
	if policies[Register] != (Policy{Mode: ModeCount, Limit: 10, Window: time.Hour}) {
		t.Fatal("register policy changed")
	}
	if policies[Login] != (Policy{Mode: ModeCount, Limit: 30, Window: time.Minute}) {
		t.Fatal("login policy changed")
	}
	policies[Register] = Policy{}
	if DefaultPolicies()[Register].Limit != 10 {
		t.Fatal("default policy map was shared")
	}
}

func TestInvalidPolicies(t *testing.T) {
	for _, policy := range []Policy{
		{Mode: ModeCount, Limit: 0, Window: time.Minute},
		{Mode: ModeCount, Limit: 1, Window: 0},
		{Mode: ModeCooldown, Limit: 2, Window: time.Minute},
		{Mode: "other", Limit: 1, Window: time.Minute},
	} {
		if policy.Validate() == nil {
			t.Fatalf("unsafe policy accepted: %+v", policy)
		}
	}
	if validAction("login:admin") || validAction("") || !validAction(Login) {
		t.Fatal("unsafe action accepted")
	}
	if _, err := NewRedis(nil, DefaultPolicies()); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("expected fail-closed nil Redis: %v", err)
	}
}

func TestAdmissionMechanismFailsClosedWithoutLimiter(t *testing.T) {
	event, err := NewMiddleware(nil).Check(context.Background(), Login, "203.0.113.10")
	if event.Action != Login || !errors.Is(err, ErrUnavailable) {
		t.Fatalf("missing limiter was not rejected: event=%+v err=%v", event, err)
	}
}

func TestRequestLocalAdmissionOwner(t *testing.T) {
	m := NewMiddleware(nil)
	ctx := WithMiddleware(context.Background(), m)
	if got := FromContext(ctx); got != m {
		t.Fatal("request did not retain its own IPActive admission capability")
	}
	defer func() {
		if recover() == nil {
			t.Fatal("missing middleware capability should fail closed")
		}
	}()
	_ = FromContext(context.Background())
}
