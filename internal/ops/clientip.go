package ops

import (
	"crypto/subtle"
	"fmt"
	"net"
	"net/http"
	"sync"
)

// ClientIPMode controls how the guard decides who a request came from. The
// choice is the whole ballgame for an IP-based rate limiter, so it is a
// first-class, runtime-switchable setting rather than a hardcoded rule.
type ClientIPMode string

const (
	// ModeTrustXFF returns the X-Forwarded-For header verbatim. This is the
	// original behavior and it is deliberately vulnerable: any client can
	// invent a new source IP per request and never accumulate a rate. The
	// xff-spoof demo exists to show exactly this.
	ModeTrustXFF ClientIPMode = "xff-trust-all"

	// ModeRemoteAddr ignores forwarding headers entirely and uses the socket
	// peer. Correct when nothing is proxying in front of the server.
	ModeRemoteAddr ClientIPMode = "remote-addr"
)

func ParseClientIPMode(s string) (ClientIPMode, error) {
	switch ClientIPMode(s) {
	case ModeTrustXFF:
		return ModeTrustXFF, nil
	case ModeRemoteAddr:
		return ModeRemoteAddr, nil
	default:
		return "", fmt.Errorf("unknown client IP mode %q (want %q or %q)", s, ModeTrustXFF, ModeRemoteAddr)
	}
}

// Headers used by the demo runner's trusted identity channel.
const (
	HeaderDemoClientIP = "X-Demo-Client-IP"
	HeaderDemoToken    = "X-Demo-Token"
)

// Resolver maps a request to the IP the security guard should attribute it to.
type Resolver struct {
	mu   sync.RWMutex
	mode ClientIPMode

	// demoToken authorizes the X-Demo-Client-IP channel. It is generated in
	// memory at startup and never persisted, so only the in-process demo runner
	// can present it. Empty disables the channel entirely.
	demoToken string
}

func NewResolver(mode ClientIPMode, demoToken string) *Resolver {
	return &Resolver{mode: mode, demoToken: demoToken}
}

func (r *Resolver) Mode() ClientIPMode {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.mode
}

func (r *Resolver) SetMode(mode ClientIPMode) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.mode = mode
}

// ClientIP resolves the source identity for a request.
//
// The demo channel is checked first and is independent of the mode. That
// separation is the point: scenarios that need synthetic source IPs keep
// working after the X-Forwarded-For hole is closed, so the same scenario can be
// run in both modes and compared. Only the spoof scenario uses raw XFF.
func (r *Resolver) ClientIP(req *http.Request) string {
	r.mu.RLock()
	mode, token := r.mode, r.demoToken
	r.mu.RUnlock()

	if ip := demoClientIP(req, token); ip != "" {
		return ip
	}

	if mode == ModeTrustXFF {
		if xff := req.Header.Get("X-Forwarded-For"); xff != "" {
			return xff
		}
	}

	return RemoteHost(req)
}

// demoClientIP honors the trusted identity header only for a loopback caller
// presenting the in-process token.
func demoClientIP(req *http.Request, token string) string {
	if token == "" {
		return ""
	}

	claimed := req.Header.Get(HeaderDemoClientIP)
	if claimed == "" {
		return ""
	}

	presented := req.Header.Get(HeaderDemoToken)
	if subtle.ConstantTimeCompare([]byte(presented), []byte(token)) != 1 {
		return ""
	}

	if ip := net.ParseIP(RemoteHost(req)); ip == nil || !ip.IsLoopback() {
		return ""
	}

	return claimed
}

// RemoteHost strips the port from RemoteAddr.
func RemoteHost(req *http.Request) string {
	host, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		return req.RemoteAddr
	}
	return host
}
