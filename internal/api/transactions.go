package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

// createTransaction handles POST /transactions.
func (a *API) createTransaction(w http.ResponseWriter, r *http.Request) {
	var req CreateTransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// TODO: insert a new row into transactions using a.DB (account_id must
	// reference an existing account), scan the generated id/timestamp back
	// into a Transaction, then writeJSON(w, http.StatusCreated, transaction).
	writeError(w, http.StatusNotImplemented, "TODO: implement createTransaction")
}

// getTransaction handles GET /transactions/{id}.
func (a *API) getTransaction(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid transaction id")
		return
	}

	// TODO: query the transactions table for id using a.DB, scan into a
	// Transaction, handle pgx.ErrNoRows -> writeError(w, http.StatusNotFound, ...),
	// otherwise writeJSON(w, http.StatusOK, transaction).
	_ = id
	writeError(w, http.StatusNotImplemented, "TODO: implement getTransaction")
}

// deleteTransaction handles DELETE /transactions/{id}.
func (a *API) deleteTransaction(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid transaction id")
		return
	}

	// TODO: delete the row from transactions using a.DB, check rows affected
	// to distinguish "not found" from "deleted", then w.WriteHeader(http.StatusNoContent).
	_ = id
	writeError(w, http.StatusNotImplemented, "TODO: implement deleteTransaction")
}
