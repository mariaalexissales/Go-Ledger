package demo

import (
	"context"
	"errors"
	"net/http"

	"go-ledger/internal/httpx"

	"github.com/go-chi/chi/v5"
)

// Handler exposes the scenario list and runner over HTTP. It is mounted only
// when demos are enabled.
type Handler struct {
	runner *Runner
}

func NewHandler(runner *Runner) *Handler {
	return &Handler{runner: runner}
}

func (h *Handler) Routes() http.Handler {
	r := chi.NewRouter()

	r.Get("/", h.list)
	r.Post("/{id}/run", h.run)

	return r
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	metas := All()
	httpx.WriteList(w, metas, len(metas), len(metas), 0)
}

func (h *Handler) run(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	result, err := h.runner.Run(r.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, ErrAlreadyRunning):
			httpx.WriteError(w, http.StatusConflict, err.Error())
		case errors.Is(err, context.Canceled):
			// The client went away mid-run; nothing to report to.
		default:
			httpx.WriteError(w, http.StatusNotFound, err.Error())
		}
		return
	}

	httpx.WriteJSON(w, http.StatusOK, result)
}
