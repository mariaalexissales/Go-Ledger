package api

import (
	"net/http"
	"strconv"

	"go-ledger/internal/httpx"

	"github.com/go-chi/chi/v5"
)

const (
	defaultLimit = 25
	maxLimit     = 100
)

// parsePageParams reads the ledger endpoints' limit/offset. The observability
// plane pages the same way with looser caps -- see Console.listEvents.
func parsePageParams(r *http.Request) httpx.Page {
	return httpx.ParsePage(r, defaultLimit, maxLimit)
}

// pathIntParam reads the {id} path parameter, writing a 400 and reporting false
// when it is not an integer.
func pathIntParam(w http.ResponseWriter, r *http.Request, noun string) (int, bool) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid "+noun+" id")
		return 0, false
	}
	return id, true
}

// optionalIntParam returns a pointer so the value can be passed straight to a
// SQL predicate of the form ($1::int IS NULL OR col = $1::int).
func optionalIntParam(r *http.Request, key string) *int {
	v := r.URL.Query().Get(key)
	if v == "" {
		return nil
	}

	n, err := strconv.Atoi(v)
	if err != nil {
		return nil
	}
	return &n
}
