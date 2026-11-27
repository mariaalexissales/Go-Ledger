package api

import (
	"context"
	"io"
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

const testIndex = `<!doctype html><script nonce="` + spa.NoncePlaceholder + `">x()</script><div id=root></div>`

var nonceInCSP = regexp.MustCompile(`script-src 'self' 'nonce-([A-Za-z0-9+/=]+)'`)

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
			"index.html":    {Data: []byte(testIndex)},
			"assets/app.js": {Data: []byte("console.log(1)")},
		}),
	}))
	t.Cleanup(srv.Close)

	return srv
}

func assertSecurityHeaders(t *testing.T, h http.Header) string {
	t.Helper()

	csp := h.Get("Content-Security-Policy")
	m := nonceInCSP.FindStringSubmatch(csp)
	if m == nil {
		t.Fatalf("Content-Security-Policy %q carries no nonce", csp)
	}
	nonce := m[1]

	want := map[string]string{
		"Content-Security-Policy":      httpx.ContentSecurityPolicy(nonce),
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
	if strings.Contains(csp, "unsafe-inline") {
		t.Errorf("Content-Security-Policy %q still allows unsafe-inline", csp)
	}

	return nonce
}

func TestSecurityHeadersOnEveryRoute(t *testing.T) {
	srv := newTestRouter(t)

	paths := []string{
		"/health",
		"/api/nope",
		"/ops/config",
		"/ops/nope",
		"/",
		"/index.html",
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

func TestIndexCarriesThePolicyNonce(t *testing.T) {
	srv := newTestRouter(t)
	seen := map[string]bool{}

	for _, p := range []string{"/", "/index.html", "/Index.HTML", "/security", "/security"} {
		resp, err := http.Get(srv.URL + p)
		if err != nil {
			t.Fatalf("GET %s: %v", p, err)
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			t.Fatal(err)
		}

		nonce := assertSecurityHeaders(t, resp.Header)
		if want := `<script nonce="` + nonce + `">`; !strings.Contains(string(body), want) {
			t.Errorf("GET %s: body %q does not contain %q", p, body, want)
		}
		if seen[nonce] {
			t.Errorf("GET %s: nonce %q was reused", p, nonce)
		}
		seen[nonce] = true
	}
}

func TestPlaceholderPageCarriesThePolicyNonce(t *testing.T) {
	srv := httptest.NewServer(httpx.SecurityHeaders(spa.PlaceholderHandler()))
	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		t.Fatal(err)
	}

	nonce := assertSecurityHeaders(t, resp.Header)
	if want := `<style nonce="` + nonce + `">`; !strings.Contains(string(body), want) {
		t.Errorf("placeholder page does not contain %q", want)
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

func TestInlineScriptsInIndexHTMLTakeTheNonce(t *testing.T) {
	for _, file := range []string{"../../web/index.html", "../spa/dist/index.html"} {
		t.Run(file, func(t *testing.T) {
			html, err := os.ReadFile(file)
			if os.IsNotExist(err) {
				t.Skip("not built")
			}
			if err != nil {
				t.Fatal(err)
			}

			inline := regexp.MustCompile(`<(script|style)(\s[^>]*)?>`).FindAllSubmatch(html, -1)
			for _, tag := range inline {
				attrs := string(tag[2])
				if regexp.MustCompile(`\ssrc=`).MatchString(attrs) {
					continue
				}
				if !strings.Contains(attrs, `nonce="`+spa.NoncePlaceholder+`"`) {
					t.Errorf("inline <%s%s> has no nonce placeholder, the policy will block it", tag[1], attrs)
				}
			}
		})
	}
}
