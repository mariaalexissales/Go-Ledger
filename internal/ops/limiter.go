package ops

import (
	"sync"
	"time"
)

type RateLimiter struct {
	mu          sync.Mutex
	requests    map[string][]time.Time
	limit       int
	window      time.Duration
	blockPeriod time.Duration
	blockedIPs  map[string]time.Time

	done      chan struct{}
	stopped   chan struct{}
	closeOnce sync.Once
}

func NewRateLimiter(limit int, window, blockPeriod time.Duration) *RateLimiter {
	rl := &RateLimiter{
		requests:    make(map[string][]time.Time),
		limit:       limit,
		window:      window,
		blockPeriod: blockPeriod,
		blockedIPs:  make(map[string]time.Time),
		done:        make(chan struct{}),
		stopped:     make(chan struct{}),
	}
	go rl.cleanup(window * 2)
	return rl
}

func (rl *RateLimiter) Close() {
	rl.closeOnce.Do(func() { close(rl.done) })
	<-rl.stopped
}

type Decision struct {
	Allowed   bool
	Limit     int
	Remaining int
	ResetAt   time.Time
}

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

	validTimestamps := rl.withinWindow(rl.requests[ip], now)

	if len(validTimestamps) >= rl.limit {
		unblockTime := now.Add(rl.blockPeriod)
		rl.blockedIPs[ip] = unblockTime
		return Decision{Limit: rl.limit, ResetAt: unblockTime}
	}

	rl.requests[ip] = append(validTimestamps, now)

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

type Policy struct {
	Limit int `json:"limit"`

	WindowText      string `json:"window"`
	BlockPeriodText string `json:"block_period"`
}

func (rl *RateLimiter) Policy() Policy {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	return Policy{
		Limit:           rl.limit,
		WindowText:      rl.window.String(),
		BlockPeriodText: rl.blockPeriod.String(),
	}
}

func (rl *RateLimiter) SetPolicy(limit int, window, blockPeriod time.Duration) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.limit = limit
	rl.window = window
	rl.blockPeriod = blockPeriod
	rl.clearState()
}

func (rl *RateLimiter) Reset() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.clearState()
}

func (rl *RateLimiter) withinWindow(timestamps []time.Time, now time.Time) []time.Time {
	var valid []time.Time
	for _, t := range timestamps {
		if now.Sub(t) <= rl.window {
			valid = append(valid, t)
		}
	}
	return valid
}

func (rl *RateLimiter) clearState() {
	rl.requests = make(map[string][]time.Time)
	rl.blockedIPs = make(map[string]time.Time)
}

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

func (rl *RateLimiter) blockedCount() int {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	return len(rl.blockedIPs)
}

func (rl *RateLimiter) cleanup(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	defer close(rl.stopped)

	for {
		select {
		case <-rl.done:
			return

		case <-ticker.C:
			rl.mu.Lock()
			now := time.Now()
			for ip, timestamps := range rl.requests {
				if valid := rl.withinWindow(timestamps, now); len(valid) > 0 {
					rl.requests[ip] = valid
				} else {
					delete(rl.requests, ip)
				}
			}

			for ip, unblockTime := range rl.blockedIPs {
				if !now.Before(unblockTime) {
					delete(rl.blockedIPs, ip)
				}
			}

			next := rl.window * 2
			rl.mu.Unlock()

			ticker.Reset(next)
		}
	}
}
