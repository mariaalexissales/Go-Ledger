package ops

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestResolverClientIP(t *testing.T) {
	const token = "secret-token"

	tests := []struct {
		name       string
		mode       ClientIPMode
		remoteAddr string
		headers    map[string]string
		want       string
	}{
		{
			name:       "no forwarding header falls back to the socket peer",
			mode:       ModeTrustXFF,
			remoteAddr: "192.0.2.10:54321",
			want:       "192.0.2.10",
		},
		{
			name:       "trust mode believes X-Forwarded-For",
			mode:       ModeTrustXFF,
			remoteAddr: "192.0.2.10:54321",
			headers:    map[string]string{"X-Forwarded-For": "203.0.113.5"},
			want:       "203.0.113.5",
		},
		{
			name:       "hardened mode ignores X-Forwarded-For",
			mode:       ModeRemoteAddr,
			remoteAddr: "192.0.2.10:54321",
			headers:    map[string]string{"X-Forwarded-For": "203.0.113.5"},
			want:       "192.0.2.10",
		},
		{
			name:       "IPv6 peer keeps its address without the port",
			mode:       ModeRemoteAddr,
			remoteAddr: "[::1]:54321",
			want:       "::1",
		},
		{
			name:       "demo header is honored from loopback with the right token",
			mode:       ModeRemoteAddr,
			remoteAddr: "127.0.0.1:54321",
			headers: map[string]string{
				HeaderDemoClientIP: "198.51.100.7",
				HeaderDemoToken:    token,
			},
			want: "198.51.100.7",
		},
		{
			name:       "demo header is rejected without the token",
			mode:       ModeRemoteAddr,
			remoteAddr: "127.0.0.1:54321",
			headers:    map[string]string{HeaderDemoClientIP: "198.51.100.7"},
			want:       "127.0.0.1",
		},
		{
			name:       "demo header is rejected with a wrong token",
			mode:       ModeRemoteAddr,
			remoteAddr: "127.0.0.1:54321",
			headers: map[string]string{
				HeaderDemoClientIP: "198.51.100.7",
				HeaderDemoToken:    "guessed",
			},
			want: "127.0.0.1",
		},
		{
			name:       "demo header is rejected from a non-loopback peer",
			mode:       ModeRemoteAddr,
			remoteAddr: "203.0.113.99:54321",
			headers: map[string]string{
				HeaderDemoClientIP: "198.51.100.7",
				HeaderDemoToken:    token,
			},
			want: "203.0.113.99",
		},
		{
			name:       "demo header wins over a spoofed X-Forwarded-For",
			mode:       ModeTrustXFF,
			remoteAddr: "127.0.0.1:54321",
			headers: map[string]string{
				"X-Forwarded-For":  "203.0.113.5",
				HeaderDemoClientIP: "198.51.100.7",
				HeaderDemoToken:    token,
			},
			want: "198.51.100.7",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/accounts/1", nil)
			req.RemoteAddr = tc.remoteAddr
			for key, value := range tc.headers {
				req.Header.Set(key, value)
			}

			got := NewResolver(tc.mode, token).ClientIP(req)
			if got != tc.want {
				t.Errorf("ClientIP() = %q, want %q", got, tc.want)
			}
		})
	}
}

// An empty token must disable the demo channel entirely, so a build with demos
// off cannot have identities asserted at it.
func TestResolverDemoChannelDisabledWithoutToken(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/accounts/1", nil)
	req.RemoteAddr = "127.0.0.1:54321"
	req.Header.Set(HeaderDemoClientIP, "198.51.100.7")
	req.Header.Set(HeaderDemoToken, "")

	if got := NewResolver(ModeRemoteAddr, "").ClientIP(req); got != "127.0.0.1" {
		t.Errorf("ClientIP() = %q, want the socket peer", got)
	}
}

func TestParseClientIPMode(t *testing.T) {
	if _, err := ParseClientIPMode("xff-trust-all"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if _, err := ParseClientIPMode("remote-addr"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if _, err := ParseClientIPMode("nonsense"); err == nil {
		t.Error("expected an error for an unknown mode")
	}
}
