package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"go-ledger/internal/httpx"
)

// errAccountNotFound distinguishes "the referenced account does not exist" from
// "the transaction does not exist".
var errAccountNotFound = errors.New("account not found")

// withTx runs fn inside a transaction, committing on success and rolling back
// on any error.
func (a *API) withTx(ctx context.Context, fn func(tx pgx.Tx) error) error {
	tx, err := a.DB.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer tx.Rollback(ctx)

	if err := fn(tx); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GET: List transactions /transactions?limit=&offset=&account_id=
func (a *API) listTransactions(w http.ResponseWriter, r *http.Request) {
	a.writeTransactionPage(w, r, optionalIntParam(r, "account_id"))
}

// GET: List one account's transactions /accounts/{id}/transactions
func (a *API) listAccountTransactions(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid account id")
		return
	}

	a.writeTransactionPage(w, r, &id)
}

// writeTransactionPage backs both transaction list endpoints. A nil accountID
// means "no filter".
func (a *API) writeTransactionPage(w http.ResponseWriter, r *http.Request, accountID *int) {
	page := parsePageParams(r)

	rows, err := a.DB.Query(r.Context(), `
		SELECT id, account_id, amount, timestamp, COUNT(*) OVER() AS total
		FROM transactions
		WHERE ($1::int IS NULL OR account_id = $1::int)
		ORDER BY timestamp DESC, id DESC
		LIMIT $2 OFFSET $3
	`, accountID, page.Limit, page.Offset)

	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "failed to list transactions")
		return
	}
	defer rows.Close()

	transactions := make([]Transaction, 0, page.Limit)
	total := 0

	for rows.Next() {
		var txn Transaction
		if err := rows.Scan(&txn.ID, &txn.AccountID, &txn.Amount, &txn.Timestamp, &total); err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "failed to list transactions")
			return
		}
		transactions = append(transactions, txn)
	}

	if rows.Err() != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "failed to list transactions")
		return
	}

	writeList(w, transactions, total, page)
}

// POST /transactions.
func (a *API) createTransaction(w http.ResponseWriter, r *http.Request) {
	var req CreateTransactionRequest
	if !httpx.DecodeJSON(w, r, &req) {
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
			return errAccountNotFound
		}
		return nil
	})
	if err != nil {
		// A bad account_id trips the transactions.account_id foreign key on the
		// INSERT, before the balance UPDATE ever runs, so the FK violation is
		// the path that actually fires. The RowsAffected check above is the
		// backstop if the constraint is ever relaxed.
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.ForeignKeyViolation {
			err = errAccountNotFound
		}

		if errors.Is(err, errAccountNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "account not found")
			return
		}

		httpx.WriteError(w, http.StatusInternalServerError, "failed to create transaction")
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, txn)
}

// GET /transactions/{id}.
func (a *API) getTransaction(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))

	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid transaction id")
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
			httpx.WriteError(w, http.StatusNotFound, "transaction not found")
		} else {
			httpx.WriteError(w, http.StatusInternalServerError, "failed to get transaction")
		}
		return
	}

	httpx.WriteJSON(w, http.StatusOK, txn)
}

// DELETE /transactions/{id}.
func (a *API) deleteTransaction(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid transaction id")
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
			httpx.WriteError(w, http.StatusNotFound, "transaction not found")
		} else {
			httpx.WriteError(w, http.StatusInternalServerError, "failed to delete transaction")
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
