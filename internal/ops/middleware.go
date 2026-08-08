package ops

import (
	"context"
	"net"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type SecurityGuard struct {
	db      *pgxpool.Pool
	limiter *RateLimiter
}

func NewSecurityGuard(db *pgxpool.Pool, limiter *RateLimiter) *SecurityGuard {
	return &SecurityGuard{
		db:      db,
		limiter: limiter,
	}
}

func (sg *SecurityGuard) SecurityLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := extractIP(r)
		actionType := r.Method + " " + r.URL.Path
		flagStatus := "ALLOWED"

		if !sg.limiter.Allow(ip) {
			flagStatus = "BLOCKED"
		}

		go sg.logSecurityEvent(ip, actionType, flagStatus)

		if flagStatus == "BLOCKED" {
			http.Error(w, "IP temporarily blocked", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (sg *SecurityGuard) logSecurityEvent(ip, actionType, flagStatus string) {
	query := `
		INSERT INTO security_events (ip, action_type, flag_status, created_at)
		VALUES ($1, $2, $3, NOW())
	`
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := sg.db.Exec(ctx, query, ip, actionType, flagStatus)
	if err != nil {
		println("Failed to insert security event:", err.Error())
	}
}

func extractIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return xff
	}

	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}
