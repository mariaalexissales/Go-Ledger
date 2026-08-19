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
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"time"

	"go-ledger/internal/demo"
	"go-ledger/internal/httpx"
	"go-ledger/internal/ops"
)

var modes = []ops.ClientIPMode{ops.ModeTrustXFF, ops.ModeRemoteAddr}

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

type opsConfig struct {
	ClientIPMode ops.ClientIPMode `json:"client_ip_mode"`
	RateLimit    ops.Policy       `json:"rate_limit"`
	Mutable      bool             `json:"mutable"`
}

func (r *recorder) run() error {
	if err := os.MkdirAll(r.out, 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}

	var config opsConfig
	if err := r.get("/ops/config", nil, &config); err != nil {
		return fmt.Errorf("read /ops/config (is the server running?): %w", err)
	}

	if !config.Mutable {
		return fmt.Errorf("the server has DEMOS_ENABLED=false, so scenarios cannot be run or recorded")
	}

	originalMode := config.ClientIPMode
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

	var demos httpx.ListResponse[demo.Meta]
	if err := r.get("/ops/demos", nil, &demos); err != nil {
		return fmt.Errorf("read /ops/demos: %w", err)
	}
	if len(demos.Data) == 0 {
		return fmt.Errorf("no demo scenarios available")
	}

	ledger, err := r.snapshotLedger()
	if err != nil {
		return err
	}

	log.Printf("Recording %d scenarios across %d client IP modes", len(demos.Data), len(modes))

	for _, mode := range modes {
		if err := r.setMode(mode); err != nil {
			return fmt.Errorf("switch to %s: %w", mode, err)
		}

		for _, meta := range demos.Data {
			if meta.ID == "" {
				continue
			}

			if err := r.recordScenario(meta.ID, mode); err != nil {
				return fmt.Errorf("record %s in %s: %w", meta.ID, mode, err)
			}
		}
	}

	return r.writeIndex(config, demos.Data, ledger)
}

// fixture is one scenario captured in one client IP mode.
type fixture struct {
	ScenarioID   string           `json:"scenario_id"`
	ClientIPMode ops.ClientIPMode `json:"client_ip_mode"`
	RecordedAt   time.Time        `json:"recorded_at"`
	Result       demo.Result      `json:"result"`
	// Events are the real security_events rows the run produced, oldest first.
	Events []ops.EventDTO `json:"events"`
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

func (r *recorder) recordScenario(id string, mode ops.ClientIPMode) error {
	if err := r.post("/ops/events/reset", nil, nil); err != nil {
		return fmt.Errorf("reset events: %w", err)
	}

	var result demo.Result
	if err := r.post("/ops/demos/"+id+"/run", nil, &result); err != nil {
		return fmt.Errorf("run scenario: %w", err)
	}

	// The recorder batches inserts every 100ms, so give the last batch time to
	// land before reading the events back.
	time.Sleep(500 * time.Millisecond)

	var events httpx.ListResponse[ops.EventDTO]
	if err := r.get("/ops/events", url.Values{"limit": {"500"}}, &events); err != nil {
		return fmt.Errorf("read events: %w", err)
	}

	// The API returns newest first; replay wants chronological order.
	slices.Reverse(events.Data)

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
		id, mode, len(result.Steps), len(events.Data), status)
	return nil
}

// recordedConfig is the write shape, and deliberately narrower than opsConfig:
// `mutable` is a per-process fact that means nothing in a recording. The replay
// console depends on the omission -- see the Omit<OpsConfig, ...> on ReplayIndex
// in web/src/replay/fixtures.ts. Adding a field here changes the fixture format.
type recordedConfig struct {
	ClientIPMode ops.ClientIPMode `json:"client_ip_mode"`
	RateLimit    ops.Policy       `json:"rate_limit"`
}

// index carries everything the console needs before any scenario is run.
type index struct {
	RecordedAt time.Time          `json:"recorded_at"`
	Modes      []ops.ClientIPMode `json:"modes"`
	Config     recordedConfig     `json:"config"`
	Demos      []demo.Meta        `json:"demos"`
	// Seed data for the ledger pages, which have no backend in replay mode.
	Accounts     []json.RawMessage `json:"accounts"`
	Transactions []json.RawMessage `json:"transactions"`
}

type ledgerSnapshot struct {
	Accounts     []json.RawMessage
	Transactions []json.RawMessage
}

func (r *recorder) snapshotLedger() (ledgerSnapshot, error) {
	var snap ledgerSnapshot

	var accounts httpx.ListResponse[json.RawMessage]
	if err := r.get("/api/accounts", url.Values{"limit": {"100"}}, &accounts); err != nil {
		return snap, fmt.Errorf("snapshot accounts: %w", err)
	}

	var transactions httpx.ListResponse[json.RawMessage]
	if err := r.get("/api/transactions", url.Values{"limit": {"200"}}, &transactions); err != nil {
		return snap, fmt.Errorf("snapshot transactions: %w", err)
	}

	snap.Accounts = accounts.Data
	snap.Transactions = transactions.Data
	return snap, nil
}

func (r *recorder) writeIndex(config opsConfig, demos []demo.Meta, ledger ledgerSnapshot) error {
	idx := index{
		RecordedAt: time.Now().UTC(),
		Modes:      modes,
		Config: recordedConfig{
			ClientIPMode: config.ClientIPMode,
			RateLimit:    config.RateLimit,
		},
		Demos:        demos,
		Accounts:     ledger.Accounts,
		Transactions: ledger.Transactions,
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
		len(idx.Accounts), len(idx.Transactions), status, r.out)
	return nil
}

func (r *recorder) setMode(mode ops.ClientIPMode) error {
	return r.put("/ops/config/client-ip-mode", map[string]string{"mode": string(mode)}, nil)
}

func (r *recorder) get(path string, query url.Values, dst any) error {
	target := r.base + path
	if len(query) > 0 {
		target += "?" + query.Encode()
	}
	return r.do(http.MethodGet, target, nil, dst)
}

func (r *recorder) post(path string, body, dst any) error {
	return r.do(http.MethodPost, r.base+path, body, dst)
}

func (r *recorder) put(path string, body, dst any) error {
	return r.do(http.MethodPut, r.base+path, body, dst)
}

func (r *recorder) do(method, target string, body, dst any) error {
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(encoded)
	}

	req, err := http.NewRequestWithContext(context.Background(), method, target, reader)
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
		return fmt.Errorf("%s %s returned %d: %s", method, target, resp.StatusCode, bytes.TrimSpace(payload))
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

// meaningfullyEqual reports whether two encoded fixtures are the same once the
// volatile keys are stripped -- that is, whether a fresh recording differs from
// the committed one in anything but wall-clock stamps and loopback timings.
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
