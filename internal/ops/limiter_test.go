package ops

import (
	"testing"
	"time"
)

func TestRateLimiterAllowsUpToTheLimit(t *testing.T) {
	rl := NewRateLimiter(3, time.Minute, time.Minute)

	for i := 1; i <= 3; i++ {
		decision := rl.Allow("203.0.113.1")
		if !decision.Allowed {
			t.Fatalf("request %d was blocked, want allowed", i)
		}
		if want := 3 - i; decision.Remaining != want {
			t.Errorf("request %d: Remaining = %d, want %d", i, decision.Remaining, want)
		}
	}

	if decision := rl.Allow("203.0.113.1"); decision.Allowed {
		t.Error("request past the limit was allowed, want blocked")
	}
}

func TestRateLimiterTracksIPsIndependently(t *testing.T) {
	rl := NewRateLimiter(2, time.Minute, time.Minute)

	rl.Allow("203.0.113.1")
	rl.Allow("203.0.113.1")

	if decision := rl.Allow("203.0.113.1"); decision.Allowed {
		t.Error("first IP should be blocked")
	}
	if decision := rl.Allow("203.0.113.2"); !decision.Allowed {
		t.Error("a different IP should be unaffected, which is what low-and-slow exploits")
	}
}

func TestRateLimiterBlockExpires(t *testing.T) {
	rl := NewRateLimiter(1, 50*time.Millisecond, 50*time.Millisecond)

	if decision := rl.Allow("203.0.113.3"); !decision.Allowed {
		t.Fatal("first request should be allowed")
	}
	if decision := rl.Allow("203.0.113.3"); decision.Allowed {
		t.Fatal("second request should be blocked")
	}

	time.Sleep(80 * time.Millisecond)

	if decision := rl.Allow("203.0.113.3"); !decision.Allowed {
		t.Error("the block should have expired")
	}
}

func TestRateLimiterDecisionCarriesRetryAfter(t *testing.T) {
	rl := NewRateLimiter(1, time.Minute, 30*time.Second)

	rl.Allow("203.0.113.4")
	decision := rl.Allow("203.0.113.4")

	if decision.Allowed {
		t.Fatal("expected a block")
	}

	retryAfter := decision.RetryAfter(time.Now())
	if retryAfter <= 0 || retryAfter > 30*time.Second {
		t.Errorf("RetryAfter = %s, want a positive value no greater than the block period", retryAfter)
	}
}

func TestRateLimiterBlockedIPs(t *testing.T) {
	rl := NewRateLimiter(1, time.Minute, time.Minute)

	rl.Allow("203.0.113.5")
	rl.Allow("203.0.113.5")

	blocked := rl.BlockedIPs()
	if _, ok := blocked["203.0.113.5"]; !ok {
		t.Error("expected the offending IP in BlockedIPs()")
	}
	if len(blocked) != 1 {
		t.Errorf("BlockedIPs() has %d entries, want 1", len(blocked))
	}
}

func TestRateLimiterSetPolicyClearsState(t *testing.T) {
	rl := NewRateLimiter(1, time.Minute, time.Minute)

	rl.Allow("203.0.113.6")
	rl.Allow("203.0.113.6") // now blocked

	// A policy change should not leave IPs serving a block under the old rules.
	rl.SetPolicy(5, 10*time.Second, 10*time.Second)

	if decision := rl.Allow("203.0.113.6"); !decision.Allowed {
		t.Error("state should be cleared when the policy changes")
	}
	if policy := rl.Policy(); policy.Limit != 5 || policy.WindowText != "10s" {
		t.Errorf("Policy() = %+v, want limit 5 and a 10s window", policy)
	}
}
