package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
)

// POST: Create Accounts
func (a *API) createAccount(w http.ResponseWriter, r *http.Request) {
	var req CreateAccountRequest
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	var account Account

	err := a.DB.QueryRow(r.Context(), `
		INSERT INTO accounts (name)
		VALUES ($1)
		RETURNING id, name, balance
	`, req.Name).Scan(&account.ID, &account.Name, &account.Balance)
	
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create account")
		return
	}

	writeJSON(w, http.StatusCreated, account)
}

// GET: Get Account /{id}
func (a *API) getAccount(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid account id")
		return
	}

	var account Account

	err = a.DB.QueryRow(r.Context(), `
		SELECT id, name, balance
		FROM accounts
		WHERE id = $1
	`, id).Scan(&account.ID, &account.Name, &account.Balance)
	
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "account not found")
		} else {
			writeError(w, http.StatusInternalServerError, "failed to get account")
		}
		return
	}

	writeJSON(w, http.StatusOK, account)
}

// DELETE: Delete Account /{id}
func (a *API) deleteAccount(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))

	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid account id")
		return
	}

	cmdTag, err:= a.DB.Exec(r.Context(), `
		DELETE FROM accounts WHERE id = $1
	`, id)

	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete account")
		return
	}
	
	if cmdTag.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "account not found")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
