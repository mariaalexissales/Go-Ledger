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
	"github.com/jackc/pgx/v5/pgtype"

	"go-ledger/internal/db"
	"go-ledger/internal/httpx"
)

var errAccountNotFound = errors.New("account not found")

func (a *API) withTx(ctx context.Context, fn func(ctx context.Context, tx pgx.Tx) error) error {
	tx, err := a.DB.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer tx.Rollback(context.WithoutCancel(ctx))

	if err := fn(ctx, tx); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// validateCreateTransaction rejects bodies the database would reject anyway,
// but with a 400 instead of a 500. An absent or null amount leaves the
// pgtype.Numeric zero-valued, which pgx encodes as NULL against a NOT NULL
// column, and that surfaces as an opaque server error rather than a message
// the caller can act on.
func validateCreateTransaction(req CreateTransactionRequest) (string, bool) {
	switch {
	case req.AccountID < 1:
		return "account_id must be a positive integer", false
	case !req.Amount.Valid:
		return "amount is required", false
	case req.Amount.NaN:
		return "amount must be a number", false
	case req.Amount.InfinityModifier != pgtype.Finite:
		return "amount must be finite", false
	// Not a database constraint: NUMERIC(15,2) accepts zero happily. This is
	// here so the API agrees with the console's own form, which already refuses
	// it (see CreateTransactionDialog). Drop this case if a zero-amount
	// transaction ever becomes meaningful.
	case req.Amount.Int != nil && req.Amount.Int.Sign() == 0:
		return "amount cannot be zero", false
	}

	return "", true
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

	total := 0

	transactions, err := db.Collect(r.Context(), a.DB, `
		SELECT id, account_id, amount, timestamp, COUNT(*) OVER() AS total
		FROM transactions
		WHERE ($1::int IS NULL OR account_id = $1::int)
		ORDER BY timestamp DESC, id DESC
		LIMIT $2 OFFSET $3
	`, func(row pgx.CollectableRow) (Transaction, error) {
		var txn Transaction
		err := row.Scan(&txn.ID, &txn.AccountID, &txn.Amount, &txn.Timestamp, &total)
		return txn, err
	}, accountID, page.Limit, page.Offset)

	if err != nil {
		httpx.WriteServerError(w, r, "failed to list transactions", err)
		return
	}

	httpx.WriteListPage(w, transactions, total, page)
}

// POST /transactions.
func (a *API) createTransaction(w http.ResponseWriter, r *http.Request) {
	var req CreateTransactionRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}

	if msg, ok := validateCreateTransaction(req); !ok {
		httpx.WriteError(w, http.StatusBadRequest, msg)
		return
	}

	var txn Transaction

	err := a.withTx(r.Context(), func(ctx context.Context, tx pgx.Tx) error {
		err := tx.QueryRow(ctx, `
			INSERT INTO transactions (account_id, amount)
			VALUES ($1, $2)
			RETURNING id, account_id, amount, timestamp
		`, req.AccountID, req.Amount).Scan(&txn.ID, &txn.AccountID, &txn.Amount, &txn.Timestamp)

		if err != nil {
			return err
		}

		cmdTag, err := tx.Exec(ctx, `
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

		httpx.WriteServerError(w, r, "failed to create transaction", err)
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
			httpx.WriteServerError(w, r, "failed to get transaction", err)
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

	err = a.withTx(r.Context(), func(ctx context.Context, tx pgx.Tx) error {
		cmdTag, err := tx.Exec(ctx, `
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
			httpx.WriteServerError(w, r, "failed to delete transaction", err)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
