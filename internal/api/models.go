package api

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

type account struct {
	ID        int            `json:"id"`
	Name      string         `json:"name"`
	Balance   pgtype.Numeric `json:"balance"`
	CreatedAt time.Time      `json:"created_at"`
}

type transaction struct {
	ID        int            `json:"id"`
	AccountID int            `json:"account_id"`
	Amount    pgtype.Numeric `json:"amount"`
	Timestamp time.Time      `json:"timestamp"`
}

type createAccountRequest struct {
	Name string `json:"name"`
}

type createTransactionRequest struct {
	AccountID int            `json:"account_id"`
	Amount    pgtype.Numeric `json:"amount"`
}
