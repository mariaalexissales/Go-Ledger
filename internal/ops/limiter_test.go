package ops

import (
	"sync"
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

func TestRateLimiterSweeperDropsExpiredBlocks(t *testing.T) {
	// The sweeper runs every window*2, so a 10ms window gives it a 20ms tick.
	rl := NewRateLimiter(1, 10*time.Millisecond, 20*time.Millisecond)
	t.Cleanup(rl.Close)

	// One blocked IP that never comes back. Allow is the only other path that
	// clears a block, and it only ever clears the IP it was called with.
	rl.Allow("198.51.100.9")
	rl.Allow("198.51.100.9")

	if got := rl.blockedCount(); got != 1 {
		t.Fatalf("blockedIPs has %d entries before the sweep, want 1", got)
	}

	deadline := time.Now().Add(2 * time.Second)
	for rl.blockedCount() > 0 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}

	if got := rl.blockedCount(); got != 0 {
		t.Errorf("blockedIPs still has %d entries; the sweeper is not reclaiming expired blocks", got)
	}
}

func TestRateLimiterSweeperKeepsLiveBlocks(t *testing.T) {
	rl := NewRateLimiter(1, 10*time.Millisecond, time.Hour)
	t.Cleanup(rl.Close)

	rl.Allow("198.51.100.10")
	rl.Allow("198.51.100.10")

	// Long enough for several sweeps. The block has an hour to run, so none of
	// them may touch it.
	time.Sleep(80 * time.Millisecond)

	if got := rl.blockedCount(); got != 1 {
		t.Errorf("blockedIPs has %d entries, want the unexpired block retained", got)
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

func TestRateLimiterCloseStopsTheSweeper(t *testing.T) {
	rl := NewRateLimiter(1, 10*time.Millisecond, time.Minute)

	time.Sleep(30 * time.Millisecond)

	done := make(chan struct{})
	go func() {
		rl.Close()
		rl.Close()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Close did not return: the sweeper goroutine is still running")
	}

	select {
	case <-rl.stopped:
	default:
		t.Error("stopped channel is not closed after Close returned")
	}
}

func TestRateLimiterCloseIsConcurrencySafe(t *testing.T) {
	rl := NewRateLimiter(1, time.Minute, time.Minute)

	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rl.Close()
		}()
	}

	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("concurrent Close calls deadlocked")
	}
}
