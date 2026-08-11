package ops

import (
	"runtime"
	"testing"
	"time"
)

func TestRateLimiterAllowsUpToTheLimit(t *testing.T) {
	rl := NewRateLimiter(3, time.Minute, time.Minute)
	t.Cleanup(rl.Close)

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
	t.Cleanup(rl.Close)

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
	t.Cleanup(rl.Close)

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
	t.Cleanup(rl.Close)

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
	t.Cleanup(rl.Close)

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
	t.Cleanup(rl.Close)

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

// The sweeper goroutine used to have no stop path at all: the ticker was never
// stopped and the loop ran forever, so a RateLimiter could never be collected
// and every test in this file leaked one. `go test -race` does not detect
// goroutine leaks, so this counts them.
func TestRateLimiterCloseStopsTheSweeper(t *testing.T) {
	baseline := runtime.NumGoroutine()

	limiters := make([]*RateLimiter, 20)
	for i := range limiters {
		limiters[i] = NewRateLimiter(1, 10*time.Millisecond, time.Minute)
	}

	// Give them a moment to actually be scheduled, or the count below proves
	// nothing.
	time.Sleep(50 * time.Millisecond)
	if got := runtime.NumGoroutine(); got < baseline+len(limiters) {
		t.Fatalf("expected at least %d goroutines after constructing %d limiters, got %d",
			baseline+len(limiters), len(limiters), got)
	}

	for _, rl := range limiters {
		rl.Close()
		// Idempotent: a double Close must not panic on a closed channel.
		rl.Close()
	}

	// Each sweeper wakes on rl.done immediately, but the scheduler still needs a
	// moment to run them.
	deadline := time.Now().Add(2 * time.Second)
	for runtime.NumGoroutine() > baseline && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}

	if got := runtime.NumGoroutine(); got > baseline {
		t.Errorf("after closing %d limiters, goroutines are %d, want back down to %d",
			len(limiters), got, baseline)
	}
}
