package api

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

type Account struct {
	ID        int            `json:"id"`
	Name      string         `json:"name"`
	Balance   pgtype.Numeric `json:"balance"`
	CreatedAt time.Time      `json:"created_at"`
}

type Transaction struct {
	ID        int            `json:"id"`
	AccountID int            `json:"account_id"`
	Amount    pgtype.Numeric `json:"amount"`
	Timestamp time.Time      `json:"timestamp"`
}

type CreateAccountRequest struct {
	Name string `json:"name"`
}

type CreateTransactionRequest struct {
	AccountID int            `json:"account_id"`
	Amount    pgtype.Numeric `json:"amount"`
}
