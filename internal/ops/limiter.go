package ops

import (
	"sync"
	"time"
)

// RateLimiter is a simple IP-based rate limiter with blocking functionality.
// It tracks recent access timestamps and applies temporary blocks when
// a certain threshold has been reached
type RateLimiter struct {
	mu          sync.Mutex
	requests    map[string][]time.Time
	limit       int
	window      time.Duration
	blockPeriod time.Duration
	blockedIPs  map[string]time.Time
}

// NewRateLimiter initializes a RateLimiter and spawns a background goroutine
// to periodically sweep andevict expired request records from memory.
// - limit: Maximum allowed requests within the time window.
// - window: Duration of the sliding tracking window.
// - blockPeriod: How long an offending IP remains blocked.
func NewRateLimiter(limit int, window time.Duration, blockPeriod time.Duration) *RateLimiter {
	rl := &RateLimiter{
		requests:    make(map[string][]time.Time),
		limit:       limit,
		window:      window,
		blockPeriod: blockPeriod,
		blockedIPs:  make(map[string]time.Time),
	}
	go rl.cleanup(window * 2)
	return rl
}

// Decision is the outcome of a single Allow check. It carries enough detail for
// the caller to populate Retry-After and the X-RateLimit-* response headers.
type Decision struct {
	Allowed   bool
	Limit     int
	Remaining int
	// ResetAt is when the caller regains capacity: the end of the block for a
	// blocked IP, or the moment the oldest tracked request leaves the window.
	ResetAt time.Time
}

// RetryAfter reports how long the caller should wait before retrying. It is
// only meaningful when the decision was a denial.
func (d Decision) RetryAfter(now time.Time) time.Duration {
	if d.ResetAt.After(now) {
		return d.ResetAt.Sub(now)
	}
	return 0
}

func (rl *RateLimiter) Allow(ip string) Decision {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()

	if unblockTime, blocked := rl.blockedIPs[ip]; blocked {
		if now.Before(unblockTime) {
			return Decision{Limit: rl.limit, ResetAt: unblockTime}
		}
		delete(rl.blockedIPs, ip)
	}

	var validTimestamps []time.Time
	for _, t := range rl.requests[ip] {
		if now.Sub(t) <= rl.window {
			validTimestamps = append(validTimestamps, t)
		}
	}

	if len(validTimestamps) >= rl.limit {
		unblockTime := now.Add(rl.blockPeriod)
		rl.blockedIPs[ip] = unblockTime
		return Decision{Limit: rl.limit, ResetAt: unblockTime}
	}

	rl.requests[ip] = append(validTimestamps, now)

	// Capacity frees up when the oldest request in the window ages out.
	resetAt := now.Add(rl.window)
	if len(validTimestamps) > 0 {
		resetAt = validTimestamps[0].Add(rl.window)
	}

	return Decision{
		Allowed:   true,
		Limit:     rl.limit,
		Remaining: rl.limit - len(validTimestamps) - 1,
		ResetAt:   resetAt,
	}
}

// Policy is the limiter's tunable configuration. It is readable and writable at
// runtime so the console can tighten or relax the guard and show the effect
// immediately, without a restart.
type Policy struct {
	Limit       int           `json:"limit"`
	Window      time.Duration `json:"-"`
	BlockPeriod time.Duration `json:"-"`

	// String forms for the wire, since a time.Duration marshals as nanoseconds.
	WindowText      string `json:"window"`
	BlockPeriodText string `json:"block_period"`
}

func (rl *RateLimiter) Policy() Policy {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	return Policy{
		Limit:           rl.limit,
		Window:          rl.window,
		BlockPeriod:     rl.blockPeriod,
		WindowText:      rl.window.String(),
		BlockPeriodText: rl.blockPeriod.String(),
	}
}

// SetPolicy replaces the limiter's configuration and clears existing state, so
// a policy change takes effect from a clean slate rather than leaving IPs
// blocked under the old rules.
func (rl *RateLimiter) SetPolicy(limit int, window, blockPeriod time.Duration) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.limit = limit
	rl.window = window
	rl.blockPeriod = blockPeriod
	rl.requests = make(map[string][]time.Time)
	rl.blockedIPs = make(map[string]time.Time)
}

// Reset clears all tracked requests and blocks without changing the policy.
func (rl *RateLimiter) Reset() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.requests = make(map[string][]time.Time)
	rl.blockedIPs = make(map[string]time.Time)
}

// BlockedIPs returns a snapshot of the IPs currently serving a block, mapped to
// the time their block expires. Used by the observability plane.
func (rl *RateLimiter) BlockedIPs() map[string]time.Time {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	out := make(map[string]time.Time)
	for ip, unblockTime := range rl.blockedIPs {
		if now.Before(unblockTime) {
			out[ip] = unblockTime
		}
	}
	return out
}

// cleanup sweeps expired request records. The interval is passed in rather than
// read from rl.window so the goroutine never touches shared state unguarded.
func (rl *RateLimiter) cleanup(interval time.Duration) {
	ticker := time.NewTicker(interval)
	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for ip, timestamps := range rl.requests {
			var valid []time.Time
			for _, t := range timestamps {
				if now.Sub(t) <= rl.window {
					valid = append(valid, t)
				}
			}
			if len(valid) > 0 {
				rl.requests[ip] = valid
			} else {
				delete(rl.requests, ip)
			}
		}
		rl.mu.Unlock()
	}
}
