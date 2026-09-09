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

func parsePageParams(r *http.Request) httpx.Page {
	return httpx.ParsePage(r, defaultLimit, maxLimit)
}

func pathIntParam(w http.ResponseWriter, r *http.Request, noun string) (int, bool) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid "+noun+" id")
		return 0, false
	}
	return id, true
}

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
