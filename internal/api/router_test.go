package api

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"go-ledger/internal/config"
	"go-ledger/internal/httpx"
	"go-ledger/internal/ops"
	"go-ledger/internal/spa"
)

func newTestRouter(t *testing.T) *httptest.Server {
	t.Helper()

	hub := ops.NewHub()
	guard := ops.NewSecurityGuard(
		ops.NewRateLimiter(1000, time.Minute, time.Minute),
		ops.NewRecorder(nil, hub),
		ops.NewResolver(ops.ModeRemoteAddr, ""),
	)

	srv := httptest.NewServer(NewRouter(Deps{
		Cfg:     &config.Config{OpsEnabled: true},
		Guard:   guard,
		Console: ops.NewConsole(nil, hub, guard, false),
		SPA: spa.Handler(fstest.MapFS{
			"index.html":    {Data: []byte("<!doctype html><div id=root></div>")},
			"assets/app.js": {Data: []byte("console.log(1)")},
		}),
	}))
	t.Cleanup(srv.Close)

	return srv
}

func assertSecurityHeaders(t *testing.T, h http.Header) {
	t.Helper()

	want := map[string]string{
		"Content-Security-Policy":      httpx.ContentSecurityPolicy,
		"X-Content-Type-Options":       "nosniff",
		"X-Frame-Options":              "DENY",
		"Referrer-Policy":              "no-referrer",
		"Permissions-Policy":           httpx.PermissionsPolicy,
		"Cross-Origin-Embedder-Policy": "require-corp",
		"Cross-Origin-Opener-Policy":   "same-origin",
		"Cross-Origin-Resource-Policy": "same-origin",
	}
	for name, value := range want {
		if got := h.Get(name); got != value {
			t.Errorf("%s = %q, want %q", name, got, value)
		}
	}
}

func TestSecurityHeadersOnEveryRoute(t *testing.T) {
	srv := newTestRouter(t)

	paths := []string{
		"/health",
		"/api/nope",
		"/ops/config",
		"/ops/nope",
		"/",
		"/security",
		"/assets/app.js",
	}

	for _, p := range paths {
		t.Run(p, func(t *testing.T) {
			resp, err := http.Get(srv.URL + p)
			if err != nil {
				t.Fatalf("GET %s: %v", p, err)
			}
			resp.Body.Close()

			assertSecurityHeaders(t, resp.Header)
		})
	}
}

func TestSecurityHeadersKeepTheEventStreamFlushing(t *testing.T) {
	srv := newTestRouter(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/ops/events/stream", nil)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("opening stream: %v", err)
	}
	defer resp.Body.Close()

	if ct := resp.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("Content-Type = %q, want text/event-stream", ct)
	}
	assertSecurityHeaders(t, resp.Header)
}

func TestThemeScriptHashMatchesIndexHTML(t *testing.T) {
	for _, file := range []string{"../../web/index.html", "../spa/dist/index.html"} {
		t.Run(file, func(t *testing.T) {
			html, err := os.ReadFile(file)
			if os.IsNotExist(err) {
				t.Skip("not built")
			}
			if err != nil {
				t.Fatal(err)
			}

			scripts := regexp.MustCompile(`(?s)<script>(.*?)</script>`).FindAllSubmatch(html, -1)
			if len(scripts) != 1 {
				t.Fatalf("found %d inline scripts, want 1", len(scripts))
			}

			sum := sha256.Sum256(scripts[0][1])
			want := "'sha256-" + base64.StdEncoding.EncodeToString(sum[:]) + "'"
			if httpx.ThemeScriptHash != want {
				t.Fatalf("ThemeScriptHash = %s, want %s", httpx.ThemeScriptHash, want)
			}
			if !strings.Contains(httpx.ContentSecurityPolicy, want) {
				t.Fatal("script-src does not carry the theme script hash")
			}
		})
	}
}
