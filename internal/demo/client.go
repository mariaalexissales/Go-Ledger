package demo

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"

	"go-ledger/internal/ops"
)

const maxRequestsPerRun = 400

var ErrBudgetExhausted = errors.New("demo: request budget exhausted")

type Identity struct {
	IP      string
	Spoofed bool
}

func As(ip string) Identity { return Identity{IP: ip} }

func Spoof(ip string) Identity { return Identity{IP: ip, Spoofed: true} }

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
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		started: time.Now(),
		budget:  maxRequestsPerRun,
	}
}

func (c *Client) Policy() ops.Policy { return c.policy }

func (c *Client) Steps() []Step { return c.steps }

func (c *Client) Note(text string) {
	c.steps = append(c.steps, Step{
		Seq:       len(c.steps) + 1,
		ElapsedMS: time.Since(c.started).Milliseconds(),
		Note:      text,
	})
}

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
	return c.do(ctx, id, http.MethodGet, path)
}

func (c *Client) do(ctx context.Context, id Identity, method, path string) (Step, error) {
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

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, nil)
	if err != nil {
		return c.record(step, err)
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

	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))

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

	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return step, err
	}
	return step, nil
}
