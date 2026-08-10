// Command record captures real demo runs from a running go-ledger server and
// writes them to web/public/replay as JSON fixtures.
//
// Usage (with the server already running):
//
//	go run ./cmd/record
//	go run ./cmd/record -base http://localhost:8080 -out web/public/replay
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

var modes = []string{"xff-trust-all", "remote-addr"}

func main() {
	base := flag.String("base", "http://localhost:8080", "base URL of a running go-ledger server")
	out := flag.String("out", filepath.Join("web", "public", "replay"), "directory to write fixtures into")
	flag.Parse()

	rec := &recorder{base: *base, out: *out, client: &http.Client{Timeout: 2 * time.Minute}}

	if err := rec.run(); err != nil {
		log.Fatalf("recording failed: %v", err)
	}
}

type recorder struct {
	base   string
	out    string
	client *http.Client
}

func (r *recorder) run() error {
	if err := os.MkdirAll(r.out, 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}

	var config map[string]any
	if err := r.get("/ops/config", nil, &config); err != nil {
		return fmt.Errorf("read /ops/config (is the server running?): %w", err)
	}

	if mutable, _ := config["mutable"].(bool); !mutable {
		return fmt.Errorf("the server has DEMOS_ENABLED=false, so scenarios cannot be run or recorded")
	}

	originalMode, _ := config["client_ip_mode"].(string)
	// Leave the dev server exactly as it was found, including on failure.
	// Otherwise the next manual test silently runs in the wrong mode.
	defer func() {
		if originalMode == "" {
			return
		}
		if err := r.setMode(originalMode); err != nil {
			log.Printf("WARNING: could not restore client IP mode to %q: %v", originalMode, err)
		} else {
			log.Printf("Restored client IP mode to %s", originalMode)
		}
	}()

	var demos listEnvelope[map[string]any]
	if err := r.get("/ops/demos", nil, &demos); err != nil {
		return fmt.Errorf("read /ops/demos: %w", err)
	}
	if len(demos.Data) == 0 {
		return fmt.Errorf("no demo scenarios registered")
	}

	log.Printf("Recording %d scenarios across %d client IP modes", len(demos.Data), len(modes))

	for _, mode := range modes {
		if err := r.setMode(mode); err != nil {
			return fmt.Errorf("switch to %s: %w", mode, err)
		}

		for _, meta := range demos.Data {
			id, _ := meta["id"].(string)
			if id == "" {
				continue
			}

			if err := r.recordScenario(id, mode); err != nil {
				return fmt.Errorf("record %s in %s: %w", id, mode, err)
			}
		}
	}

	return r.writeIndex(config, demos.Data)
}

// fixture is one scenario captured in one client IP mode.
type fixture struct {
	ScenarioID   string         `json:"scenario_id"`
	ClientIPMode string         `json:"client_ip_mode"`
	RecordedAt   time.Time      `json:"recorded_at"`
	Result       map[string]any `json:"result"`
	// Events are the real security_events rows the run produced, oldest first.
	Events []map[string]any `json:"events"`
}

// volatileKeys are fields that differ on every recording without carrying any
// meaning: wall-clock stamps and millisecond timings measured over loopback.
var volatileKeys = map[string]bool{
	"recorded_at": true,
	"started_at":  true,
	"finished_at": true,
	"timestamp":   true,
	"elapsed_ms":  true,
	"duration_ms": true,
}

func (r *recorder) recordScenario(id, mode string) error {
	if err := r.post("/ops/events/reset", nil, nil); err != nil {
		return fmt.Errorf("reset events: %w", err)
	}

	var result map[string]any
	if err := r.post("/ops/demos/"+id+"/run", nil, &result); err != nil {
		return fmt.Errorf("run scenario: %w", err)
	}

	// The recorder batches inserts every 100ms, so give the last batch time to
	// land before reading the events back.
	time.Sleep(500 * time.Millisecond)

	var events listEnvelope[map[string]any]
	if err := r.get("/ops/events", map[string]string{"limit": "500"}, &events); err != nil {
		return fmt.Errorf("read events: %w", err)
	}

	// The API returns newest first; replay wants chronological order.
	reverse(events.Data)

	f := fixture{
		ScenarioID:   id,
		ClientIPMode: mode,
		RecordedAt:   time.Now().UTC(),
		Result:       result,
		Events:       events.Data,
	}

	name := fmt.Sprintf("%s.%s.json", id, mode)
	written, err := r.writeIfChanged(name, f)
	if err != nil {
		return err
	}

	status := "unchanged"
	if written {
		status = "updated"
	}

	log.Printf("  %-14s %-14s %3d steps, %3d events  (%s)",
		id, mode, countSteps(result), len(events.Data), status)
	return nil
}

