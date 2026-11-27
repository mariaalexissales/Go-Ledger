package httpx

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"strings"
)

const PermissionsPolicy = "accelerometer=(), autoplay=(), camera=(), display-capture=(), " +
	"encrypted-media=(), fullscreen=(), geolocation=(), gyroscope=(), magnetometer=(), " +
	"microphone=(), midi=(), payment=(), picture-in-picture=(), publickey-credentials-get=(), " +
	"screen-wake-lock=(), usb=(), xr-spatial-tracking=()"

type nonceKey struct{}

func ContentSecurityPolicy(nonce string) string {
	n := "'nonce-" + nonce + "'"

	return strings.Join([]string{
		"default-src 'self'",
		"script-src 'self' " + n,
		"style-src 'self' " + n,
		"img-src 'self' data:",
		"font-src 'self' data:",
		"connect-src 'self'",
		"object-src 'none'",
		"base-uri 'self'",
		"form-action 'self'",
		"frame-ancestors 'none'",
	}, "; ")
}

func CSPNonce(ctx context.Context) string {
	nonce, _ := ctx.Value(nonceKey{}).(string)
	return nonce
}

func newNonce() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return base64.StdEncoding.EncodeToString(b)
}

func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nonce := newNonce()

		h := w.Header()
		h.Set("Content-Security-Policy", ContentSecurityPolicy(nonce))
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("Permissions-Policy", PermissionsPolicy)
		h.Set("Cross-Origin-Embedder-Policy", "require-corp")
		h.Set("Cross-Origin-Opener-Policy", "same-origin")
		h.Set("Cross-Origin-Resource-Policy", "same-origin")

		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), nonceKey{}, nonce)))
	})
}
