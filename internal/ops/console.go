package ops

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"go-ledger/internal/db"
	"go-ledger/internal/httpx"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Console is the observability plane: the read side of the security logger,
// plus the controls that make it demonstrable.
//
// It is deliberately not wrapped in SecurityLogger: the guard protects the
// ledger, and the console is the instrument watching it.
type Console struct {
	db    *pgxpool.Pool
	hub   *Hub
	guard *SecurityGuard

	// mutable enables the controls that change guard behavior at runtime. Off
	// outside demo mode.
	mutable bool
}

func NewConsole(db *pgxpool.Pool, hub *Hub, guard *SecurityGuard, mutable bool) *Console {
	return &Console{db: db, hub: hub, guard: guard, mutable: mutable}
}

// Mount registers the console's routes onto an existing router, so the caller
// can attach other handlers (the demo runner) to the same mount point.
func (c *Console) Mount(r chi.Router) {
	r.Get("/config", c.getConfig)
	r.Get("/events", c.listEvents)
	r.Get("/events/stream", c.streamEvents)
	r.Get("/stats", c.getStats)

	if c.mutable {
		r.Put("/config/client-ip-mode", c.setClientIPMode)
		r.Put("/config/limiter-policy", c.setLimiterPolicy)
		r.Post("/events/reset", c.resetEvents)
	}
}

type configResponse struct {
	ClientIPMode string `json:"client_ip_mode"`
	RateLimit    Policy `json:"rate_limit"`
	// YourIP is how the guard would identify the caller right now. It makes the
	// whole IP-attribution story legible at a glance.
	YourIP        string `json:"your_ip"`
	RemoteAddr    string `json:"remote_addr"`
	Mutable       bool   `json:"mutable"`
	Subscribers   int    `json:"stream_subscribers"`
	DroppedEvents int64  `json:"dropped_events"`
	FailedEvents  int64  `json:"failed_events"`
}

func (c *Console) getConfig(w http.ResponseWriter, r *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, configResponse{
		ClientIPMode:  string(c.guard.Resolver().Mode()),
		RateLimit:     c.guard.Limiter().Policy(),
		YourIP:        c.guard.Resolver().ClientIP(r),
		RemoteAddr:    RemoteHost(r),
		Mutable:       c.mutable,
		Subscribers:   c.hub.SubscriberCount(),
		DroppedEvents: c.guard.Recorder().Dropped(),
		FailedEvents:  c.guard.Recorder().Failed(),
	})
}

func (c *Console) setClientIPMode(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Mode string `json:"mode"`
	}
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}

	mode, err := ParseClientIPMode(req.Mode)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	c.guard.Resolver().SetMode(mode)
	// Identity just changed meaning, so previously accumulated per-IP state is
	// no longer comparable.
	c.guard.Limiter().Reset()

	c.getConfig(w, r)
}

func (c *Console) setLimiterPolicy(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Limit       int    `json:"limit"`
		Window      string `json:"window"`
		BlockPeriod string `json:"block_period"`
	}
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}

	if req.Limit < 1 {
		httpx.WriteError(w, http.StatusBadRequest, "limit must be at least 1")
		return
	}

	window, err := time.ParseDuration(req.Window)
	if err != nil || window <= 0 {
		httpx.WriteError(w, http.StatusBadRequest, "window must be a positive duration such as 10s")
		return
	}

	blockPeriod, err := time.ParseDuration(req.BlockPeriod)
	if err != nil || blockPeriod <= 0 {
		httpx.WriteError(w, http.StatusBadRequest, "block_period must be a positive duration such as 1m")
		return
	}

	c.guard.Limiter().SetPolicy(req.Limit, window, blockPeriod)
	c.getConfig(w, r)
}

func (c *Console) resetEvents(w http.ResponseWriter, r *http.Request) {
	// seed.Reset deliberately leaves security_events alone, so clearing the
	// audit trail for a fresh demo needs its own path.
	if _, err := c.db.Exec(r.Context(), "TRUNCATE security_events RESTART IDENTITY"); err != nil {
		httpx.WriteServerError(w, r, "failed to reset security events", err)
		return
	}

	c.guard.Limiter().Reset()
	w.WriteHeader(http.StatusNoContent)
}

func (c *Console) listEvents(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	limit := 50
	if n, err := strconv.Atoi(q.Get("limit")); err == nil && n > 0 {
		limit = min(n, 500)
	}
	offset := 0
	if n, err := strconv.Atoi(q.Get("offset")); err == nil && n > 0 {
		offset = n
	}

	// ip_address accepts a comma-separated set so the demos page can scope the
	// feed to just the IPs a run used.
	var ips []string
	for _, part := range strings.Split(q.Get("ip_address"), ",") {
		if part = strings.TrimSpace(part); part != "" {
			ips = append(ips, part)
		}
	}
	if ips == nil {
		ips = []string{}
	}

	var since *time.Time
	if v := q.Get("since"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			since = &t
		}
	}

	total := 0

	events, err := db.Collect(r.Context(), c.db, `
		SELECT id, timestamp, ip_address, action_type, flag_status, COUNT(*) OVER() AS total
		FROM security_events
		WHERE ($1 = '' OR flag_status = $1)
		  AND (cardinality($2::varchar[]) = 0 OR ip_address = ANY($2::varchar[]))
		  AND ($3 = '' OR action_type ILIKE '%' || $3 || '%')
		  AND ($4::timestamptz IS NULL OR timestamp >= $4::timestamptz)
		ORDER BY timestamp DESC, id DESC
		LIMIT $5 OFFSET $6
	`, func(row pgx.CollectableRow) (EventDTO, error) {
		var e SecurityEvent
		if err := row.Scan(&e.ID, &e.Timestamp, &e.IPAddress, &e.ActionType, &e.FlagStatus, &total); err != nil {
			return EventDTO{}, err
		}
		return NewEventDTO(e), nil
	}, q.Get("flag_status"), ips, q.Get("action_type"), since, limit, offset)

	if err != nil {
		httpx.WriteServerError(w, r, "failed to list security events", err)
		return
	}

	httpx.WriteList(w, events, total, limit, offset)
}

