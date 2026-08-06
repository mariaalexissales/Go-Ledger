package api

import "github.com/jackc/pgx/v5/pgxpool"

type API struct {
	DB *pgxpool.Pool
}
