// Package api assembles the HTTP surface: the guarded ledger routes, the ops plane and the SPA fallback.
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

type Deps struct {
	Cfg     *config.Config
	Pool    *pgxpool.Pool
	Guard   *ops.SecurityGuard
	Console *ops.Console
	Demos   http.Handler
	SPA     http.Handler
}

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

func ledgerRoutes(a *API, guard *ops.SecurityGuard) http.Handler {
	r := chi.NewRouter()

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
