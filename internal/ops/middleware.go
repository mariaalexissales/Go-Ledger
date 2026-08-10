package ops

import (
	"math"
	"net/http"
	"strconv"
	"time"

	"go-ledger/internal/httpx"
)

type SecurityGuard struct {
	limiter  *RateLimiter
	recorder *Recorder
	resolver *Resolver
}

func NewSecurityGuard(limiter *RateLimiter, recorder *Recorder, resolver *Resolver) *SecurityGuard {
	return &SecurityGuard{
		limiter:  limiter,
		recorder: recorder,
		resolver: resolver,
	}
}

func (sg *SecurityGuard) Limiter() *RateLimiter { return sg.limiter }
func (sg *SecurityGuard) Resolver() *Resolver   { return sg.resolver }
func (sg *SecurityGuard) Recorder() *Recorder   { return sg.recorder }

func (sg *SecurityGuard) SecurityLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := sg.resolver.ClientIP(r)
		actionType := r.Method + " " + r.URL.Path

		decision := sg.limiter.Allow(ip)
		flagStatus := FlagAllowed
		if !decision.Allowed {
			flagStatus = FlagBlocked
		}

		sg.recorder.Record(ip, actionType, flagStatus, time.Now())

		w.Header().Set("X-RateLimit-Limit", strconv.Itoa(decision.Limit))
		w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(decision.Remaining))
		w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(decision.ResetAt.Unix(), 10))

		if !decision.Allowed {
			writeRateLimited(w, decision)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func writeRateLimited(w http.ResponseWriter, decision Decision) {
	retryAfter := int(math.Ceil(decision.RetryAfter(time.Now()).Seconds()))
	if retryAfter < 1 {
		retryAfter = 1
	}

	w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
	httpx.WriteError(w, http.StatusTooManyRequests, "rate limit exceeded")
}
