package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
)

// GET: List accounts /accounts?limit=&offset=&q=
func (a *API) listAccounts(w http.ResponseWriter, r *http.Request) {
	page := parsePageParams(r)
	search := r.URL.Query().Get("q")

	// COUNT(*) OVER() returns the unpaginated total alongside each row, which
	// avoids a second round trip just to fill in the envelope.
	rows, err := a.DB.Query(r.Context(), `
		SELECT id, name, balance, created_at, COUNT(*) OVER() AS total
		FROM accounts
		WHERE ($1 = '' OR name ILIKE '%' || $1 || '%')
		ORDER BY id
		LIMIT $2 OFFSET $3
	`, search, page.Limit, page.Offset)

	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list accounts")
		return
	}
	defer rows.Close()

	accounts := make([]Account, 0, page.Limit)
	total := 0

	for rows.Next() {
		var acc Account
		if err := rows.Scan(&acc.ID, &acc.Name, &acc.Balance, &acc.CreatedAt, &total); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to list accounts")
			return
		}
		accounts = append(accounts, acc)
	}

	if rows.Err() != nil {
		writeError(w, http.StatusInternalServerError, "failed to list accounts")
		return
	}

	writeList(w, accounts, total, page)
}

// POST: Create Accounts
func (a *API) createAccount(w http.ResponseWriter, r *http.Request) {
	var req CreateAccountRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if strings.TrimSpace(req.Name) == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	var acc Account

	err := a.DB.QueryRow(r.Context(), `
		INSERT INTO accounts (name)
		VALUES ($1)
		RETURNING id, name, balance, created_at
	`, req.Name).Scan(&acc.ID, &acc.Name, &acc.Balance, &acc.CreatedAt)

	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create account")
		return
	}

	writeJSON(w, http.StatusCreated, acc)
}

// GET: Get Account /{id}
func (a *API) getAccount(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))

	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid account id")
		return
	}

	var acc Account

	err = a.DB.QueryRow(r.Context(), `
		SELECT id, name, balance, created_at
		FROM accounts
		WHERE id = $1
	`, id).Scan(&acc.ID, &acc.Name, &acc.Balance, &acc.CreatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "account not found")
		} else {
			writeError(w, http.StatusInternalServerError, "failed to get account")
		}
		return
	}

	writeJSON(w, http.StatusOK, acc)
}

// DELETE: Delete Account /{id}
func (a *API) deleteAccount(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))

	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid account id")
		return
	}

	cmdTag, err := a.DB.Exec(r.Context(), `
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
