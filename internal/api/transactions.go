package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"

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

func validateCreateTransaction(req createTransactionRequest) (string, bool) {
	switch {
	case req.AccountID < 1:
		return "account_id must be a positive integer", false
	case !req.Amount.Valid:
		return "amount is required", false
	case req.Amount.NaN:
		return "amount must be a number", false
	case req.Amount.InfinityModifier != pgtype.Finite:
		return "amount must be finite", false
	case req.Amount.Int != nil && req.Amount.Int.Sign() == 0:
		return "amount cannot be zero", false
	}

	return "", true
}

func (a *API) listTransactions(w http.ResponseWriter, r *http.Request) {
	a.writeTransactionPage(w, r, optionalIntParam(r, "account_id"))
}

func (a *API) listAccountTransactions(w http.ResponseWriter, r *http.Request) {
	id, ok := pathIntParam(w, r, "account")
	if !ok {
		return
	}

	a.writeTransactionPage(w, r, &id)
}

func (a *API) writeTransactionPage(w http.ResponseWriter, r *http.Request, accountID *int) {
	page := parsePageParams(r)

	total := 0

	transactions, err := db.Collect(r.Context(), a.DB, `
		SELECT id, account_id, amount, timestamp, COUNT(*) OVER() AS total
		FROM transactions
		WHERE ($1::int IS NULL OR account_id = $1::int)
		ORDER BY timestamp DESC, id DESC
		LIMIT $2 OFFSET $3
	`, func(row pgx.CollectableRow) (transaction, error) {
		var txn transaction
		err := row.Scan(&txn.ID, &txn.AccountID, &txn.Amount, &txn.Timestamp, &total)
		return txn, err
	}, accountID, page.Limit, page.Offset)

	if err != nil {
		httpx.WriteServerError(w, r, "failed to list transactions", err)
		return
	}

	httpx.WriteListPage(w, transactions, total, page)
}

func (a *API) createTransaction(w http.ResponseWriter, r *http.Request) {
	var req createTransactionRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}

	if msg, ok := validateCreateTransaction(req); !ok {
		httpx.WriteError(w, http.StatusBadRequest, msg)
		return
	}

	var txn transaction

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

func (a *API) getTransaction(w http.ResponseWriter, r *http.Request) {
	id, ok := pathIntParam(w, r, "transaction")
	if !ok {
		return
	}

	var txn transaction

	err := a.DB.QueryRow(r.Context(), `
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

func (a *API) deleteTransaction(w http.ResponseWriter, r *http.Request) {
	id, ok := pathIntParam(w, r, "transaction")
	if !ok {
		return
	}

	cmdTag, err := a.DB.Exec(r.Context(), `
		DELETE FROM transactions WHERE id = $1
	`, id)

	if err != nil {
		httpx.WriteServerError(w, r, "failed to delete transaction", err)
		return
	}

	if cmdTag.RowsAffected() == 0 {
		httpx.WriteError(w, http.StatusNotFound, "transaction not found")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
