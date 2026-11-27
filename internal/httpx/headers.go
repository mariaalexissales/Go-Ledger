package httpx

import (
	"net/http"
	"strings"
)

// ThemeScriptHash must match the inline theme script in web/index.html, or the
// browser refuses to run it and the console flashes white on every load.
const ThemeScriptHash = "'sha256-Gr9pmUMFVDPjKhM30KlZpu98wzt6eNfsIfj0Ys8zvkQ='"

var ContentSecurityPolicy = strings.Join([]string{
	"default-src 'self'",
	"script-src 'self' " + ThemeScriptHash,
	"style-src 'self' 'unsafe-inline'",
	"img-src 'self' data:",
	"font-src 'self' data:",
	"connect-src 'self'",
	"object-src 'none'",
	"base-uri 'self'",
	"form-action 'self'",
	"frame-ancestors 'none'",
}, "; ")

const PermissionsPolicy = "accelerometer=(), autoplay=(), camera=(), display-capture=(), " +
	"encrypted-media=(), fullscreen=(), geolocation=(), gyroscope=(), magnetometer=(), " +
	"microphone=(), midi=(), payment=(), picture-in-picture=(), publickey-credentials-get=(), " +
	"screen-wake-lock=(), usb=(), xr-spatial-tracking=()"

func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Content-Security-Policy", ContentSecurityPolicy)
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("Permissions-Policy", PermissionsPolicy)
		h.Set("Cross-Origin-Embedder-Policy", "require-corp")
		h.Set("Cross-Origin-Opener-Policy", "same-origin")
		h.Set("Cross-Origin-Resource-Policy", "same-origin")

		next.ServeHTTP(w, r)
	})
}
