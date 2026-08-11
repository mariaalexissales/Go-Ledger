package api

import (
	"net/http"
	"strconv"

	"go-ledger/internal/httpx"
)

const (
	defaultLimit = 25
	maxLimit     = 100
)

// pageParams carries the normalized limit/offset for a list endpoint.
type pageParams struct {
	Limit  int
	Offset int
}

// parsePageParams reads limit/offset from the query string. Malformed or
// out-of-range values fall back to the defaults rather than erroring. A list
// endpoint returning 400 for a stray query param is more annoying than useful.
func parsePageParams(r *http.Request) pageParams {
	q := r.URL.Query()

	limit := defaultLimit
	if n, err := strconv.Atoi(q.Get("limit")); err == nil && n > 0 {
		limit = min(n, maxLimit)
	}

	offset := 0
	if n, err := strconv.Atoi(q.Get("offset")); err == nil && n > 0 {
		offset = n
	}

	return pageParams{Limit: limit, Offset: offset}
}

// writeList unpacks pageParams into the flat limit/offset that httpx takes, so
// list handlers do not have to destructure it at every call site.
func writeList[T any](w http.ResponseWriter, items []T, total int, p pageParams) {
	httpx.WriteList(w, items, total, p.Limit, p.Offset)
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
