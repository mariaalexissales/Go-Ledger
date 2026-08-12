// Layer 2 - API to create and interact with accounts and transactions
package api

import (
	"net/http"

	"go-ledger/internal/config"
	"go-ledger/internal/httpx"
	"go-ledger/internal/ops"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"
)

type API struct {
	DB *pgxpool.Pool
}

// Deps are the long-lived components the router wires together. They are
// constructed in main so their lifecycles (the recorder's worker goroutine in
// particular) can be tied to the process, not to the router.
type Deps struct {
	Cfg     *config.Config
	Pool    *pgxpool.Pool
	Guard   *ops.SecurityGuard
	Console *ops.Console
	// Demos is mounted under /ops/demos when non-nil.
	Demos http.Handler
	// SPA handles any path the API does not claim. Unknown /api and /ops paths
	// still return JSON 404s because those sub-routers set their own NotFound
	// handlers before being mounted.
	SPA http.Handler
}

// NewRouter assembles the HTTP surface.
func NewRouter(d Deps) http.Handler {
	a := &API{DB: d.Pool}
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)

	// Deliberately NOT middleware.RealIP: it would rewrite RemoteAddr from
	// X-Forwarded-For, turning remote-addr into xff-trust-all and making
	// CLIENT_IP_MODE meaningless. ops.Resolver owns that decision.
	//
	// Nor middleware.Logger: it wraps the ResponseWriter that the SSE handler
	// reaches through http.NewResponseController.

	// Only needed in dev, where Vite serves the SPA from a different origin.
	// In container mode the SPA is same-origin and this list is empty.
	if len(d.Cfg.CORSAllowedOrigins) > 0 {
		r.Use(cors.Handler(cors.Options{
			AllowedOrigins: d.Cfg.CORSAllowedOrigins,
			AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowedHeaders: []string{"Accept", "Content-Type", "Last-Event-ID"},
			ExposedHeaders: []string{
				"Retry-After",
				"X-RateLimit-Limit",
				"X-RateLimit-Remaining",
				"X-RateLimit-Reset",
			},
			MaxAge: 300,
		}))
	}

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	r.Mount("/api", ledgerRoutes(a, d.Guard))

	if d.Cfg.OpsEnabled {
		r.Mount("/ops", opsRoutes(d.Console, d.Demos))
	}

	if d.SPA != nil {
		r.NotFound(d.SPA.ServeHTTP)
	} else {
		r.NotFound(jsonNotFound)
	}

	return r
}

// ledgerRoutes is the guarded business API.
func ledgerRoutes(a *API, guard *ops.SecurityGuard) http.Handler {
	r := chi.NewRouter()

	// Set before any mounting so chi does not inherit a parent handler. Without
	// this, the SPA fallback added in the server layer would return HTML for
	// unknown API paths.
	r.NotFound(jsonNotFound)
	r.MethodNotAllowed(jsonMethodNotAllowed)

	r.Use(guard.SecurityLogger)

	r.Route("/accounts", func(r chi.Router) {
		r.Get("/", a.listAccounts)
		r.Post("/", a.createAccount)
		r.Get("/{id}", a.getAccount)
		r.Delete("/{id}", a.deleteAccount)
		r.Get("/{id}/transactions", a.listAccountTransactions)
	})

	r.Route("/transactions", func(r chi.Router) {
		r.Get("/", a.listTransactions)
		r.Post("/", a.createTransaction)
		r.Get("/{id}", a.getTransaction)
		r.Delete("/{id}", a.deleteTransaction)
	})

	return r
}

// opsRoutes is the unguarded observability plane.
func opsRoutes(console *ops.Console, demos http.Handler) http.Handler {
	r := chi.NewRouter()

	r.NotFound(jsonNotFound)
	r.MethodNotAllowed(jsonMethodNotAllowed)

	console.Mount(r)

	if demos != nil {
		r.Mount("/demos", demos)
	}

	return r
}

func jsonNotFound(w http.ResponseWriter, r *http.Request) {
	httpx.WriteError(w, http.StatusNotFound, "not found")
}

func jsonMethodNotAllowed(w http.ResponseWriter, r *http.Request) {
	httpx.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
}
