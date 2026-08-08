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
	go rl.cleanup()
	return rl
}

func (rl *RateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()

	if unblockTime, blocked := rl.blockedIPs[ip]; blocked {
		if now.Before(unblockTime) {
			return false
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
		rl.blockedIPs[ip] = now.Add(rl.blockPeriod)
		return false
	}

	rl.requests[ip] = append(validTimestamps, now)
	return true
}

func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(rl.window * 2)
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
