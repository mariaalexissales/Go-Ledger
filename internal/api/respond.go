package api

import (
	"net/http"

	"go-ledger/internal/httpx"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	httpx.WriteJSON(w, status, v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	httpx.WriteError(w, status, msg)
}

func writeList[T any](w http.ResponseWriter, items []T, total int, p pageParams) {
	httpx.WriteList(w, items, total, p.Limit, p.Offset)
}
