package config

import (
	"os"
	"testing"
	"time"
)

// setEnv points every variable Load reads at a known state. t.Setenv restores
// the previous values, and an empty string is how these helpers spell "unset"
// for everything except CORS_ALLOWED_ORIGINS.
func setEnv(t *testing.T, overrides map[string]string) {
	t.Helper()

	base := map[string]string{
		"DATABASE_URL":         "postgres://localhost:5432/test",
		"PORT":                 "",
		"CORS_ALLOWED_ORIGINS": "",
		"CLIENT_IP_MODE":       "",
		"OPS_ENABLED":          "",
		"DEMOS_ENABLED":        "",
		"SEED_ON_START":        "",
		"SPA_DIR":              "",
		"RATE_LIMIT":           "",
		"RATE_WINDOW":          "",
		"RATE_BLOCK_PERIOD":    "",
		"FAKE_SEED":            "",
	}

	for k, v := range overrides {
		base[k] = v
	}
	for k, v := range base {
		t.Setenv(k, v)
	}
}

func TestLoadDefaults(t *testing.T) {
	setEnv(t, nil)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// These are the values a fresh clone runs on, and the ones the README and
	// .env.example document. Changing one should break this test.
	if cfg.Port != "8080" {
		t.Errorf("Port = %q, want 8080", cfg.Port)
	}
	if cfg.ClientIPMode != "xff-trust-all" {
		t.Errorf("ClientIPMode = %q, want xff-trust-all (the deliberately spoofable default)", cfg.ClientIPMode)
	}
	if cfg.RateLimit != 30 {
		t.Errorf("RateLimit = %d, want 30", cfg.RateLimit)
	}
	if cfg.RateWindow != 10*time.Second {
		t.Errorf("RateWindow = %s, want 10s", cfg.RateWindow)
	}
	if cfg.RateBlockPeriod != 30*time.Second {
		t.Errorf("RateBlockPeriod = %s, want 30s", cfg.RateBlockPeriod)
	}
	if !cfg.OpsEnabled {
		t.Error("OpsEnabled = false, want true")
	}
	if !cfg.DemosEnabled {
		t.Error("DemosEnabled = false, want true")
	}
	if cfg.SeedOnStart {
		t.Error("SeedOnStart = true, want false")
	}
	if cfg.FakeSeed != 0 {
		t.Errorf("FakeSeed = %d, want 0 (gofakeit's crypto-random default)", cfg.FakeSeed)
	}
}

func TestLoadRequiresDatabaseURL(t *testing.T) {
	setEnv(t, map[string]string{"DATABASE_URL": ""})

	if _, err := Load(); err == nil {
		t.Fatal("Load() succeeded without DATABASE_URL, want an error")
	}
}

func TestLoadRejectsMalformedValues(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
	}{
		{"non-numeric rate limit", map[string]string{"RATE_LIMIT": "lots"}},
		{"zero rate limit", map[string]string{"RATE_LIMIT": "0"}},
		{"negative rate limit", map[string]string{"RATE_LIMIT": "-1"}},
		{"unparseable window", map[string]string{"RATE_WINDOW": "ten seconds"}},
		{"zero window", map[string]string{"RATE_WINDOW": "0s"}},
		{"negative window", map[string]string{"RATE_WINDOW": "-10s"}},
		{"unparseable block period", map[string]string{"RATE_BLOCK_PERIOD": "soon"}},
		{"zero block period", map[string]string{"RATE_BLOCK_PERIOD": "0s"}},
		{"negative fake seed", map[string]string{"FAKE_SEED": "-1"}},
		{"non-numeric fake seed", map[string]string{"FAKE_SEED": "today"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setEnv(t, tt.env)

			if _, err := Load(); err == nil {
				t.Errorf("Load() succeeded with %v, want an error", tt.env)
			}
		})
	}
}

