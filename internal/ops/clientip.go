package ops

import (
	"crypto/subtle"
	"fmt"
	"net"
	"net/http"
	"sync"
)

type ClientIPMode string

// ModeTrustXFF is deliberately vulnerable: it returns X-Forwarded-For verbatim,
// so any client can invent a source IP per request and never accumulate a rate.
// See SECURITY.md before lifting this anywhere real.
const (
	ModeTrustXFF ClientIPMode = "xff-trust-all"

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

const (
	HeaderDemoClientIP = "X-Demo-Client-IP"
	HeaderDemoToken    = "X-Demo-Token"
)

type Resolver struct {
	mu   sync.RWMutex
	mode ClientIPMode

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

	return remoteHost(req)
}

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

	if ip := net.ParseIP(remoteHost(req)); ip == nil || !ip.IsLoopback() {
		return ""
	}

	return claimed
}

func remoteHost(req *http.Request) string {
	host, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		return req.RemoteAddr
	}
	return host
}
