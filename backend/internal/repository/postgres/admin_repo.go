package postgres

import (
	"github.com/jackc/pgx/v5/pgxpool"
)

const baseWhereTrue = "WHERE 1=1"

type AdminRepo struct {
	pool *pgxpool.Pool
}

func NewAdminRepo(pool *pgxpool.Pool) *AdminRepo {
	return &AdminRepo{pool: pool}
}