func TestLoadOverrides(t *testing.T) {
	setEnv(t, map[string]string{
		"PORT":              "9000",
		"CLIENT_IP_MODE":    "remote-addr",
		"OPS_ENABLED":       "false",
		"SEED_ON_START":     "true",
		"RATE_LIMIT":        "5",
		"RATE_WINDOW":       "1m",
		"RATE_BLOCK_PERIOD": "2h",
		"FAKE_SEED":         "20260808",
	})

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Port != "9000" {
		t.Errorf("Port = %q, want 9000", cfg.Port)
	}
	if cfg.ClientIPMode != "remote-addr" {
		t.Errorf("ClientIPMode = %q, want remote-addr", cfg.ClientIPMode)
	}
	if cfg.OpsEnabled {
		t.Error("OpsEnabled = true, want false")
	}
	if !cfg.SeedOnStart {
		t.Error("SeedOnStart = false, want true")
	}
	if cfg.RateLimit != 5 {
		t.Errorf("RateLimit = %d, want 5", cfg.RateLimit)
	}
	if cfg.RateWindow != time.Minute {
		t.Errorf("RateWindow = %s, want 1m", cfg.RateWindow)
	}
	if cfg.RateBlockPeriod != 2*time.Hour {
		t.Errorf("RateBlockPeriod = %s, want 2h", cfg.RateBlockPeriod)
	}
	if cfg.FakeSeed != 20260808 {
		t.Errorf("FakeSeed = %d, want 20260808", cfg.FakeSeed)
	}
}

// A malformed bool is deliberately not an error: it falls back rather than
// refusing to boot over a typo in an optional flag.
func TestLoadUnparseableBoolFallsBack(t *testing.T) {
	setEnv(t, map[string]string{"OPS_ENABLED": "yes-please"})

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !cfg.OpsEnabled {
		t.Error("OpsEnabled = false, want the true default to survive a malformed value")
	}
}

func TestLoadCORSOrigins(t *testing.T) {
	tests := []struct {
		name string
		set  bool
		v    string
		want []string
	}{
		{
			name: "unset falls back to the Vite dev server",
			want: []string{"http://localhost:5173"},
		},
		{
			// This is how the container build disables CORS: the SPA is served
			// from the same origin, so no headers are wanted at all. An unset
			// variable and an explicitly empty one must not mean the same thing.
			name: "explicitly empty means no origins",
			set:  true,
			v:    "",
		},
		{
			name: "a list is split and trimmed",
			set:  true,
			v:    "https://a.example, https://b.example ,",
			want: []string{"https://a.example", "https://b.example"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setEnv(t, nil)

			// envStringSlice distinguishes unset from empty via os.LookupEnv, so
			// the two cases need genuinely different environments. t.Setenv can
			// only assign, but calling it first registers the cleanup that
			// restores the original value after os.Unsetenv.
			t.Setenv("CORS_ALLOWED_ORIGINS", tt.v)
			if !tt.set {
				if err := os.Unsetenv("CORS_ALLOWED_ORIGINS"); err != nil {
					t.Fatalf("os.Unsetenv: %v", err)
				}
			}

			cfg, err := Load()
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}

			if len(cfg.CORSAllowedOrigins) != len(tt.want) {
				t.Fatalf("CORSAllowedOrigins = %v, want %v", cfg.CORSAllowedOrigins, tt.want)
			}
			for i, want := range tt.want {
				if cfg.CORSAllowedOrigins[i] != want {
					t.Errorf("CORSAllowedOrigins[%d] = %q, want %q", i, cfg.CORSAllowedOrigins[i], want)
				}
			}
		})
	}
}

func TestLoadTrimsSPADir(t *testing.T) {
	// A stray trailing space in a .env file becomes a path that does not exist,
	// and the server silently falls back to the placeholder page.
	setEnv(t, map[string]string{"SPA_DIR": "  ./web/dist  "})

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.SPADir != "./web/dist" {
		t.Errorf("SPADir = %q, want %q", cfg.SPADir, "./web/dist")
	}
}
