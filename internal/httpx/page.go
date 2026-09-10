package httpx

import (
	"net/http"
	"strconv"
)

type Page struct {
	Limit  int
	Offset int
}

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

func WriteListPage[T any](w http.ResponseWriter, items []T, total int, p Page) {
	WriteList(w, items, total, p.Limit, p.Offset)
}
