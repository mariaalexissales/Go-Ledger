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
		at := time.Now()
		ip := sg.resolver.ClientIP(r)
		actionType := r.Method + " " + r.URL.Path

		decision := sg.limiter.Allow(ip)
		flagStatus := FlagAllowed
		if !decision.Allowed {
			flagStatus = FlagBlocked
		}

		w.Header().Set("X-RateLimit-Limit", strconv.Itoa(decision.Limit))
		w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(decision.Remaining))
		w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(decision.ResetAt.Unix(), 10))

		sw := &statusWriter{ResponseWriter: w}

		// Deferred so a panicking handler is still logged.
		defer func() {
			status := sw.status
			if status == 0 {
				status = http.StatusOK
			}
			if p := recover(); p != nil {
				sg.recorder.Record(ip, actionType, flagStatus, http.StatusInternalServerError, at)
				panic(p)
			}
			sg.recorder.Record(ip, actionType, flagStatus, status, at)
		}()

		if !decision.Allowed {
			writeRateLimited(sw, decision)
			return
		}

		next.ServeHTTP(sw, r)
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	if w.status == 0 {
		w.status = code
	}
	w.ResponseWriter.WriteHeader(code)
}

// Without Unwrap, http.NewResponseController cannot Flush through this writer.
func (w *statusWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

func writeRateLimited(w http.ResponseWriter, decision Decision) {
	retryAfter := int(math.Ceil(decision.RetryAfter(time.Now()).Seconds()))
	if retryAfter < 1 {
		retryAfter = 1
	}

	w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
	httpx.WriteError(w, http.StatusTooManyRequests, "rate limit exceeded")
}
