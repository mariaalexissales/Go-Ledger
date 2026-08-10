package demo

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"go-ledger/internal/ops"
)

// maxRequestsPerRun bounds a scenario so a bug (or an over-enthusiastic new
// scenario) cannot turn the demo endpoint into a load generator.
const maxRequestsPerRun = 400

// ErrBudgetExhausted is returned once a scenario hits maxRequestsPerRun.
var ErrBudgetExhausted = errors.New("demo: request budget exhausted")

// Identity is how a scenario claims a source IP.
type Identity struct {
	IP string
	// Spoofed sends the IP as a raw X-Forwarded-For header. That is the
	// untrusted path, which the guard believes or ignores depending on the
	// client IP mode.
	Spoofed bool
}

// As claims an IP over the trusted demo channel. The guard honors it in every
// client IP mode, so scenarios using As behave identically before and after the
// X-Forwarded-For hole is closed. Use this for everything except the spoof demo.
func As(ip string) Identity { return Identity{IP: ip} }

// Spoof claims an IP the way an attacker would: by asserting X-Forwarded-For and
// hoping the server believes it.
func Spoof(ip string) Identity { return Identity{IP: ip, Spoofed: true} }

// Client is the handle a scenario uses to generate traffic. It records every
// request as a Step.
type Client struct {
	baseURL string
	token   string
	http    *http.Client
	policy  ops.Policy

	started time.Time
	steps   []Step
	budget  int
}

func newClient(baseURL, token string, policy ops.Policy) *Client {
	return &Client{
		baseURL: baseURL,
		token:   token,
		policy:  policy,
		http: &http.Client{
			Timeout: 5 * time.Second,
			// Each demo request is its own logical client, so connection reuse
			// is fine but redirects are not.
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		started: time.Now(),
		budget:  maxRequestsPerRun,
	}
}

// Policy exposes the guard's live rate-limit settings so scenarios can scale
// their request counts and stay meaningful at any configured limit.
func (c *Client) Policy() ops.Policy { return c.policy }

func (c *Client) Steps() []Step { return c.steps }

// Note records an annotation in the timeline without making a request.
func (c *Client) Note(text string) {
	c.steps = append(c.steps, Step{
		Seq:       len(c.steps) + 1,
		ElapsedMS: time.Since(c.started).Milliseconds(),
		Note:      text,
	})
}

// Sleep pauses, respecting cancellation.
func (c *Client) Sleep(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func (c *Client) Get(ctx context.Context, id Identity, path string) (Step, error) {
	return c.do(ctx, id, http.MethodGet, path, nil)
}

func (c *Client) Post(ctx context.Context, id Identity, path string, body any) (Step, error) {
	return c.do(ctx, id, http.MethodPost, path, body)
}

func (c *Client) do(ctx context.Context, id Identity, method, path string, body any) (Step, error) {
	if c.budget <= 0 {
		return Step{}, ErrBudgetExhausted
	}
	c.budget--

	step := Step{
		Seq:       len(c.steps) + 1,
		ElapsedMS: time.Since(c.started).Milliseconds(),
		ClientIP:  id.IP,
		Spoofed:   id.Spoofed,
		Method:    method,
		Path:      path,
	}

	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return c.record(step, fmt.Errorf("encode body: %w", err))
		}
		reader = bytes.NewReader(encoded)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return c.record(step, err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	if id.Spoofed {
		req.Header.Set("X-Forwarded-For", id.IP)
	} else {
		req.Header.Set(ops.HeaderDemoClientIP, id.IP)
		req.Header.Set(ops.HeaderDemoToken, c.token)
	}

	start := time.Now()
	resp, err := c.http.Do(req)
	step.DurationMS = time.Since(start).Milliseconds()

	if err != nil {
		return c.record(step, err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))

	step.Status = resp.StatusCode
	step.Blocked = resp.StatusCode == http.StatusTooManyRequests
	if v := resp.Header.Get("Retry-After"); v != "" {
		step.RetryAfter, _ = strconv.Atoi(v)
	}

	return c.record(step, nil)
}

func (c *Client) record(step Step, err error) (Step, error) {
	if err != nil {
		step.Error = err.Error()
	}

	c.steps = append(c.steps, step)

	// A transport hiccup on one request should not abort the scenario. It is
	// recorded and the run continues. Cancellation must stop it immediately.
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return step, err
	}
	return step, nil
}
