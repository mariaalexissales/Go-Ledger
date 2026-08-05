// Layer 2 - API to create and interact with accounts and transactions
package api

import (
	"net/http"

	// "github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	// "go-ledger/internal/ops"
)

func NewRouter(pool *pgxpool.Pool) http.Handler {
	// TODO Layer 2: accounts + transactions endpoints, chi router, ops middleware
	return nil
}