// index carries everything the console needs before any scenario is run.
type index struct {
	RecordedAt time.Time        `json:"recorded_at"`
	Modes      []string         `json:"modes"`
	Config     map[string]any   `json:"config"`
	Demos      []map[string]any `json:"demos"`
	// Seed data for the ledger pages, which have no backend in replay mode.
	Accounts     []map[string]any `json:"accounts"`
	Transactions []map[string]any `json:"transactions"`
}

func (r *recorder) writeIndex(config map[string]any, demos []map[string]any) error {
	var accounts listEnvelope[map[string]any]
	if err := r.get("/api/accounts", map[string]string{"limit": "100"}, &accounts); err != nil {
		return fmt.Errorf("snapshot accounts: %w", err)
	}

	var transactions listEnvelope[map[string]any]
	if err := r.get("/api/transactions", map[string]string{"limit": "200"}, &transactions); err != nil {
		return fmt.Errorf("snapshot transactions: %w", err)
	}

	idx := index{
		RecordedAt:   time.Now().UTC(),
		Modes:        modes,
		Config:       stripVolatile(config),
		Demos:        demos,
		Accounts:     accounts.Data,
		Transactions: transactions.Data,
	}

	written, err := r.writeIfChanged("index.json", idx)
	if err != nil {
		return err
	}

	status := "unchanged"
	if written {
		status = "updated"
	}

	log.Printf("index.json: %d accounts, %d transactions (%s) in %s",
		len(accounts.Data), len(transactions.Data), status, r.out)
	return nil
}

type listEnvelope[T any] struct {
	Data  []T `json:"data"`
	Total int `json:"total"`
}

func (r *recorder) setMode(mode string) error {
	return r.put("/ops/config/client-ip-mode", map[string]string{"mode": mode}, nil)
}

func (r *recorder) get(path string, query map[string]string, dst any) error {
	url := r.base + path
	if len(query) > 0 {
		sep := "?"
		for key, value := range query {
			url += sep + key + "=" + value
			sep = "&"
		}
	}
	return r.do(http.MethodGet, url, nil, dst)
}

func (r *recorder) post(path string, body, dst any) error {
	return r.do(http.MethodPost, r.base+path, body, dst)
}

func (r *recorder) put(path string, body, dst any) error {
	return r.do(http.MethodPut, r.base+path, body, dst)
}

func (r *recorder) do(method, url string, body, dst any) error {
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(encoded)
	}

	req, err := http.NewRequest(method, url, reader)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	payload, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("%s %s returned %d: %s", method, url, resp.StatusCode, bytes.TrimSpace(payload))
	}

	if dst == nil || len(bytes.TrimSpace(payload)) == 0 {
		return nil
	}

	return json.Unmarshal(payload, dst)
}

// writeIfChanged leaves an existing fixture alone when the new recording differs
// only in timing noise, and reports whether it wrote.
func (r *recorder) writeIfChanged(name string, v any) (bool, error) {
	encoded, err := json.MarshalIndent(v, "", " ")
	if err != nil {
		return false, err
	}
	encoded = append(encoded, '\n')

	path := filepath.Join(r.out, name)

	if existing, err := os.ReadFile(path); err == nil {
		var previous, next any
		if json.Unmarshal(existing, &previous) == nil &&
			json.Unmarshal(encoded, &next) == nil &&
			meaningfullyEqual(previous, next) {
			return false, nil
		}
	}

	return true, os.WriteFile(path, encoded, 0o644)
}

// meaningfullyEqual reports whether two encoded fixtures differ in anything
// other than timing noise.
func meaningfullyEqual(a, b any) bool {
	left, err := json.Marshal(withoutVolatile(a))
	if err != nil {
		return false
	}

	right, err := json.Marshal(withoutVolatile(b))
	if err != nil {
		return false
	}

	return bytes.Equal(left, right)
}

// withoutVolatile returns a copy with every volatile key removed, at any depth.
func withoutVolatile(v any) any {
	switch typed := v.(type) {
	case map[string]any:
		cleaned := make(map[string]any, len(typed))
		for key, value := range typed {
			if volatileKeys[key] {
				continue
			}
			cleaned[key] = withoutVolatile(value)
		}
		return cleaned

	case []any:
		cleaned := make([]any, len(typed))
		for i, value := range typed {
			cleaned[i] = withoutVolatile(value)
		}
		return cleaned

	default:
		return v
	}
}

// stripVolatile removes per-process fields from the recorded config. They are
// meaningless in a recording, since the replay transport overrides them, and
// they would change on every run.
func stripVolatile(config map[string]any) map[string]any {
	cleaned := make(map[string]any, len(config))
	for key, value := range config {
		switch key {
		case "your_ip", "remote_addr", "stream_subscribers", "dropped_events", "failed_events", "mutable":
			continue
		}
		cleaned[key] = value
	}
	return cleaned
}

func reverse[T any](items []T) {
	for i, j := 0, len(items)-1; i < j; i, j = i+1, j-1 {
		items[i], items[j] = items[j], items[i]
	}
}

func countSteps(result map[string]any) int {
	steps, _ := result["steps"].([]any)
	return len(steps)
}
