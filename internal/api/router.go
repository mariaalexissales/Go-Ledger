// Layer 2 - API to create and interact with accounts and transactions
package api

import (
	"go-ledger/internal/ops"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func NewRouter(pool *pgxpool.Pool) http.Handler {
	a := &API{DB: pool}

	r := chi.NewRouter()
	r.Use(ops.NewSecurityGuard(pool, ops.NewRateLimiter()).SecurityLogger)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	r.Route("/accounts", func(r chi.Router) {
		r.Post("/", a.createAccount)
		r.Get("/{id}", a.getAccount)
		r.Delete("/{id}", a.deleteAccount)
	})

	r.Route("/transactions", func(r chi.Router) {
		r.Post("/", a.createTransaction)
		r.Get("/{id}", a.getTransaction)
		r.Delete("/{id}", a.deleteTransaction)
	})

	return r
}
