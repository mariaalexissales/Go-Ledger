package demo

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"go-ledger/internal/ops"
)

// runTimeout caps a single scenario.
const runTimeout = 90 * time.Second

// ErrAlreadyRunning is returned when a run is requested while one is in flight.
var ErrAlreadyRunning = errors.New("a demo is already running")

// Runner executes scenarios against the server's own API over loopback.
//
// Security note: the target base URL comes from configuration and the request
// paths are compile-time constants inside scenario files. Nothing about the
// destination is ever read from an HTTP request. Otherwise this would be an
// SSRF gadget rather than a demo.
type Runner struct {
	baseURL  string
	token    string
	guard    *ops.SecurityGuard
	resolver *ops.Resolver

	mu      sync.Mutex
	running bool
}

func NewRunner(baseURL, token string, guard *ops.SecurityGuard) *Runner {
	return &Runner{
		baseURL:  baseURL,
		token:    token,
		guard:    guard,
		resolver: guard.Resolver(),
	}
}

func (r *Runner) Run(ctx context.Context, id string) (*Result, error) {
	scenario, ok := Get(id)
	if !ok {
		return nil, fmt.Errorf("unknown scenario %q", id)
	}

	// One at a time: concurrent runs would interleave in the event feed and
	// make both unreadable.
	r.mu.Lock()
	if r.running {
		r.mu.Unlock()
		return nil, ErrAlreadyRunning
	}
	r.running = true
	r.mu.Unlock()

	defer func() {
		r.mu.Lock()
		r.running = false
		r.mu.Unlock()
	}()

	ctx, cancel := context.WithTimeout(ctx, runTimeout)
	defer cancel()

	policy := r.guard.Limiter().Policy()

	// Start from a clean slate so the run's numbers reflect the run alone and
	// not whatever happened to be tracked beforehand.
	r.guard.Limiter().Reset()

	client := newClient(r.baseURL, r.token, policy)
	startedAt := time.Now()

	runErr := scenario.Run(ctx, client)

	result := &Result{
		ScenarioID:   scenario.Meta.ID,
		StartedAt:    startedAt,
		FinishedAt:   time.Now(),
		ClientIPMode: string(r.resolver.Mode()),
		RateLimit:    policy.Limit,
		RateWindow:   policy.WindowText,
		Steps:        client.Steps(),
		Teaches:      scenario.Meta.Teaches,
	}

	if runErr != nil && !errors.Is(runErr, ErrBudgetExhausted) {
		result.Error = runErr.Error()
	}

	result.Summary = summarize(result, scenario.Meta)
	return result, nil
}

func summarize(result *Result, meta Meta) Summary {
	s := Summary{
		DurationMS: result.FinishedAt.Sub(result.StartedAt).Milliseconds(),
	}

	ips := make(map[string]struct{})
	for _, step := range result.Steps {
		if step.Method == "" {
			continue // an annotation, not a request
		}

		s.Sent++
		ips[step.ClientIP] = struct{}{}

		switch {
		case step.Error != "":
			s.Errors++
		case step.Blocked:
			s.Blocked++
		case step.Status > 0 && step.Status < http.StatusInternalServerError:
			s.Allowed++
		default:
			s.Errors++
		}
	}

	s.DistinctIPs = len(ips)
	s.Verdict = verdict(s, result, meta)
	return s
}

func verdict(s Summary, result *Result, meta Meta) string {
	base := fmt.Sprintf("%d requests from %d source %s, %d blocked.",
		s.Sent, s.DistinctIPs, plural(s.DistinctIPs, "IP", "IPs"), s.Blocked)

	switch {
	case meta.RequiresVulnerableMode && result.ClientIPMode == string(ops.ModeTrustXFF) && s.Blocked == 0:
		return base + " Every request claimed a different X-Forwarded-For and the guard believed all of them."
	case meta.RequiresVulnerableMode && result.ClientIPMode == string(ops.ModeRemoteAddr):
		return base + " With forwarding headers ignored, the spoofing no longer works and the guard sees one machine."
	case s.Blocked == 0 && s.Sent > result.RateLimit:
		return base + fmt.Sprintf(" That is more than the %d-request limit, yet nothing tripped: no single IP ever exceeded it.", result.RateLimit)
	case s.Blocked > 0:
		return base + fmt.Sprintf(" The limit is %d per %s, so everything after that was refused.", result.RateLimit, result.RateWindow)
	default:
		return base + " Normal traffic, nothing tripped."
	}
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}
