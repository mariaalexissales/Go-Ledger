// Package demo runs scripted traffic patterns against the ledger API.
package demo

import (
	"context"
	"time"
)

type Step struct {
	Seq        int    `json:"seq"`
	ElapsedMS  int64  `json:"elapsed_ms"`
	ClientIP   string `json:"client_ip"`
	Spoofed    bool   `json:"spoofed"`
	Method     string `json:"method"`
	Path       string `json:"path"`
	Status     int    `json:"status"`
	Blocked    bool   `json:"blocked"`
	RetryAfter int    `json:"retry_after_sec"`
	DurationMS int64  `json:"duration_ms"`
	Note       string `json:"note,omitempty"`
	Error      string `json:"error,omitempty"`
}

type Meta struct {
	ID                     string   `json:"id"`
	Name                   string   `json:"name"`
	Summary                string   `json:"summary"`
	Teaches                string   `json:"teaches"`
	Expect                 string   `json:"expect"`
	Tags                   []string `json:"tags"`
	EstimatedSeconds       int      `json:"estimated_seconds"`
	RequiresVulnerableMode bool     `json:"requires_vulnerable_mode"`
}

type Scenario struct {
	Meta Meta
	Run  func(ctx context.Context, c *Client) error
}

func All() []Meta {
	metas := make([]Meta, 0, len(scenarios))
	for _, s := range scenarios {
		metas = append(metas, s.Meta)
	}
	return metas
}

func get(id string) (Scenario, bool) {
	for _, s := range scenarios {
		if s.Meta.ID == id {
			return s, true
		}
	}
	return Scenario{}, false
}

type Summary struct {
	Sent        int    `json:"sent"`
	Allowed     int    `json:"allowed"`
	Blocked     int    `json:"blocked"`
	Errors      int    `json:"errors"`
	DistinctIPs int    `json:"distinct_ips"`
	DurationMS  int64  `json:"duration_ms"`
	Verdict     string `json:"verdict"`
}

type Result struct {
	ScenarioID   string    `json:"scenario_id"`
	StartedAt    time.Time `json:"started_at"`
	FinishedAt   time.Time `json:"finished_at"`
	ClientIPMode string    `json:"client_ip_mode"`
	RateLimit    int       `json:"rate_limit"`
	RateWindow   string    `json:"rate_window"`
	Steps        []Step    `json:"steps"`
	Summary      Summary   `json:"summary"`
	Teaches      string    `json:"teaches"`
	Error        string    `json:"error,omitempty"`
}
