// Package config centralizes environment-driven configuration so the rest of
// the application never reaches for os.Getenv directly.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds every tunable knob for the server. Values come from the
// environment. The defaults are chosen so a fresh clone runs, and so the rate
// limiter still trips easily enough to demonstrate.
type Config struct {
	DatabaseURL string
	Port        string

	// CORSAllowedOrigins is empty in container mode, where the SPA is served
	// from the same origin as the API and no CORS headers are required.
	CORSAllowedOrigins []string

	// Defaults are loose enough that ordinary browsing never trips the guard.
	// The demo scenarios read the live policy and scale their request counts to
	// it, so they stay dramatic at any setting.
	RateLimit       int
	RateWindow      time.Duration
	RateBlockPeriod time.Duration

	// ClientIPMode selects how a request's source identity is resolved. The
	// default reproduces the original, deliberately spoofable behavior; the
	// demos let you switch it and watch the difference.
	ClientIPMode string

	// OpsEnabled gates the whole /ops observability plane. It has no auth, so
	// it must be turned off anywhere that is not a local demo.
	OpsEnabled bool
	// DemosEnabled gates the scenario runner, which generates load on command.
	DemosEnabled bool
	// SeedOnStart fills an empty database before serving. Idempotent.
	SeedOnStart bool

	// FakeSeed makes the generated ledger reproducible. The recording workflow
	// sets it so re-recording the public demo yields the same 50 accounts, and
	// fixture diffs show what actually changed. Zero keeps gofakeit's
	// crypto-random default.
	FakeSeed uint64

	// SPADir serves the built console from disk. Empty falls back to the
	// bundle embedded with -tags embed_spa, if any.
	SPADir string
}

// Load reads configuration from the environment. It returns an error only for
// values that are genuinely required or malformed; everything else falls back
// to a documented default.
func Load() (*Config, error) {
	cfg := &Config{
		DatabaseURL:        os.Getenv("DATABASE_URL"),
		Port:               envString("PORT", "8080"),
		CORSAllowedOrigins: envStringSlice("CORS_ALLOWED_ORIGINS", "http://localhost:5173"),
		ClientIPMode:       envString("CLIENT_IP_MODE", "xff-trust-all"),
		OpsEnabled:         envBool("OPS_ENABLED", true),
		DemosEnabled:       envBool("DEMOS_ENABLED", true),
		SeedOnStart:        envBool("SEED_ON_START", false),
		// Trimmed because a stray trailing space is easy to introduce in a .env
		// file or a shell one-liner, and it turns into a path that does not
		// exist. The server then falls back to the placeholder page.
		SPADir: strings.TrimSpace(os.Getenv("SPA_DIR")),
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL environment variable is not set")
	}

	var err error
	if cfg.RateLimit, err = envInt("RATE_LIMIT", 30); err != nil {
		return nil, err
	}
	if cfg.RateWindow, err = envDuration("RATE_WINDOW", 10*time.Second); err != nil {
		return nil, err
	}
	if cfg.RateBlockPeriod, err = envDuration("RATE_BLOCK_PERIOD", 30*time.Second); err != nil {
		return nil, err
	}
	if cfg.FakeSeed, err = envUint("FAKE_SEED", 0); err != nil {
		return nil, err
	}

	return cfg, nil
}

func envString(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// envStringSlice splits a comma-separated list. An explicit empty value (as
// opposed to an unset variable) means "no origins", which is how container
// mode disables CORS entirely.
func envStringSlice(key, fallback string) []string {
	v, ok := os.LookupEnv(key)
	if !ok {
		v = fallback
	}

	var out []string
	for _, part := range strings.Split(v, ",") {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}

func envInt(key string, fallback int) (int, error) {
	v := os.Getenv(key)
	if v == "" {
		return fallback, nil
	}

	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer: %w", key, err)
	}
	if n < 1 {
		return 0, fmt.Errorf("%s must be at least 1, got %d", key, n)
	}
	return n, nil
}

func envUint(key string, fallback uint64) (uint64, error) {
	v := os.Getenv(key)
	if v == "" {
		return fallback, nil
	}

	n, err := strconv.ParseUint(v, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%s must be a non-negative integer: %w", key, err)
	}
	return n, nil
}

func envDuration(key string, fallback time.Duration) (time.Duration, error) {
	v := os.Getenv(key)
	if v == "" {
		return fallback, nil
	}

	d, err := time.ParseDuration(v)
	if err != nil {
		return 0, fmt.Errorf("%s must be a duration such as 10s or 1m: %w", key, err)
	}
	if d <= 0 {
		return 0, fmt.Errorf("%s must be positive, got %s", key, d)
	}
	return d, nil
}

func envBool(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}

	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}
