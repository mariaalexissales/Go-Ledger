package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
)

// POST /transactions.
func (a *API) createTransaction(w http.ResponseWriter, r *http.Request) {
	var req CreateTransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	var txn Transaction

	err := a.withTx(r.Context(), func(tx pgx.Tx) error {
		err := tx.QueryRow(r.Context(), `
			INSERT INTO transactions (account_id, amount) 
			VALUES ($1, $2) 
			RETURNING id, account_id, amount, timestamp
		`, req.AccountID, req.Amount).Scan(&txn.ID, &txn.AccountID, &txn.Amount, &txn.Timestamp)

		if err != nil {
			return err
		}

		cmdTag, err := tx.Exec(r.Context(), `
			UPDATE accounts
			SET balance = balance + $1
			WHERE id = $2
		`,
			req.Amount, req.AccountID)
		
		if err != nil {
			return err
		}

		if cmdTag.RowsAffected() == 0 {
			return pgx.ErrNoRows
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "transaction not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to create transaction")
		return
	}
	writeJSON(w, http.StatusCreated, txn)
}

// GET /transactions/{id}.
func (a *API) getTransaction(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid transaction id")
		return
	}

	var txn Transaction

	err = a.DB.QueryRow(r.Context(), `
		SELECT id, account_id, amount, timestamp
		FROM transactions
		WHERE id = $1
	`, id).Scan(&txn.ID, &txn.AccountID, &txn.Amount, &txn.Timestamp)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "transaction not found")
		} else {
			writeError(w, http.StatusInternalServerError, "failed to get transaction")
		}
		return
	}

	writeJSON(w, http.StatusOK, txn)
}

// DELETE /transactions/{id}.
func (a *API) deleteTransaction(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid transaction id")
		return
	}

	err = a.withTx(r.Context(), func(tx pgx.Tx) error {
		cmdTag, err := tx.Exec(r.Context(), `
			DELETE FROM transactions
			WHERE id = $1
		`, id)
		if err != nil {
			return err
		}
		if cmdTag.RowsAffected() == 0 {
			return pgx.ErrNoRows
		}
		return nil
	})

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "transaction not found")
		} else {
			writeError(w, http.StatusInternalServerError, "failed to delete transaction")
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
