// Package config centralizes environment-driven configuration so nothing else reaches for os.Getenv.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	DatabaseURL string
	Port        string

	CORSAllowedOrigins []string

	RateLimit       int
	RateWindow      time.Duration
	RateBlockPeriod time.Duration

	ClientIPMode string

	OpsEnabled   bool
	DemosEnabled bool
	SeedOnStart  bool

	FakeSeed uint64

	SPADir string
}

func Load() (*Config, error) {
	cfg := &Config{
		DatabaseURL:        os.Getenv("DATABASE_URL"),
		Port:               envString("PORT", "8080"),
		CORSAllowedOrigins: envStringSlice("CORS_ALLOWED_ORIGINS", "http://localhost:5173"),
		ClientIPMode:       envString("CLIENT_IP_MODE", "xff-trust-all"),
		OpsEnabled:         envBool("OPS_ENABLED", true),
		DemosEnabled:       envBool("DEMOS_ENABLED", true),
		SeedOnStart:        envBool("SEED_ON_START", false),
		SPADir:             strings.TrimSpace(os.Getenv("SPA_DIR")),
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