type statsResponse struct {
	Window      string         `json:"window"`
	Totals      map[string]int `json:"totals"`
	DistinctIPs int            `json:"distinct_ips"`
	TopIPs      []ipStat       `json:"top_ips"`
	Buckets     []bucketStat   `json:"buckets"`
	BlockedNow  []blockedIP    `json:"blocked_now"`
}

type ipStat struct {
	IPAddress string `json:"ip_address"`
	Total     int    `json:"total"`
	Blocked   int    `json:"blocked"`
}

type bucketStat struct {
	Bucket  time.Time `json:"bucket"`
	Allowed int       `json:"allowed"`
	Blocked int       `json:"blocked"`
}

type blockedIP struct {
	IPAddress string    `json:"ip_address"`
	Until     time.Time `json:"until"`
}

func (c *Console) getStats(w http.ResponseWriter, r *http.Request) {
	window := 30 * time.Minute
	if v := r.URL.Query().Get("window"); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			window = min(d, 24*time.Hour)
		}
	}

	since := time.Now().Add(-window)
	resp := statsResponse{
		Window:     window.String(),
		Totals:     map[string]int{FlagAllowed: 0, FlagBlocked: 0},
		TopIPs:     []ipStat{},
		Buckets:    []bucketStat{},
		BlockedNow: []blockedIP{},
	}

	// Totals and distinct IP count in one pass.
	var allowed, blocked int
	err := c.db.QueryRow(r.Context(), `
		SELECT
			COUNT(*) FILTER (WHERE flag_status = $2),
			COUNT(*) FILTER (WHERE flag_status = $3),
			COUNT(DISTINCT ip_address)
		FROM security_events
		WHERE timestamp >= $1
	`, since, FlagAllowed, FlagBlocked).Scan(&allowed, &blocked, &resp.DistinctIPs)
	if err != nil {
		httpx.WriteServerError(w, r, "failed to compute security stats", fmt.Errorf("totals: %w", err))
		return
	}

	resp.Totals[FlagAllowed] = allowed
	resp.Totals[FlagBlocked] = blocked

	topIPs, err := db.Collect(r.Context(), c.db, `
		SELECT ip_address,
		       COUNT(*) AS total,
		       COUNT(*) FILTER (WHERE flag_status = $2) AS blocked
		FROM security_events
		WHERE timestamp >= $1
		GROUP BY ip_address
		ORDER BY total DESC, ip_address
		LIMIT 10
	`, func(row pgx.CollectableRow) (ipStat, error) {
		var s ipStat
		err := row.Scan(&s.IPAddress, &s.Total, &s.Blocked)
		return s, err
	}, since, FlagBlocked)
	if err != nil {
		httpx.WriteServerError(w, r, "failed to compute security stats", fmt.Errorf("top ips: %w", err))
		return
	}
	if topIPs != nil {
		resp.TopIPs = topIPs
	}

	// generate_series produces a row per bucket, so quiet minutes appear as
	// explicit zeros instead of gaps the chart would interpolate across.
	buckets, err := db.Collect(r.Context(), c.db, `
		SELECT g.bucket,
		       COUNT(e.id) FILTER (WHERE e.flag_status = $3) AS allowed,
		       COUNT(e.id) FILTER (WHERE e.flag_status = $4) AS blocked
		FROM generate_series(
			date_trunc('minute', $1::timestamptz),
			date_trunc('minute', $2::timestamptz),
			interval '1 minute'
		) AS g(bucket)
		LEFT JOIN security_events e
			ON e.timestamp >= g.bucket
			AND e.timestamp < g.bucket + interval '1 minute'
		GROUP BY g.bucket
		ORDER BY g.bucket
	`, func(row pgx.CollectableRow) (bucketStat, error) {
		var b bucketStat
		err := row.Scan(&b.Bucket, &b.Allowed, &b.Blocked)
		return b, err
	}, since, time.Now(), FlagAllowed, FlagBlocked)
	if err != nil {
		httpx.WriteServerError(w, r, "failed to compute security stats", fmt.Errorf("buckets: %w", err))
		return
	}
	if buckets != nil {
		resp.Buckets = buckets
	}

	for ip, until := range c.guard.Limiter().BlockedIPs() {
		resp.BlockedNow = append(resp.BlockedNow, blockedIP{IPAddress: ip, Until: until})
	}

	httpx.WriteJSON(w, http.StatusOK, resp)
}
