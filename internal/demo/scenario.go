// Package demo runs scripted traffic patterns against the ledger API so the
// security guard can be observed reacting to them in real time.
//
// Adding a scenario is one file: declare it and call Register from an init
// function. Nothing else needs wiring. It appears in the API listing and in the
// UI automatically.
package demo

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"
)

// Step is one request the scenario made, as observed by the client.
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

// Meta is everything the UI needs to describe a scenario before running it.
type Meta struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Summary string `json:"summary"`
	// Teaches is the point of the scenario, in plain English.
	Teaches string `json:"teaches"`
	// Expect is what the operator should watch for while it runs.
	Expect           string   `json:"expect"`
	Tags             []string `json:"tags"`
	EstimatedSeconds int      `json:"estimated_seconds"`
	// RequiresVulnerableMode marks scenarios whose result only makes sense
	// while the server trusts X-Forwarded-For.
	RequiresVulnerableMode bool `json:"requires_vulnerable_mode"`
}

type Scenario struct {
	Meta Meta
	Run  func(ctx context.Context, c *Client) error
}

var (
	registryMu sync.RWMutex
	registry   = map[string]Scenario{}
)

// Register adds a scenario. Call it from init in the scenario's own file.
func Register(s Scenario) {
	registryMu.Lock()
	defer registryMu.Unlock()

	if s.Meta.ID == "" {
		panic("demo: scenario is missing an ID")
	}
	if _, exists := registry[s.Meta.ID]; exists {
		panic(fmt.Sprintf("demo: scenario %q registered twice", s.Meta.ID))
	}

	registry[s.Meta.ID] = s
}

// All returns scenario metadata in a stable order.
func All() []Meta {
	registryMu.RLock()
	defer registryMu.RUnlock()

	metas := make([]Meta, 0, len(registry))
	for _, s := range registry {
		metas = append(metas, s.Meta)
	}

	sort.Slice(metas, func(i, j int) bool { return metas[i].ID < metas[j].ID })
	return metas
}

func Get(id string) (Scenario, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()

	s, ok := registry[id]
	return s, ok
}

// Summary aggregates a completed run.
type Summary struct {
	Sent        int    `json:"sent"`
	Allowed     int    `json:"allowed"`
	Blocked     int    `json:"blocked"`
	Errors      int    `json:"errors"`
	DistinctIPs int    `json:"distinct_ips"`
	DurationMS  int64  `json:"duration_ms"`
	Verdict     string `json:"verdict"`
}

// Result is the full record of one run.
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
