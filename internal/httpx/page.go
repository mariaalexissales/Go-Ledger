package httpx

import (
	"net/http"
	"strconv"
)

// Page carries the normalized limit/offset for a list endpoint.
type Page struct {
	Limit  int
	Offset int
}

// ParsePage reads limit/offset from the query string, falling back to fallback
// and capping at max. Malformed or out-of-range values take the fallback rather
// than erroring: a list endpoint returning 400 for a stray query param is more
// annoying than useful.
//
// It lives here rather than in internal/api because internal/ops needs it too,
// and api already imports ops.
func ParsePage(r *http.Request, fallback, max int) Page {
	q := r.URL.Query()

	limit := fallback
	if n, err := strconv.Atoi(q.Get("limit")); err == nil && n > 0 {
		limit = min(n, max)
	}

	offset := 0
	if n, err := strconv.Atoi(q.Get("offset")); err == nil && n > 0 {
		offset = n
	}

	return Page{Limit: limit, Offset: offset}
}

// WriteListPage is WriteList with the paging values taken from a Page.
func WriteListPage[T any](w http.ResponseWriter, items []T, total int, p Page) {
	WriteList(w, items, total, p.Limit, p.Offset)
}
