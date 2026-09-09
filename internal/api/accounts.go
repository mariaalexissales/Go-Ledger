package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"

	"go-ledger/internal/db"
	"go-ledger/internal/httpx"
)

func (a *API) listAccounts(w http.ResponseWriter, r *http.Request) {
	page := parsePageParams(r)
	search := r.URL.Query().Get("q")

	total := 0

	accounts, err := db.Collect(r.Context(), a.DB, `
		SELECT id, name, balance, created_at, COUNT(*) OVER() AS total
		FROM accounts
		WHERE ($1 = '' OR name ILIKE '%' || $1 || '%')
		ORDER BY id
		LIMIT $2 OFFSET $3
	`, func(row pgx.CollectableRow) (account, error) {
		var acc account
		err := row.Scan(&acc.ID, &acc.Name, &acc.Balance, &acc.CreatedAt, &total)
		return acc, err
	}, search, page.Limit, page.Offset)

	if err != nil {
		httpx.WriteServerError(w, r, "failed to list accounts", err)
		return
	}

	httpx.WriteListPage(w, accounts, total, page)
}

func (a *API) createAccount(w http.ResponseWriter, r *http.Request) {
	var req createAccountRequest

	if !httpx.DecodeJSON(w, r, &req) {
		return
	}

	if strings.TrimSpace(req.Name) == "" {
		httpx.WriteError(w, http.StatusBadRequest, "name is required")
		return
	}

	var acc account

	err := a.DB.QueryRow(r.Context(), `
		INSERT INTO accounts (name)
		VALUES ($1)
		RETURNING id, name, balance, created_at
	`, req.Name).Scan(&acc.ID, &acc.Name, &acc.Balance, &acc.CreatedAt)

	if err != nil {
		httpx.WriteServerError(w, r, "failed to create account", err)
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, acc)
}

func (a *API) getAccount(w http.ResponseWriter, r *http.Request) {
	id, ok := pathIntParam(w, r, "account")
	if !ok {
		return
	}

	var acc account

	err := a.DB.QueryRow(r.Context(), `
		SELECT id, name, balance, created_at
		FROM accounts
		WHERE id = $1
	`, id).Scan(&acc.ID, &acc.Name, &acc.Balance, &acc.CreatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.WriteError(w, http.StatusNotFound, "account not found")
		} else {
			httpx.WriteServerError(w, r, "failed to get account", err)
		}
		return
	}

	httpx.WriteJSON(w, http.StatusOK, acc)
}

func (a *API) deleteAccount(w http.ResponseWriter, r *http.Request) {
	id, ok := pathIntParam(w, r, "account")
	if !ok {
		return
	}

	cmdTag, err := a.DB.Exec(r.Context(), `
		DELETE FROM accounts WHERE id = $1
	`, id)

	if err != nil {
		httpx.WriteServerError(w, r, "failed to delete account", err)
		return
	}

	if cmdTag.RowsAffected() == 0 {
		httpx.WriteError(w, http.StatusNotFound, "account not found")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
